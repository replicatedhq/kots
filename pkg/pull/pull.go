package pull

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/apparchive"
	"github.com/replicatedhq/kots/pkg/archives"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/downstream"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadmconfig"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/rendered"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	"github.com/replicatedhq/kots/pkg/upstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsv1beta2 "github.com/replicatedhq/kotskinds/apis/kots/v1beta2"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

type PullOptions struct {
	RootDir                 string
	Namespace               string
	Downstreams             []string
	LocalPath               string
	LicenseObj              *kotsv1beta1.License
	LicenseFile             string
	LicenseEndpointOverride string // only used for testing
	InstallationFile        string
	AirgapRoot              string
	AirgapBundle            string
	ConfigFile              string
	IdentityConfigFile      string
	UpdateCursor            string
	ExcludeKotsKinds        bool
	ExcludeAdminConsole     bool
	IncludeMinio            bool
	SharedPassword          string
	CreateAppDir            bool
	Silent                  bool
	RewriteImages           bool
	RewriteImageOptions     registrytypes.RegistrySettings
	SkipHelmChartCheck      bool
	ReportWriter            io.Writer
	AppID                   string
	AppSlug                 string
	AppSequence             int64
	AppVersionLabel         string
	IsGitOps                bool
	HTTPProxyEnvValue       string
	HTTPSProxyEnvValue      string
	NoProxyEnvValue         string
	ReportingInfo           *reportingtypes.ReportingInfo
	SkipCompatibilityCheck  bool
	KotsKinds               *kotsutil.KotsKinds
}

var (
	ErrConfigNeeded = errors.New("version needs config")
)

// PullApplicationMetadata will return the application metadata yaml, if one is
// available for the upstream
func PullApplicationMetadata(upstreamURI string, versionLabel string) (*replicatedapp.ApplicationMetadata, error) {
	u, err := url.ParseRequestURI(upstreamURI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse uri")
	}

	// metadata is only currently supported on licensed apps
	if u.Scheme != "replicated" {
		return nil, nil
	}

	metadata, err := replicatedapp.GetApplicationMetadata(u, versionLabel)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get application metadata")
	}

	return metadata, nil
}

