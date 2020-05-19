package upstream

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/crypto"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/replicatedhq/kots/pkg/version"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

const DefaultMetadata = `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: "default-application"
spec:
  title: "the application"
  icon: https://cdn2.iconfinder.com/data/icons/mixd/512/16_kubernetes-512.png
  releaseNotes: |
    release notes`

type ReplicatedUpstream struct {
	Channel      *string
	AppSlug      string
	VersionLabel *string
	Sequence     *int
}

type ReplicatedCursor struct {
	ChannelName string
	Cursor      string
}

type App struct {
	Name string
}

type Release struct {
	UpdateCursor ReplicatedCursor
	VersionLabel string
	ReleaseNotes string
	Manifests    map[string][]byte
}

type ChannelRelease struct {
	ChannelSequence int    `json:"channelSequence"`
	ReleaseSequence int    `json:"releaseSequence"`
	VersionLabel    string `json:"versionLabel"`
	CreatedAt       string `json:"createdAt"`
}

func (this ReplicatedCursor) Equal(other ReplicatedCursor) bool {
	return this.ChannelName == other.ChannelName && this.Cursor == other.Cursor
}

func getUpdatesReplicated(u *url.URL, localPath string, currentCursor ReplicatedCursor, versionLabel string, license *kotsv1beta1.License) ([]Update, error) {
	if localPath != "" {
		parsedLocalRelease, err := readReplicatedAppFromLocalPath(localPath, currentCursor, versionLabel)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read replicated app from local path")
		}

		return []Update{{Cursor: parsedLocalRelease.UpdateCursor.Cursor, VersionLabel: versionLabel}}, nil
	}

	// A license file is required to be set for this to succeed
	if license == nil {
		return nil, errors.New("No license was provided")
	}

	replicatedUpstream, err := parseReplicatedURL(u)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse replicated upstream")
	}

	remoteLicense, err := getSuccessfulHeadResponse(replicatedUpstream, license)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get successful head response")
	}

	pendingReleases, err := listPendingChannelReleases(replicatedUpstream, remoteLicense, currentCursor)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list replicated app releases")
	}

	updates := []Update{}
	for _, pendingRelease := range pendingReleases {
		updates = append(updates, Update{
			Cursor:       strconv.Itoa(pendingRelease.ChannelSequence),
			VersionLabel: pendingRelease.VersionLabel,
		})
	}
	return updates, nil
}

func downloadReplicated(u *url.URL, localPath string, rootDir string, useAppDir bool, license *kotsv1beta1.License, existingConfigValues *kotsv1beta1.ConfigValues, updateCursor ReplicatedCursor, versionLabel string, cipher *crypto.AESCipher) (*types.Upstream, error) {
	var release *Release

	if localPath != "" {
		parsedLocalRelease, err := readReplicatedAppFromLocalPath(localPath, updateCursor, versionLabel)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read replicated app from local path")
		}

		release = parsedLocalRelease
	} else {
		// A license file is required to be set for this to succeed
		if license == nil {
			return nil, errors.New("No license was provided")
		}

		replicatedUpstream, err := parseReplicatedURL(u)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse replicated upstream")
		}

		remoteLicense, err := getSuccessfulHeadResponse(replicatedUpstream, license)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get successful head response")
		}

		downloadedRelease, err := downloadReplicatedApp(replicatedUpstream, remoteLicense, updateCursor)
		if err != nil {
			return nil, errors.Wrap(err, "failed to download replicated app")
		}

		license, err = kotslicense.GetLatestLicense(license)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get latest license")
		}

		release = downloadedRelease
	}

	// Find the config in the upstream and write out default values

	application := findAppInRelease(release) // this function never returns nil

	// NOTE: this currently comes from the application spec and not the channel release meta
	if release.ReleaseNotes == "" {
		release.ReleaseNotes = application.Spec.ReleaseNotes
	}

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

	config, _, _, _, err := findTemplateContextDataInRelease(release)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find config in release")
	}
	if config != nil || existingConfigValues != nil {
		// If config existed and was removed from the app,
		// values will be carried over to the new version anyway.
		configValues, err := createConfigValues(application.Name, config, existingConfigValues, cipher, license, nil)
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

	// get channel name from license, if one was provided
	channelName := ""
	if license != nil {
		channelName = license.Spec.ChannelName
	}

	upstream := &types.Upstream{
		URI:           u.RequestURI(),
		Name:          application.Name,
		Files:         files,
		Type:          "replicated",
		UpdateCursor:  release.UpdateCursor.Cursor,
		ChannelName:   channelName,
		VersionLabel:  release.VersionLabel,
		ReleaseNotes:  release.ReleaseNotes,
		EncryptionKey: cipher.ToString(),
	}

	return upstream, nil
}

