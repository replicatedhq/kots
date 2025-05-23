package upstream

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	reporting "github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"go.uber.org/zap"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

type IncompatibleAppError struct {
	KotsApplication *kotsv1beta1.Application
	License         *kotsv1beta1.License
	Message         string
}

func (e IncompatibleAppError) Error() string {
	return e.Message
}

type App struct {
	Name string
}

type Release struct {
	UpdateCursor             replicatedapp.ReplicatedCursor
	VersionLabel             string
	IsRequired               bool
	ReleaseNotes             string
	ReleasedAt               *time.Time
	ReplicatedRegistryDomain string
	ReplicatedProxyDomain    string
	ReplicatedChartNames     []string
	EmbeddedClusterArtifacts *kotsv1beta1.EmbeddedClusterArtifacts
	Manifests                map[string][]byte
}

type ChannelRelease struct {
	ChannelSequence int    `json:"channelSequence"`
	ReleaseSequence int    `json:"releaseSequence"`
	VersionLabel    string `json:"versionLabel"`
	IsRequired      bool   `json:"isRequired"`
	CreatedAt       string `json:"createdAt"`
	ReleaseNotes    string `json:"releaseNotes"`
}

func getUpdatesReplicated(fetchOptions *types.FetchOptions) (*types.UpdateCheckResult, error) {
	currentCursor := replicatedapp.ReplicatedCursor{
		ChannelID:   fetchOptions.CurrentChannelID,
		ChannelName: fetchOptions.CurrentChannelName,
		Cursor:      fetchOptions.CurrentCursor,
	}

	// A license file is required to be set for this to succeed
	if fetchOptions.License == nil {
		return nil, errors.New("No license was provided")
	}

	pendingReleases, updateCheckTime, err := listPendingChannelReleases(fetchOptions.License, fetchOptions.LastUpdateCheckAt, currentCursor, fetchOptions.ChannelChanged, fetchOptions.SortOrder, fetchOptions.ReportingInfo, fetchOptions.CurrentChannelID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list replicated app releases")
	}

	updates := []types.Update{}
	for _, pendingRelease := range pendingReleases {
		var releasedAt *time.Time
		r, err := time.Parse(time.RFC3339, pendingRelease.CreatedAt)
		if err == nil {
			releasedAt = &r
		}
		updates = append(updates, types.Update{
			ChannelID:    fetchOptions.CurrentChannelID,
			ChannelName:  fetchOptions.CurrentChannelName,
			Cursor:       strconv.Itoa(pendingRelease.ChannelSequence),
			VersionLabel: pendingRelease.VersionLabel,
			IsRequired:   pendingRelease.IsRequired,
			ReleasedAt:   releasedAt,
			ReleaseNotes: pendingRelease.ReleaseNotes,
		})
	}
	return &types.UpdateCheckResult{
		Updates:         updates,
		UpdateCheckTime: *updateCheckTime,
	}, nil
}