// Pull will download the application specified in upstreamURI using the options
// specified in pullOptions. It returns the directory that the app was pulled to
func Pull(upstreamURI string, pullOptions PullOptions) (string, error) {
	log := logger.NewCLILogger(os.Stdout)

	if pullOptions.Silent {
		log.Silence()
	}

	log.Initialize()

	if pullOptions.ReportWriter == nil {
		pullOptions.ReportWriter = ioutil.Discard
	}

	uri, err := url.ParseRequestURI(upstreamURI)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse uri")
	}

	fetchOptions := upstreamtypes.FetchOptions{
		RootDir:                pullOptions.RootDir,
		UseAppDir:              pullOptions.CreateAppDir,
		LocalPath:              pullOptions.LocalPath,
		CurrentCursor:          pullOptions.UpdateCursor,
		AppSlug:                pullOptions.AppSlug,
		AppSequence:            pullOptions.AppSequence,
		AppVersionLabel:        pullOptions.AppVersionLabel,
		LocalRegistry:          pullOptions.RewriteImageOptions,
		ReportingInfo:          pullOptions.ReportingInfo,
		SkipCompatibilityCheck: pullOptions.SkipCompatibilityCheck,
	}

	var installation *kotsv1beta1.Installation

	_, localConfigValues, localLicense, localInstallation, localIdentityConfig, err := findConfig(pullOptions.LocalPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to find config files in local path")
	}

	if pullOptions.LicenseObj != nil {
		fetchOptions.License = pullOptions.LicenseObj
	} else if pullOptions.LicenseFile != "" {
		license, err := kotsutil.LoadLicenseFromPath(pullOptions.LicenseFile)
		if err != nil {
			if errors.Cause(err) == kotslicense.ErrSignatureInvalid {
				return "", kotslicense.ErrSignatureInvalid
			}
			if errors.Cause(err) == kotslicense.ErrSignatureMissing {
				return "", kotslicense.ErrSignatureMissing
			}
			return "", errors.Wrap(err, "failed to parse license from file")
		}
		fetchOptions.License = license
	} else {
		fetchOptions.License = localLicense
	}

	if fetchOptions.License != nil {
		verifiedLicense, err := kotslicense.VerifySignature(fetchOptions.License)
		if err != nil {
			return "", errors.Wrap(err, "failed to verify signature")
		}
		fetchOptions.License = verifiedLicense

		if pullOptions.LicenseEndpointOverride != "" {
			fetchOptions.License.Spec.Endpoint = pullOptions.LicenseEndpointOverride
		}
	}

	encryptConfig := false
	if pullOptions.ConfigFile != "" {
		config, err := ParseConfigValuesFromFile(pullOptions.ConfigFile)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse config values from file")
		}
		fetchOptions.ConfigValues = config
		encryptConfig = true
	} else {
		fetchOptions.ConfigValues = localConfigValues
	}

	var identityConfig *kotsv1beta1.IdentityConfig
	if pullOptions.IdentityConfigFile != "" {
		identityConfig, err = ParseIdentityConfigFromFile(pullOptions.IdentityConfigFile)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse identity config from file")
		}
	} else {
		identityConfig = localIdentityConfig
	}
	fetchOptions.IdentityConfig = identityConfig

	if pullOptions.InstallationFile != "" {
		i, err := parseInstallationFromFile(pullOptions.InstallationFile)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse installation from file")
		}
		installation = i
	} else {
		installation = localInstallation
	}

	if installation != nil {
		fetchOptions.EncryptionKey = installation.Spec.EncryptionKey
		fetchOptions.CurrentVersionLabel = installation.Spec.VersionLabel
		fetchOptions.CurrentChannelID = installation.Spec.ChannelID
		fetchOptions.CurrentChannelName = installation.Spec.ChannelName
		if fetchOptions.CurrentCursor == "" {
			fetchOptions.CurrentCursor = installation.Spec.UpdateCursor
		}
		fetchOptions.CurrentVersionIsRequired = installation.Spec.IsRequired
		fetchOptions.CurrentReplicatedRegistryDomain = installation.Spec.ReplicatedRegistryDomain
		fetchOptions.CurrentReplicatedProxyDomain = installation.Spec.ReplicatedProxyDomain
	}

	if pullOptions.AirgapRoot != "" {
		if expired, err := kotslicense.LicenseIsExpired(fetchOptions.License); err != nil {
			return "", errors.Wrap(err, "failed to check license expiration")
		} else if expired {
			return "", util.ActionableError{Message: "License is expired"}
		}

		airgap, err := kotsutil.FindAirgapMetaInDir(pullOptions.AirgapRoot)
		if err != nil {
			return "", errors.Wrap(err, "failed to find airgap meta")
		}

		if fetchOptions.AppVersionLabel != "" && fetchOptions.AppVersionLabel != airgap.Spec.VersionLabel {
			logger.Infof("Expecting to install version %s but airgap bundle version is %s.", fetchOptions.AppVersionLabel, airgap.Spec.VersionLabel)
		}

		if fetchOptions.License.Spec.ChannelID != airgap.Spec.ChannelID {
			return "", util.ActionableError{
				NoRetry: true, // if this is airgap upload, make sure to free up tmp space
				Message: fmt.Sprintf("License (%s) and airgap bundle (%s) channels do not match.", fetchOptions.License.Spec.ChannelName, airgap.Spec.ChannelName),
			}
		}

		if err := publicKeysMatch(log, fetchOptions.License, airgap); err != nil {
			return "", errors.Wrap(err, "failed to validate app key")
		}

		airgapAppFiles, err := os.MkdirTemp("", "airgap-kots")
		if err != nil {
			return "", errors.Wrap(err, "failed to create temp airgap dir")
		}
		defer os.RemoveAll(airgapAppFiles)

		err = util.ExtractTGZArchive(filepath.Join(pullOptions.AirgapRoot, "app.tar.gz"), airgapAppFiles)
		if err != nil {
			return "", errors.Wrap(err, "failed to extract app files")
		}

		fetchOptions.Airgap = airgap
		fetchOptions.LocalPath = airgapAppFiles
	}

	prevV1Beta1HelmCharts := []*kotsv1beta1.HelmChart{}
	if pullOptions.KotsKinds != nil && pullOptions.KotsKinds.V1Beta1HelmCharts != nil {
		for _, v1Beta1Chart := range pullOptions.KotsKinds.V1Beta1HelmCharts.Items {
			kc := v1Beta1Chart
			prevV1Beta1HelmCharts = append(prevV1Beta1HelmCharts, &kc)
		}
	}

	log.ActionWithSpinner("Pulling upstream")
	io.WriteString(pullOptions.ReportWriter, "Pulling upstream\n")
	u, err := upstream.FetchUpstream(upstreamURI, &fetchOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to fetch upstream")
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to get k8s clientset")
	}

	includeAdminConsole := uri.Scheme == "replicated" && !pullOptions.ExcludeAdminConsole

	writeUpstreamOptions := upstreamtypes.WriteOptions{
		RootDir:             pullOptions.RootDir,
		CreateAppDir:        pullOptions.CreateAppDir,
		IncludeAdminConsole: includeAdminConsole,
		SharedPassword:      pullOptions.SharedPassword,
		EncryptConfig:       encryptConfig,
		HTTPProxyEnvValue:   pullOptions.HTTPProxyEnvValue,
		HTTPSProxyEnvValue:  pullOptions.HTTPSProxyEnvValue,
		NoProxyEnvValue:     pullOptions.NoProxyEnvValue,
		IsOpenShift:         k8sutil.IsOpenShift(clientset),
		IsGKEAutopilot:      k8sutil.IsGKEAutopilot(clientset),
		IncludeMinio:        pullOptions.IncludeMinio,
		IsAirgap:            pullOptions.AirgapRoot != "",
		KotsadmID:           k8sutil.GetKotsadmID(clientset),
		AppID:               pullOptions.AppID,
	}
	if err := upstream.WriteUpstream(u, writeUpstreamOptions); err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to write upstream")
	}
	log.FinishSpinner()

	registrySettings := registrytypes.RegistrySettings{
		Hostname:   pullOptions.RewriteImageOptions.Hostname,
		Namespace:  pullOptions.RewriteImageOptions.Namespace,
		Username:   pullOptions.RewriteImageOptions.Username,
		Password:   pullOptions.RewriteImageOptions.Password,
		IsReadOnly: pullOptions.RewriteImageOptions.IsReadOnly,
	}

	renderOptions := base.RenderOptions{
		SplitMultiDocYAML: true,
		Namespace:         pullOptions.Namespace,
		RegistrySettings:  registrySettings,
		ExcludeKotsKinds:  pullOptions.ExcludeKotsKinds,
		Log:               log,
		AppSlug:           pullOptions.AppSlug,
		Sequence:          pullOptions.AppSequence,
		IsAirgap:          pullOptions.AirgapRoot != "",
	}
	log.ActionWithSpinner("Rendering KOTS custom resources")
	io.WriteString(pullOptions.ReportWriter, "Rendering KOTS custom resources\n")

	renderedKotsKindsMap, err := base.RenderKotsKinds(u, &renderOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to render upstream")
	}

	renderedKotsKinds, err := kotsutil.KotsKindsFromMap(renderedKotsKindsMap)
	if err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to load rendered kotskinds from map")
	}
	log.FinishSpinner()

	needsConfig, err := kotsadmconfig.NeedsConfiguration(pullOptions.AppSlug, pullOptions.AppSequence, pullOptions.AirgapRoot != "", renderedKotsKinds, registrySettings)
	if err != nil {
		return "", errors.Wrap(err, "failed to check if version needs configuration")
	}

	processImageOptions := image.ProcessImageOptions{
		AppSlug:          pullOptions.AppSlug,
		Namespace:        pullOptions.Namespace,
		RewriteImages:    pullOptions.RewriteImages,
		RegistrySettings: pullOptions.RewriteImageOptions,
		CopyImages:       !pullOptions.RewriteImageOptions.IsReadOnly,
		RootDir:          pullOptions.RootDir,
		IsAirgap:         pullOptions.AirgapRoot != "",
		AirgapRoot:       pullOptions.AirgapRoot,
		AirgapBundle:     pullOptions.AirgapBundle,
		PushImages:       pullOptions.RewriteImageOptions.Hostname != "",
		CreateAppDir:     pullOptions.CreateAppDir,
		ReportWriter:     pullOptions.ReportWriter,
	}

	if needsConfig {
		if err := kotsutil.WriteKotsKinds(renderedKotsKindsMap, u.GetKotsKindsDir(writeUpstreamOptions)); err != nil {
			return "", errors.Wrap(err, "failed to write the rendered kots kinds")
		}
		if processImageOptions.RewriteImages && processImageOptions.AirgapRoot != "" {
			// if this is an airgap install, we still need to process the images
			if _, err = midstream.ProcessAirgapImages(processImageOptions, nil, nil, renderedKotsKinds, fetchOptions.License, log); err != nil {
				return "", errors.Wrap(err, "failed to process airgap images")
			}
		}
		return "", ErrConfigNeeded
	}

	var v1Beta1HelmCharts []*kotsv1beta1.HelmChart
	if renderedKotsKinds.V1Beta1HelmCharts != nil {
		for _, v1Beta1Chart := range renderedKotsKinds.V1Beta1HelmCharts.Items {
			kc := v1Beta1Chart
			v1Beta1HelmCharts = append(v1Beta1HelmCharts, &kc)
		}
	}

	var v1Beta2HelmCharts []*kotsv1beta2.HelmChart
	if renderedKotsKinds.V1Beta2HelmCharts != nil {
		for _, v1Beta2Chart := range renderedKotsKinds.V1Beta2HelmCharts.Items {
			kc := v1Beta2Chart
			v1Beta2HelmCharts = append(v1Beta2HelmCharts, &kc)
		}
	}

	if !pullOptions.SkipHelmChartCheck {
		for _, prevChart := range prevV1Beta1HelmCharts {
			if !prevChart.Spec.Exclude.IsEmpty() {
				isExcluded, err := prevChart.Spec.Exclude.Boolean()
				if err == nil && isExcluded {
					continue // this chart was excluded, so any changes to UseHelmInstall are irrelevant
				}
			}

			for _, newChart := range v1Beta1HelmCharts {
				if prevChart.GetReleaseName() != newChart.GetReleaseName() {
					continue
				}
				if prevChart.Spec.UseHelmInstall != newChart.Spec.UseHelmInstall {
					return "", errors.Errorf("deployment method for chart release %s has changed", newChart.GetReleaseName())
				}
			}

			for _, newChart := range v1Beta2HelmCharts {
				if prevChart.GetReleaseName() != newChart.GetReleaseName() {
					continue
				}
				if !prevChart.Spec.UseHelmInstall {
					return "", errors.Errorf("cannot upgrade chart release %s to v1beta2 because useHelmInstall is false", newChart.GetReleaseName())
				}
			}
		}
	}

	log.ActionWithSpinner("Creating base")
	io.WriteString(pullOptions.ReportWriter, "Creating base\n")

	commonBase, helmBases, err := base.RenderUpstream(u, &renderOptions, renderedKotsKinds)
	if err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to render upstream")
	}

	errorFiles := []base.BaseFile{}
	errorFiles = append(errorFiles, base.PrependBaseFilesPath(commonBase.ListErrorFiles(), commonBase.Path)...)
	for _, helmBase := range helmBases {
		errorFiles = append(errorFiles, base.PrependBaseFilesPath(helmBase.ListErrorFiles(), helmBase.Path)...)
	}

	if ff := errorFiles; len(ff) > 0 {
		files := make([]kotsv1beta1.InstallationYAMLError, 0, len(ff))
		for _, f := range ff {
			file := kotsv1beta1.InstallationYAMLError{
				Path: f.Path,
			}
			if f.Error != nil {
				file.Error = f.Error.Error()
			}
			files = append(files, file)
		}

		renderedKotsKinds.Installation.Spec.YAMLErrors = files

		if err := kotsutil.SaveInstallation(&renderedKotsKinds.Installation, u.GetUpstreamDir(writeUpstreamOptions)); err != nil {
			log.FinishSpinnerWithError()
			return "", errors.Wrap(err, "failed to save installation")
		}
	}

	writeBaseOptions := base.WriteOptions{
		BaseDir:          u.GetBaseDir(writeUpstreamOptions),
		SkippedDir:       u.GetSkippedDir(writeUpstreamOptions),
		Overwrite:        true,
		ExcludeKotsKinds: pullOptions.ExcludeKotsKinds,
		IsHelmBase:       false,
	}
	if err := commonBase.WriteBase(writeBaseOptions); err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to write common base")
	}

	for _, helmBase := range helmBases {
		helmBaseCopy := helmBase.DeepCopy()
		// strip namespace. helm render takes care of injecting the namespace
		helmBaseCopy.SetNamespace("")
		writeBaseOptions := base.WriteOptions{
			BaseDir:          u.GetBaseDir(writeUpstreamOptions),
			SkippedDir:       u.GetSkippedDir(writeUpstreamOptions),
			Overwrite:        true,
			ExcludeKotsKinds: pullOptions.ExcludeKotsKinds,
			IsHelmBase:       true,
		}
		if err := helmBaseCopy.WriteBase(writeBaseOptions); err != nil {
			log.FinishSpinnerWithError()
			return "", errors.Wrapf(err, "failed to write helm base %s", helmBaseCopy.Path)
		}
	}

	log.FinishSpinner()

	log.ActionWithSpinner("Creating midstreams")
	io.WriteString(pullOptions.ReportWriter, "Creating midstreams\n")

	builder, _, err := base.NewConfigContextTemplateBuilder(u, &renderOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to create new config context template builder")
	}

	if err := crypto.InitFromString(renderedKotsKinds.Installation.Spec.EncryptionKey); err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to load encryption cipher")
	}

	commonWriteMidstreamOptions := midstream.WriteOptions{
		AppSlug:            pullOptions.AppSlug,
		IsGitOps:           pullOptions.IsGitOps,
		IsOpenShift:        k8sutil.IsOpenShift(clientset),
		Builder:            *builder,
		HTTPProxyEnvValue:  pullOptions.HTTPProxyEnvValue,
		HTTPSProxyEnvValue: pullOptions.HTTPSProxyEnvValue,
		NoProxyEnvValue:    pullOptions.NoProxyEnvValue,
		NewHelmCharts:      v1Beta1HelmCharts,
		License:            fetchOptions.License,
		RenderedKotsKinds:  renderedKotsKinds,
		IdentityConfig:     identityConfig,
		UpstreamDir:        u.GetUpstreamDir(writeUpstreamOptions),
		Log:                log,
	}

	writeMidstreamOptions := commonWriteMidstreamOptions
	writeMidstreamOptions.MidstreamDir = filepath.Join(u.GetOverlaysDir(writeUpstreamOptions), "midstream")
	writeMidstreamOptions.BaseDir = filepath.Join(u.GetBaseDir(writeUpstreamOptions), commonBase.Path)
	writeMidstreamOptions.ProcessImageOptions = processImageOptions
	writeMidstreamOptions.Base = commonBase

	m, err := midstream.WriteMidstream(writeMidstreamOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to write common midstream")
	}

	helmMidstreams := []midstream.Midstream{}
	for _, helmBase := range helmBases {
		helmBaseCopy := helmBase.DeepCopy()

		processImageOptionsCopy := processImageOptions
		processImageOptionsCopy.Namespace = helmBaseCopy.Namespace
		processImageOptionsCopy.PushImages = false // never push images more than once

		writeMidstreamOptions.MidstreamDir = filepath.Join(u.GetOverlaysDir(writeUpstreamOptions), "midstream", helmBaseCopy.Path)
		writeMidstreamOptions.BaseDir = filepath.Join(u.GetBaseDir(writeUpstreamOptions), helmBaseCopy.Path)
		writeMidstreamOptions.ProcessImageOptions = processImageOptionsCopy
		writeMidstreamOptions.Base = helmBaseCopy

		helmMidstream, err := midstream.WriteMidstream(writeMidstreamOptions)
		if err != nil {
			log.FinishSpinnerWithError()
			return "", errors.Wrapf(err, "failed to write helm midstream %s", helmBase.Path)
		}

		helmMidstreams = append(helmMidstreams, *helmMidstream)
	}

	err = removeUnusedHelmOverlays(writeMidstreamOptions.MidstreamDir, writeMidstreamOptions.BaseDir)
	if err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrapf(err, "failed to remove unused helm midstreams")
	}

	writeV1Beta2HelmChartsOpts := apparchive.WriteV1Beta2HelmChartsOptions{
		Upstream:             u,
		WriteUpstreamOptions: writeUpstreamOptions,
		RenderOptions:        &renderOptions,
		ProcessImageOptions:  processImageOptions,
		KotsKinds:            renderedKotsKinds,
		Clientset:            clientset,
	}

	if err := apparchive.WriteV1Beta2HelmCharts(writeV1Beta2HelmChartsOpts); err != nil {
		return "", errors.Wrap(err, "failed to write v1beta2 helm charts")
	}

	log.FinishSpinner()

	if err := writeDownstreams(pullOptions, u.GetOverlaysDir(writeUpstreamOptions), m, helmMidstreams, log); err != nil {
		return "", errors.Wrap(err, "failed to write downstreams")
	}

	if includeAdminConsole {
		if err := writeArchiveAsConfigMap(pullOptions, u, u.GetBaseDir(writeUpstreamOptions)); err != nil {
			return "", errors.Wrap(err, "failed to write archive as config map")
		}
	}

	// installation spec gets updated during the process, ensure the map has the latest version
	installationBytes, err := os.ReadFile(filepath.Join(u.GetUpstreamDir(writeUpstreamOptions), "userdata", "installation.yaml"))
	if err != nil {
		return "", errors.Wrap(err, "failed to read installation file")
	}
	renderedKotsKindsMap["userdata/installation.yaml"] = []byte(installationBytes)

	if err := kotsutil.WriteKotsKinds(renderedKotsKindsMap, u.GetKotsKindsDir(writeUpstreamOptions)); err != nil {
		return "", errors.Wrap(err, "failed to write the rendered kots kinds")
	}

	if err := rendered.WriteRenderedApp(&rendered.WriteOptions{
		BaseDir:             u.GetBaseDir(writeUpstreamOptions),
		OverlaysDir:         u.GetOverlaysDir(writeUpstreamOptions),
		RenderedDir:         u.GetRenderedDir(writeUpstreamOptions),
		Downstreams:         pullOptions.Downstreams,
		KustomizeBinPath:    renderedKotsKinds.GetKustomizeBinaryPath(),
		HelmDir:             u.GetHelmDir(writeUpstreamOptions),
		Log:                 log,
		KotsKinds:           renderedKotsKinds,
		ProcessImageOptions: processImageOptions,
		Clientset:           clientset,
	}); err != nil {
		return "", errors.Wrap(err, "failed to write rendered")
	}

	return filepath.Join(pullOptions.RootDir, u.Name), nil
}