func (r *ReplicatedUpstream) getRequest(method string, license *kotsv1beta1.License, cursor ReplicatedCursor) (*http.Request, error) {
	u, err := url.Parse(license.Spec.Endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse endpoint from license")
	}

	hostname := u.Hostname()
	if u.Port() != "" {
		hostname = fmt.Sprintf("%s:%s", u.Hostname(), u.Port())
	}

	urlPath := path.Join(hostname, "release", license.Spec.AppSlug)
	if r.Channel != nil {
		urlPath = path.Join(urlPath, *r.Channel)
	}

	urlValues := url.Values{}
	urlValues.Set("channelSequence", cursor.Cursor)
	urlValues.Add("licenseSequence", fmt.Sprintf("%d", license.Spec.LicenseSequence))
	url := fmt.Sprintf("%s://%s?%s", u.Scheme, urlPath, urlValues.Encode())

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}

	req.Header.Add("User-Agent", fmt.Sprintf("KOTS/%s", version.Version()))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", license.Spec.LicenseID, license.Spec.LicenseID)))))

	return req, nil
}

func parseReplicatedURL(u *url.URL) (*ReplicatedUpstream, error) {
	replicatedUpstream := ReplicatedUpstream{}

	if u.User != nil {
		if u.User.Username() != "" {
			replicatedUpstream.AppSlug = u.User.Username()
			versionLabel := u.Hostname()
			replicatedUpstream.VersionLabel = &versionLabel
		}
	}

	if replicatedUpstream.AppSlug == "" {
		replicatedUpstream.AppSlug = u.Hostname()
		if u.Path != "" {
			channel := strings.TrimPrefix(u.Path, "/")
			replicatedUpstream.Channel = &channel
		}
	}

	return &replicatedUpstream, nil
}

func getSuccessfulHeadResponse(replicatedUpstream *ReplicatedUpstream, license *kotsv1beta1.License) (*kotsv1beta1.License, error) {
	headReq, err := replicatedUpstream.getRequest("HEAD", license, ReplicatedCursor{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	headResp, err := http.DefaultClient.Do(headReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute head request")
	}
	defer headResp.Body.Close()

	if headResp.StatusCode == 401 {
		return nil, errors.New("license was not accepted")
	}

	if headResp.StatusCode == 403 {
		return nil, util.ActionableError{Message: "License is expired"}
	}

	if headResp.StatusCode >= 400 {
		return nil, errors.Errorf("unexpected result from head request: %d", headResp.StatusCode)
	}

	return license, nil
}

func readReplicatedAppFromLocalPath(localPath string, localCursor ReplicatedCursor, versionLabel string) (*Release, error) {
	release := Release{
		Manifests:    make(map[string][]byte),
		UpdateCursor: localCursor,
		VersionLabel: versionLabel,
	}

	err := filepath.Walk(localPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			contents, err := ioutil.ReadFile(path)
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

func downloadReplicatedApp(replicatedUpstream *ReplicatedUpstream, license *kotsv1beta1.License, cursor ReplicatedCursor) (*Release, error) {
	getReq, err := replicatedUpstream.getRequest("GET", license, cursor)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get request")
	}
	defer getResp.Body.Close()

	if getResp.StatusCode >= 300 {
		body, _ := ioutil.ReadAll(getResp.Body)
		if len(body) > 0 {
			return nil, util.ActionableError{Message: string(body)}
		}
		return nil, errors.Errorf("unexpected result from get request: %d", getResp.StatusCode)
	}

	updateSequence := getResp.Header.Get("X-Replicated-ChannelSequence")
	updateChannel := getResp.Header.Get("X-Replicated-ChannelName")
	versionLabel := getResp.Header.Get("X-Replicated-VersionLabel")

	gzf, err := gzip.NewReader(getResp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new gzip reader")
	}

	release := Release{
		Manifests: make(map[string][]byte),
		UpdateCursor: ReplicatedCursor{
			ChannelName: updateChannel,
			Cursor:      updateSequence,
		},
		VersionLabel: versionLabel,
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
			return nil, errors.Wrap(err, "failed to get next file from reader")
		}

		name := header.Name

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			content, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read file from tar")
			}

			release.Manifests[name] = content
		}

		i++
	}

	return &release, nil
}