func downloadReplicated(
	u *url.URL,
	localPath string,
	rootDir string,
	useAppDir bool,
	license *kotsv1beta1.License,
	existingConfigValues *kotsv1beta1.ConfigValues,
	existingIdentityConfig *kotsv1beta1.IdentityConfig,
	updateCursor replicatedapp.ReplicatedCursor,
	versionLabel string,
	isRequired bool,
	replicatedRegistryDomain string,
	replicatedProxyDomain string,
	replicatedChartNames []string,
	embeddedClusterArtifacts *kotsv1beta1.EmbeddedClusterArtifacts,
	appSlug string,
	appSequence int64,
	isAirgap bool,
	airgapMetadata *kotsv1beta1.Airgap,
	registry registrytypes.RegistrySettings,
	reportingInfo *reportingtypes.ReportingInfo,
	skipCompatibilityCheck bool,
	appSelectedChannelID string,
) (*types.Upstream, error) {
	var release *Release

	if localPath != "" {
		parsedLocalRelease, err := readReplicatedAppFromLocalPath(localPath, updateCursor, versionLabel, isRequired, replicatedRegistryDomain, replicatedProxyDomain, replicatedChartNames, embeddedClusterArtifacts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read replicated app from local path")
		}

		// airgapMetadata is nil when saving initial config
		if airgapMetadata != nil {
			parsedLocalRelease.ReleaseNotes = airgapMetadata.Spec.ReleaseNotes
		}

		release = parsedLocalRelease
	} else {
		// A license file is required to be set for this to succeed
		if license == nil {
			return nil, errors.New("No license was provided")
		}

		replicatedUpstream, err := replicatedapp.ParseReplicatedURL(u)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse replicated upstream")
		}

		// Don't include version label if it's an upgrade.  It's used to get a specific version on new installs only.
		if appSequence == 0 && versionLabel != "" {
			if replicatedUpstream.VersionLabel != nil && *replicatedUpstream.VersionLabel != versionLabel {
				return nil, errors.Errorf("upstream version is %q, but requested installation version is %q", *replicatedUpstream.VersionLabel, versionLabel)
			}

			if replicatedUpstream.VersionLabel == nil && versionLabel != "" {
				replicatedUpstream.VersionLabel = &versionLabel
			}
		}

		downloadedRelease, err := downloadReplicatedApp(replicatedUpstream, license, updateCursor, reportingInfo, appSelectedChannelID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to download replicated app")
		}

		licenseData, err := replicatedapp.GetLatestLicense(license, appSelectedChannelID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get latest license")
		}
		license = licenseData.License

		release = downloadedRelease
	}

	application := findAppInRelease(release) // this function never returns nil

	if !skipCompatibilityCheck {
		isInstall := appSequence == 0
		isCompatible := kotsutil.IsKotsVersionCompatibleWithApp(*application, isInstall)
		if !isCompatible {
			return nil, IncompatibleAppError{
				KotsApplication: application,
				License:         license,
				Message:         kotsutil.GetIncompatbileKotsVersionMessage(*application, isInstall),
			}
		}
	}

	// NOTE: this currently comes from the application spec and not the channel release meta
	if release.ReleaseNotes == "" {
		release.ReleaseNotes = application.Spec.ReleaseNotes
	}

	// get channel name from license, if one was provided
	channelID, channelName := "", ""
	if license != nil {
		if appSelectedChannelID != "" {
			channel, err := kotsutil.FindChannelInLicense(appSelectedChannelID, license)
			if err != nil {
				return nil, errors.Wrap(err, "failed to find channel in license")
			}
			channelID = channel.ChannelID
			channelName = channel.ChannelName
		} else {
			channelID = license.Spec.ChannelID
			channelName = license.Spec.ChannelName
		}
	}

	if existingIdentityConfig == nil {
		var prevIdentityConfigFile string
		if useAppDir {
			prevIdentityConfigFile = filepath.Join(rootDir, application.Name, "upstream", "userdata", "identityconfig.yaml")
		} else {
			prevIdentityConfigFile = filepath.Join(rootDir, "upstream", "userdata", "identityconfig.yaml")
		}
		var err error
		existingIdentityConfig, err = findIdentityConfigInFile(prevIdentityConfigFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load existing identity config")
		}
	}

	if existingIdentityConfig != nil {
		release.Manifests["userdata/identityconfig.yaml"] = mustMarshalIdentityConfig(existingIdentityConfig)
	}

	// Find the config in the upstream and write out default values
	if existingConfigValues == nil {
		var prevConfigFile string
		if useAppDir {
			prevConfigFile = filepath.Join(rootDir, application.Name, "upstream", "userdata", "config.yaml")
		} else {
			prevConfigFile = filepath.Join(rootDir, "upstream", "userdata", "config.yaml")
		}
		var err error
		existingConfigValues, err = findConfigValuesInFile(prevConfigFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load existing config values")
		}
	}

	config := findConfigInRelease(release)

	if config != nil || existingConfigValues != nil {
		appInfo := template.ApplicationInfo{
			Slug: appSlug,
		}

		versionInfo := template.VersionInfo{
			Sequence:                 appSequence,
			Cursor:                   updateCursor.Cursor,
			ChannelName:              channelName,
			VersionLabel:             release.VersionLabel,
			IsRequired:               release.IsRequired,
			ReleaseNotes:             release.ReleaseNotes,
			IsAirgap:                 isAirgap,
			ReplicatedRegistryDomain: release.ReplicatedRegistryDomain,
			ReplicatedProxyDomain:    release.ReplicatedProxyDomain,
		}

		// If config existed and was removed from the app,
		// values will be carried over to the new version anyway.
		configValues, err := createConfigValues(application.Name, config, existingConfigValues, license, application, &appInfo, &versionInfo, registry, existingIdentityConfig)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create empty config values")
		}

		release.Manifests["userdata/config.yaml"] = mustMarshalConfigValues(configValues)
	}

	// Add the license to the upstream, if one was provided
	if license != nil {
		release.Manifests["userdata/license.yaml"] = MustMarshalLicense(license)
	}

	files, err := releaseToFiles(release)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get files from release")
	}

	upstream := &types.Upstream{
		URI:                      u.RequestURI(),
		Name:                     application.Name,
		Files:                    files,
		Type:                     "replicated",
		UpdateCursor:             release.UpdateCursor.Cursor,
		License:                  license,
		ChannelID:                channelID,
		ChannelName:              channelName,
		VersionLabel:             release.VersionLabel,
		IsRequired:               release.IsRequired,
		ReleaseNotes:             release.ReleaseNotes,
		ReleasedAt:               release.ReleasedAt,
		ReplicatedRegistryDomain: release.ReplicatedRegistryDomain,
		ReplicatedProxyDomain:    release.ReplicatedProxyDomain,
		ReplicatedChartNames:     release.ReplicatedChartNames,
		EmbeddedClusterArtifacts: release.EmbeddedClusterArtifacts,
	}

	return upstream, nil
}