func removeUnusedHelmOverlays(overlayRoot string, baseRoot string) error {
	// Only cleanup "charts" subdirectory. This can be isolated from customer overlays, so we don't destroy them.
	return removeUnusedHelmOverlaysRec(overlayRoot, baseRoot, "charts")
}

func removeUnusedHelmOverlaysRec(overlayRoot string, baseRoot string, overlayRelDir string) error {
	curMidstreamDir := filepath.Join(overlayRoot, overlayRelDir)
	midstreamFiles, err := ioutil.ReadDir(curMidstreamDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrap(err, "failed to read midstream dir")
	}

	for _, midstreamFile := range midstreamFiles {
		if !midstreamFile.IsDir() {
			continue
		}

		midstreamPath := filepath.Join(curMidstreamDir, midstreamFile.Name())
		relPath, err := filepath.Rel(overlayRoot, midstreamPath)
		if err != nil {
			return errors.Wrap(err, "failed to get relative midstream path")
		}
		basePath := filepath.Join(baseRoot, relPath)

		_, err = os.Stat(basePath)
		if err == nil {
			err := removeUnusedHelmOverlaysRec(overlayRoot, baseRoot, relPath)
			if err != nil {
				return err // not wrapping recursive errors
			}
		}
		if os.IsNotExist(err) {
			// File is in midstream, but is no longer in base
			err := os.RemoveAll(midstreamPath)
			if err != nil {
				return errors.Wrapf(err, "failed to remove %s from midtream", relPath)
			}
			continue
		}
		return errors.Wrap(err, "failed to stat base file")
	}

	return nil
}