func listPendingChannelReleases(replicatedUpstream *ReplicatedUpstream, license *kotsv1beta1.License, cursor ReplicatedCursor) ([]ChannelRelease, error) {
	u, err := url.Parse(license.Spec.Endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse endpoint from license")
	}

	hostname := u.Hostname()
	if u.Port() != "" {
		hostname = fmt.Sprintf("%s:%s", u.Hostname(), u.Port())
	}

	sequence := cursor.Cursor
	if license.Spec.ChannelName != cursor.ChannelName {
		sequence = ""
	}

	urlValues := url.Values{}
	urlValues.Set("channelSequence", sequence)
	urlValues.Add("licenseSequence", fmt.Sprintf("%d", license.Spec.LicenseSequence))
	url := fmt.Sprintf("%s://%s/release/%s/pending?%s", u.Scheme, hostname, license.Spec.AppSlug, urlValues.Encode())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}

	req.Header.Add("User-Agent", fmt.Sprintf("KOTS/%s", version.Version()))
	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", license.Spec.LicenseID, license.Spec.LicenseID)))))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get request")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode >= 400 {
		if len(body) > 0 {
			return nil, util.ActionableError{Message: string(body)}
		}
		return nil, errors.Errorf("unexpected result from get request: %d", resp.StatusCode)
	}

	var channelReleases struct {
		ChannelReleases []ChannelRelease `json:"channelReleases"`
	}
	if err := json.Unmarshal(body, &channelReleases); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response")
	}

	return channelReleases.ChannelReleases, nil
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