func readReplicatedAppFromLocalPath(localPath string, localCursor replicatedapp.ReplicatedCursor, versionLabel string, isRequired bool, replicatedRegistryDomain string, replicatedProxyDomain string, replicatedChartNames []string, embeddedClusterArtifacts *kotsv1beta1.EmbeddedClusterArtifacts) (*Release, error) {
	release := Release{
		Manifests:                make(map[string][]byte),
		UpdateCursor:             localCursor,
		VersionLabel:             versionLabel,
		IsRequired:               isRequired,
		ReplicatedRegistryDomain: replicatedRegistryDomain,
		ReplicatedProxyDomain:    replicatedProxyDomain,
		ReplicatedChartNames:     replicatedChartNames,
		EmbeddedClusterArtifacts: embeddedClusterArtifacts,
	}

	err := filepath.Walk(localPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			contents, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// remove localpath prefix
			appPath := strings.TrimPrefix(path, localPath)
			appPath = strings.TrimLeft(appPath, string(os.PathSeparator))

			release.Manifests[appPath] = contents

			return nil
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to walk local path")
	}

	return &release, nil
}

func downloadReplicatedApp(replicatedUpstream *replicatedapp.ReplicatedUpstream, license *kotsv1beta1.License, cursor replicatedapp.ReplicatedCursor, reportingInfo *reportingtypes.ReportingInfo, selectedAppChannel string) (*Release, error) {
	getReq, err := replicatedUpstream.GetRequest("GET", license, cursor.Cursor, selectedAppChannel)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}

	reporting.InjectReportingInfoHeaders(getReq.Header, reportingInfo)

	logger.Info("pulling app from replicated",
		zap.String("url", getReq.URL.String()))

	getResp, err := util.DefaultHTTPClient.Do(getReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get request")
	}
	defer getResp.Body.Close()

	if getResp.StatusCode >= 300 {
		body, _ := io.ReadAll(getResp.Body)
		if len(body) > 0 {
			return nil, util.ActionableError{Message: string(body)}
		}
		return nil, errors.Errorf("unexpected result from get request: %d", getResp.StatusCode)
	}

	updateSequence := getResp.Header.Get("X-Replicated-ChannelSequence")
	updateChannelID := getResp.Header.Get("X-Replicated-ChannelID")
	updateChannelName := getResp.Header.Get("X-Replicated-ChannelName")
	versionLabel := getResp.Header.Get("X-Replicated-VersionLabel")
	isRequiredStr := getResp.Header.Get("X-Replicated-IsRequired")
	releasedAtStr := getResp.Header.Get("X-Replicated-ReleasedAt")
	replicatedChartNamesStr := getResp.Header.Get("X-Replicated-ReplicatedChartNames")
	replicatedAppDomain := getResp.Header.Get("X-Replicated-ReplicatedAppDomain")
	replicatedRegistryDomain := getResp.Header.Get("X-Replicated-ReplicatedRegistryDomain")
	replicatedProxyDomain := getResp.Header.Get("X-Replicated-ReplicatedProxyDomain")

	// If this is embedded cluster, update the environment variables with the new domains
	// so that the upgrade service can pick them up and use them.
	if util.IsEmbeddedCluster() {
		os.Setenv("REPLICATED_APP_ENDPOINT", "https://"+replicatedAppDomain)
		os.Setenv("REPLICATED_REGISTRY_DOMAIN", replicatedRegistryDomain)
		os.Setenv("PROXY_REGISTRY_DOMAIN", replicatedProxyDomain)
	}

	var releasedAt *time.Time
	r, err := time.Parse(time.RFC3339, releasedAtStr)
	if err == nil {
		releasedAt = &r
	}

	isRequired, _ := strconv.ParseBool(isRequiredStr)

	var replicatedChartNames []string
	if replicatedChartNamesStr != "" {
		replicatedChartNames = strings.Split(replicatedChartNamesStr, ",")
	}

	cachingBodyReader := util.NewCachingReader(getResp.Body, 4096)
	gzf, err := gzip.NewReader(cachingBodyReader)
	if err != nil {
		cachingBodyReader.LogReadableText("failed to create gzip reader")
		return nil, errors.Wrap(err, "failed to create gzip reader")
	}

	release := Release{
		Manifests: make(map[string][]byte),
		UpdateCursor: replicatedapp.ReplicatedCursor{
			ChannelID:   updateChannelID,
			ChannelName: updateChannelName,
			Cursor:      updateSequence,
		},
		VersionLabel:             versionLabel,
		IsRequired:               isRequired,
		ReleasedAt:               releasedAt,
		ReplicatedRegistryDomain: replicatedRegistryDomain,
		ReplicatedProxyDomain:    replicatedProxyDomain,
		ReplicatedChartNames:     replicatedChartNames,
		// NOTE: release notes come from Application spec
	}
	tarReader := tar.NewReader(gzf)
	i := 0
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			cachingBodyReader.LogReadableText("failed to get next file from reader")
			return nil, errors.Wrap(err, "failed to get next file from reader")
		}

		name := header.Name

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			content, err := io.ReadAll(tarReader)
			if err != nil {
				cachingBodyReader.LogReadableText("failed to read file from tar")
				return nil, errors.Wrap(err, "failed to read file from tar")
			}

			release.Manifests[name] = content
		}

		i++
	}

	return &release, nil
}