func writeDownstreams(options PullOptions, overlaysDir string, m *midstream.Midstream, helmMidstreams []midstream.Midstream, log *logger.CLILogger) error {
	for _, downstreamName := range options.Downstreams {
		log.ActionWithSpinner("Creating downstream %q", downstreamName)
		io.WriteString(options.ReportWriter, fmt.Sprintf("Creating downstream %q\n", downstreamName))

		d, err := downstream.CreateDownstream(m)
		if err != nil {
			return errors.Wrapf(err, "failed to create downstream %s", downstreamName)
		}

		writeDownstreamOptions := downstream.WriteOptions{
			DownstreamDir: filepath.Join(overlaysDir, "downstreams", downstreamName),
			MidstreamDir:  filepath.Join(overlaysDir, "midstream"),
		}
		if err := d.WriteDownstream(writeDownstreamOptions); err != nil {
			return errors.Wrapf(err, "failed to write downstream %s", downstreamName)
		}

		for _, mid := range helmMidstreams {
			helmMidstream := mid // bug? this object contains pointers which are not deep-copied here

			d, err := downstream.CreateDownstream(&helmMidstream)
			if err != nil {
				return errors.Wrapf(err, "failed to create downstream %s for helm midstream %s", downstreamName, mid.Base.Path)
			}

			writeDownstreamOptions := downstream.WriteOptions{
				DownstreamDir: filepath.Join(overlaysDir, "downstreams", downstreamName, mid.Base.Path),
				MidstreamDir:  filepath.Join(overlaysDir, "midstream", mid.Base.Path),
			}
			if err := d.WriteDownstream(writeDownstreamOptions); err != nil {
				return errors.Wrapf(err, "failed to write downstream %s for helm midstream %s", downstreamName, mid.Base.Path)
			}
		}

		err = removeUnusedHelmOverlays(writeDownstreamOptions.DownstreamDir, writeDownstreamOptions.MidstreamDir)
		if err != nil {
			return errors.Wrapf(err, "failed to remove unused helm downstreams")
		}

		log.FinishSpinner()
	}

	return nil
}