func createConfigValues(applicationName string, config *kotsv1beta1.Config, existingConfigValues *kotsv1beta1.ConfigValues, cipher *crypto.AESCipher, license *kotsv1beta1.License, unsignedLicense *kotsv1beta1.UnsignedLicense) (*kotsv1beta1.ConfigValues, error) {
	templateContextValues := make(map[string]template.ItemValue)

	var newValues kotsv1beta1.ConfigValuesSpec
	if existingConfigValues != nil {
		for k, v := range existingConfigValues.Spec.Values {
			templateContextValues[k] = template.ItemValue{
				Value:   v.Value,
				Default: v.Default,
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

	// Today, these aren't needed in this function
	// They are needed in rendering the base
	// We should get this supported before 1.13.0 ships
	localRegistry := template.LocalRegistry{}

	builder, _, err := template.NewBuilder(config.Spec.Groups, templateContextValues, localRegistry, cipher, license, unsignedLicense)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create config context")
	}

	for _, group := range config.Spec.Groups {
		for _, item := range group.Items {
			var foundValue string
			prevValue, ok := newValues.Values[item.Name]
			if ok && prevValue.Value != "" {
				foundValue = prevValue.Value
			}

			renderedValue, err := builder.RenderTemplate(item.Name, item.Value.String())
			if err != nil {
				return nil, errors.Wrap(err, "failed to render config item value")
			}

			renderedDefault, err := builder.RenderTemplate(item.Name, item.Default.String())
			if err != nil {
				return nil, errors.Wrap(err, "failed to render config item default")
			}

			if renderedValue == "" && renderedDefault == "" && foundValue == "" {
				continue
			}

			if foundValue != "" {
				newValues.Values[item.Name] = kotsv1beta1.ConfigValue{
					Value:   foundValue,
					Default: renderedDefault,
				}
			} else {
				newValues.Values[item.Name] = kotsv1beta1.ConfigValue{
					Value:   renderedValue,
					Default: renderedDefault,
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

func findConfigValuesInFile(filename string) (*kotsv1beta1.ConfigValues, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to open file")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, nil
	}

	if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "ConfigValues" {
		return obj.(*kotsv1beta1.ConfigValues), nil
	}

	return nil, nil
}

func findTemplateContextDataInRelease(release *Release) (*kotsv1beta1.Config, *kotsv1beta1.ConfigValues, *kotsv1beta1.License, *kotsv1beta1.Installation, error) {
	var config *kotsv1beta1.Config
	var values *kotsv1beta1.ConfigValues
	var license *kotsv1beta1.License
	var installation *kotsv1beta1.Installation

	for _, content := range release.Manifests {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode(content, nil, nil)
		if err != nil {
			continue
		}

		if gvk.Group == "kots.io" {
			if gvk.Version == "v1beta1" {
				if gvk.Kind == "Config" {
					config = obj.(*kotsv1beta1.Config)
				} else if gvk.Kind == "ConfigValues" {
					values = obj.(*kotsv1beta1.ConfigValues)
				} else if gvk.Kind == "License" {
					license = obj.(*kotsv1beta1.License)
				} else if gvk.Kind == "Installation" {
					installation = obj.(*kotsv1beta1.Installation)
				}
			}
		}
	}

	return config, values, license, installation, nil
}

func findAppInRelease(release *Release) *kotsv1beta1.Application {
	for _, content := range release.Manifests {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode(content, nil, nil)
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

	// Using Ship apps for now, so let's create an app manifest on the fly
	app := &kotsv1beta1.Application{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "Application",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "replicated-kots-app",
		},
		Spec: kotsv1beta1.ApplicationSpec{
			Title: "Replicated Kots App",
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

// GetApplicationMetadata will return any available application yaml from
// the upstream. If there is no application.yaml, it will return
// a placeholder one
func GetApplicationMetadata(upstream *url.URL) ([]byte, error) {
	metadata, err := getApplicationMetadataFromHost("replicated.app", upstream)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get metadata from replicated.app")
	}

	if metadata == nil {
		otherMetadata, err := getApplicationMetadataFromHost("staging.replicated.app", upstream)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get metadata from staging.replicated.app")
		}

		metadata = otherMetadata
	}

	if metadata == nil {
		metadata = []byte(DefaultMetadata)
	}

	return metadata, nil
}

func getApplicationMetadataFromHost(host string, upstream *url.URL) ([]byte, error) {
	r, err := parseReplicatedURL(upstream)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse replicated upstream")
	}

	url := fmt.Sprintf("https://%s/metadata/%s", host, r.AppSlug)

	if r.Channel != nil {
		url = fmt.Sprintf("%s/%s", url, *r.Channel)
	}

	getReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}

	getReq.Header.Add("User-Agent", fmt.Sprintf("KOTS/%s", version.Version()))

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get request")
	}
	defer getResp.Body.Close()

	if getResp.StatusCode == 404 {
		// no metadata is not an error
		return nil, nil
	}

	if getResp.StatusCode >= 400 {
		return nil, errors.Errorf("unexpected result from get request: %d", getResp.StatusCode)
	}

	respBody, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	return respBody, nil
}