func listPendingChannelReleases(license *kotsv1beta1.License, lastUpdateCheckAt *time.Time, currentCursor replicatedapp.ReplicatedCursor, channelChanged bool, sortOrder string, reportingInfo *reportingtypes.ReportingInfo, selectedChannelID string) ([]ChannelRelease, *time.Time, error) {
	u, err := url.Parse(util.ReplicatedAppEndpoint(license))
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse endpoint from license")
	}

	hostname := u.Hostname()
	if u.Port() != "" {
		hostname = fmt.Sprintf("%s:%s", u.Hostname(), u.Port())
	}

	sequence := currentCursor.Cursor
	if channelChanged {
		sequence = ""
	}

	urlValues := url.Values{}
	urlValues.Set("channelSequence", sequence)
	urlValues.Add("licenseSequence", fmt.Sprintf("%d", license.Spec.LicenseSequence))
	urlValues.Add("isSemverSupported", "true")
	urlValues.Add("isEmbeddedCluster", fmt.Sprintf("%t", util.IsEmbeddedCluster()))
	urlValues.Add("selectedChannelId", selectedChannelID)

	if lastUpdateCheckAt != nil {
		urlValues.Add("lastUpdateCheckAt", lastUpdateCheckAt.UTC().Format(time.RFC3339))
	}

	if sortOrder != "" {
		urlValues.Add("sortOrder", sortOrder)
	}

	url := fmt.Sprintf("%s://%s/release/%s/pending?%s", u.Scheme, hostname, license.Spec.AppSlug, urlValues.Encode())

	req, err := util.NewRetryableRequest("GET", url, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to call newrequest")
	}

	reporting.InjectReportingInfoHeaders(req.Header, reportingInfo)

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", license.Spec.LicenseID, license.Spec.LicenseID)))))

	resp, err := util.DefaultHTTPClient.Do(req)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to execute get request")
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode >= 400 {
		if len(body) > 0 {
			return nil, nil, util.ActionableError{Message: string(body)}
		}
		return nil, nil, errors.Errorf("unexpected result from get request: %d", resp.StatusCode)
	}

	updateCheckTime, err := time.Parse(time.RFC3339, resp.Header.Get("X-Replicated-UpdateCheckAt"))
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to parse server side time")
	}

	var channelReleases struct {
		ChannelReleases []ChannelRelease `json:"channelReleases"`
	}
	if err := json.Unmarshal(body, &channelReleases); err != nil {
		return nil, nil, errors.Wrap(err, "failed to unmarshal response")
	}

	return channelReleases.ChannelReleases, &updateCheckTime, nil
}