func ParseConfigValuesFromFile(filename string) (*kotsv1beta1.ConfigValues, error) {
	contents, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to read config values file")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	decoded, gvk, err := decode(contents, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode config values file")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "ConfigValues" {
		return nil, errors.New("not config values")
	}

	config := decoded.(*kotsv1beta1.ConfigValues)

	return config, nil
}

func ParseIdentityConfigFromFile(filename string) (*kotsv1beta1.IdentityConfig, error) {
	contents, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to read identity config file")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	decoded, gvk, err := decode(contents, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode identity config file")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "IdentityConfig" {
		return nil, errors.New("not identity config")
	}

	identityConfig := decoded.(*kotsv1beta1.IdentityConfig)

	return identityConfig, nil
}

func GetAppMetadataFromAirgap(airgapArchive string) (*replicatedapp.ApplicationMetadata, error) {
	appArchive, err := archives.GetFileFromAirgap("app.tar.gz", airgapArchive)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract app archive")
	}

	tempDir, err := os.MkdirTemp("", "kotsadm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tempDir)

	err = archives.ExtractTGZArchiveFromReader(bytes.NewReader(appArchive), tempDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract app archive")
	}

	kotsApp, err := kotsutil.FindKotsAppInPath(tempDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read kots kinds")
	}
	if kotsApp == nil {
		ka := kotsutil.EmptyKotsKinds().KotsApplication
		kotsApp = &ka
	}

	s := k8sjson.NewYAMLSerializer(k8sjson.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(kotsApp, &b); err != nil {
		return nil, errors.Wrap(err, "failed to encode metadata")
	}

	branding, err := kotsutil.LoadBrandingArchiveFromPath(tempDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load branding archive")
	}

	return &replicatedapp.ApplicationMetadata{
		Manifest: b.Bytes(),
		Branding: branding.Bytes(),
	}, nil
}

func parseInstallationFromFile(filename string) (*kotsv1beta1.Installation, error) {
	contents, err := os.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to read installation file")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	decoded, gvk, err := decode(contents, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode installation file")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Installation" {
		return nil, errors.New("not installation file")
	}

	installation := decoded.(*kotsv1beta1.Installation)

	return installation, nil
}