func MustMarshalLicense(license *kotsv1beta1.License) []byte {
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(license, &b); err != nil {
		panic(err)
	}

	return b.Bytes()
}

func mustMarshalConfigValues(configValues *kotsv1beta1.ConfigValues) []byte {
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(configValues, &b); err != nil {
		panic(err)
	}

	return b.Bytes()
}

func createConfigValues(applicationName string, config *kotsv1beta1.Config, existingConfigValues *kotsv1beta1.ConfigValues, license *kotsv1beta1.License, app *kotsv1beta1.Application, appInfo *template.ApplicationInfo, versionInfo *template.VersionInfo, localRegistry registrytypes.RegistrySettings, identityConfig *kotsv1beta1.IdentityConfig) (*kotsv1beta1.ConfigValues, error) {
	templateContextValues := make(map[string]template.ItemValue)

	var newValues kotsv1beta1.ConfigValuesSpec
	if existingConfigValues != nil {
		for k, v := range existingConfigValues.Spec.Values {
			value := v.Value
			if value == "" {
				value = v.ValuePlaintext
			}
			templateContextValues[k] = template.ItemValue{
				Value:          value,
				Default:        v.Default,
				Filename:       v.Filename,
				RepeatableItem: v.RepeatableItem,
			}
		}
		newValues = kotsv1beta1.ConfigValuesSpec{
			Values: existingConfigValues.Spec.Values,
		}
	} else {
		newValues = kotsv1beta1.ConfigValuesSpec{
			Values: map[string]kotsv1beta1.ConfigValue{},
		}
	}

	if config == nil {
		return &kotsv1beta1.ConfigValues{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kots.io/v1beta1",
				Kind:       "ConfigValues",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: applicationName,
			},
			Spec: newValues,
		}, nil
	}

	builderOptions := template.BuilderOptions{
		ConfigGroups:    config.Spec.Groups,
		ExistingValues:  templateContextValues,
		LocalRegistry:   localRegistry,
		License:         license,
		Application:     app,
		ApplicationInfo: appInfo,
		VersionInfo:     versionInfo,
		IdentityConfig:  identityConfig,
		DecryptValues:   true,
	}
	builder, _, err := template.NewBuilder(builderOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create config context")
	}

	for _, group := range config.Spec.Groups {
		for _, item := range group.Items {
			err := createConfigValue(newValues, builder, builderOptions, item.Name, item.Value.String(), item.Default.String(), "")
			if err != nil {
				return nil, errors.Wrapf(err, "failed to create config value for %s", item.Name)
			}

			for _, repeatableValues := range item.ValuesByGroup {
				for name, defaultValue := range repeatableValues {
					// currently, `ValuesByGroup` defined in the config by the vendor are considered the defaults for repeatable items.
					err := createConfigValue(newValues, builder, builderOptions, name, "", defaultValue, item.Name)
					if err != nil {
						return nil, errors.Wrapf(err, "failed to create config value for %s", name)
					}
				}
			}
		}
	}

	configValues := kotsv1beta1.ConfigValues{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "ConfigValues",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: applicationName,
		},
		Spec: newValues,
	}

	return &configValues, nil
}

func createConfigValue(
	valuesSpec kotsv1beta1.ConfigValuesSpec,
	builder template.Builder,
	builderOptions template.BuilderOptions,
	itemName string,
	itemValue string,
	itemDefault string,
	repeatableItem string,
) error {
	var foundValue, foundValuePlaintext, foundFilename, foundRepeatableItem string
	prevValue, ok := valuesSpec.Values[itemName]
	if ok {
		foundValue = prevValue.Value
		foundValuePlaintext = prevValue.ValuePlaintext
		foundFilename = prevValue.Filename
		foundRepeatableItem = prevValue.RepeatableItem
	}

	renderedValue, err := builder.RenderTemplate(itemName, itemValue)
	if err != nil {
		return errors.Wrapf(err, "failed to render value for config item %q", itemName)
	}

	renderedDefault, err := builder.RenderTemplate(itemName, itemDefault)
	if err != nil {
		return errors.Wrapf(err, "failed to render default for config item %q", itemName)
	}

	if foundValue != "" || foundValuePlaintext != "" {
		// found a user value for this item, don't overwrite it
		valuesSpec.Values[itemName] = kotsv1beta1.ConfigValue{
			Value:          foundValue,
			ValuePlaintext: foundValuePlaintext,
			Default:        renderedDefault,
			Filename:       foundFilename,
			RepeatableItem: foundRepeatableItem,
		}
	} else {
		// no user value found, use the value defined in the config by the vendor
		valuesSpec.Values[itemName] = kotsv1beta1.ConfigValue{
			Value:          renderedValue,
			Default:        renderedDefault,
			Filename:       foundFilename,
			RepeatableItem: repeatableItem,
		}
		builderOptions.ExistingValues[itemName] = template.ItemValue{
			Value:          renderedValue,
			Default:        renderedDefault,
			Filename:       foundFilename,
			RepeatableItem: repeatableItem,
		}
	}

	return nil
}

func findConfigValuesInFile(filename string) (*kotsv1beta1.ConfigValues, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to open file")
	}

	return contentToConfigValues(content), nil
}

func contentToConfigValues(content []byte) *kotsv1beta1.ConfigValues {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil
	}

	if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "ConfigValues" {
		return obj.(*kotsv1beta1.ConfigValues)
	}

	return nil
}

func mustMarshalIdentityConfig(identityConfig *kotsv1beta1.IdentityConfig) []byte {
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(identityConfig, &b); err != nil {
		panic(err)
	}

	return b.Bytes()
}