func publicKeysMatch(log *logger.CLILogger, license *kotsv1beta1.License, airgap *kotsv1beta1.Airgap) error {
	if license == nil || airgap == nil {
		// not sure when this would happen, but earlier logic allows this combination
		return nil
	}

	publicKey, err := kotslicense.GetAppPublicKey(license)
	if err != nil {
		return errors.Wrap(err, "failed to get public key from license")
	}

	if err := kotslicense.Verify([]byte(license.Spec.AppSlug), []byte(airgap.Spec.Signature), publicKey); err != nil {
		log.Info("got error validating airgap bundle: %s", err.Error())
		if airgap.Spec.AppSlug != "" {
			return util.ActionableError{
				NoRetry: true,
				Message: fmt.Sprintf("Failed to verify bundle signature - license is for app %q, airgap package for app %q", license.Spec.AppSlug, airgap.Spec.AppSlug),
			}
		} else {
			return util.ActionableError{
				NoRetry: true,
				Message: fmt.Sprintf("Failed to verify bundle signature - airgap package does not match license app %q", license.Spec.AppSlug),
			}
		}
	}

	return nil
}

func findConfig(localPath string) (*kotsv1beta1.Config, *kotsv1beta1.ConfigValues, *kotsv1beta1.License, *kotsv1beta1.Installation, *kotsv1beta1.IdentityConfig, error) {
	if localPath == "" {
		return nil, nil, nil, nil, nil, nil
	}

	var config *kotsv1beta1.Config
	var values *kotsv1beta1.ConfigValues
	var license *kotsv1beta1.License
	var installation *kotsv1beta1.Installation
	var identityConfig *kotsv1beta1.IdentityConfig

	err := filepath.Walk(localPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			decode := scheme.Codecs.UniversalDeserializer().Decode
			obj, gvk, err := decode(content, nil, nil)
			if err != nil {
				return nil
			}

			if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Config" {
				config = obj.(*kotsv1beta1.Config)
			} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "ConfigValues" {
				values = obj.(*kotsv1beta1.ConfigValues)
			} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "License" {
				license = obj.(*kotsv1beta1.License)
			} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Installation" {
				installation = obj.(*kotsv1beta1.Installation)
			} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "IdentityConfig" {
				identityConfig = obj.(*kotsv1beta1.IdentityConfig)
			}

			return nil
		})

	if err != nil {
		return nil, nil, nil, nil, nil, errors.Wrap(err, "failed to walk local dir")
	}

	return config, values, license, installation, identityConfig, nil
}