func findIdentityConfigInFile(filename string) (*kotsv1beta1.IdentityConfig, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to open file")
	}

	return contentToIdentityConfig(content), nil
}

func contentToIdentityConfig(content []byte) *kotsv1beta1.IdentityConfig {
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil
	}

	if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "IdentityConfig" {
		return obj.(*kotsv1beta1.IdentityConfig)
	}

	return nil
}

func findConfigInRelease(release *Release) *kotsv1beta1.Config {
	var config *kotsv1beta1.Config

	for _, content := range release.Manifests {
		docs := util.ConvertToSingleDocs(content)
		for _, doc := range docs {
			decode := scheme.Codecs.UniversalDeserializer().Decode
			obj, gvk, err := decode(doc, nil, nil)
			if err != nil {
				continue
			}

			if gvk.Group == "kots.io" {
				if gvk.Version == "v1beta1" {
					if gvk.Kind == "Config" {
						config = obj.(*kotsv1beta1.Config)
					}
				}
			}
		}
	}

	return config
}

func findAppInRelease(release *Release) *kotsv1beta1.Application {
	for _, content := range release.Manifests {
		for _, doc := range util.ConvertToSingleDocs(content) {
			decode := scheme.Codecs.UniversalDeserializer().Decode
			obj, gvk, err := decode(doc, nil, nil)
			if err != nil {
				continue
			}

			if gvk.Group == "kots.io" {
				if gvk.Version == "v1beta1" {
					if gvk.Kind == "Application" {
						return obj.(*kotsv1beta1.Application)
					}
				}
			}
		}
	}

	// create an app manifest on the fly
	app := &kotsv1beta1.Application{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "Application",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "replicated-kots-app",
		},
		Spec: kotsv1beta1.ApplicationSpec{
			Title: "Replicated KOTS App",
			Icon:  "",
		},
	}
	return app
}

func releaseToFiles(release *Release) ([]types.UpstreamFile, error) {
	upstreamFiles := []types.UpstreamFile{}

	for filename, content := range release.Manifests {
		upstreamFile := types.UpstreamFile{
			Path:    filename,
			Content: content,
		}

		upstreamFiles = append(upstreamFiles, upstreamFile)
	}

	// Stash the user data for this search (we will readd at the end)
	userdataFiles := []types.UpstreamFile{}
	withoutUserdataFiles := []types.UpstreamFile{}
	for _, file := range upstreamFiles {
		d, _ := path.Split(file.Path)
		dirs := strings.Split(d, string(os.PathSeparator))

		if dirs[0] == "userdata" {
			userdataFiles = append(userdataFiles, file)
		} else {
			withoutUserdataFiles = append(withoutUserdataFiles, file)
		}
	}

	// remove any common prefix from all files
	if len(withoutUserdataFiles) > 0 {
		firstFileDir, _ := path.Split(withoutUserdataFiles[0].Path)
		commonPrefix := strings.Split(firstFileDir, string(os.PathSeparator))

		for _, file := range withoutUserdataFiles {
			d, _ := path.Split(file.Path)
			dirs := strings.Split(d, string(os.PathSeparator))

			commonPrefix = util.CommonSlicePrefix(commonPrefix, dirs)

		}

		cleanedUpstreamFiles := []types.UpstreamFile{}
		for _, file := range withoutUserdataFiles {
			d, f := path.Split(file.Path)
			d2 := strings.Split(d, string(os.PathSeparator))

			cleanedUpstreamFile := file
			d2 = d2[len(commonPrefix):]
			cleanedUpstreamFile.Path = path.Join(path.Join(d2...), f)

			cleanedUpstreamFiles = append(cleanedUpstreamFiles, cleanedUpstreamFile)
		}

		upstreamFiles = cleanedUpstreamFiles
	}

	upstreamFiles = append(upstreamFiles, userdataFiles...)

	return upstreamFiles, nil
}
