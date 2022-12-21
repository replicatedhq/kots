package pull

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/archives"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	dockerregistrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/downstream"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadmconfig"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	"github.com/replicatedhq/kots/pkg/upstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
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
	RewriteImageOptions     RewriteImageOptions
	SkipHelmChartCheck      bool
	ReportWriter            io.Writer
	AppSlug                 string
	AppSequence             int64
	AppVersionLabel         string
	IsGitOps                bool
	HTTPProxyEnvValue       string
	HTTPSProxyEnvValue      string
	NoProxyEnvValue         string
	ReportingInfo           *reportingtypes.ReportingInfo
	SkipCompatibilityCheck  bool
}

type RewriteImageOptions struct {
	Host       string
	Namespace  string
	Username   string
	Password   string
	IsReadOnly bool
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
		RootDir:         pullOptions.RootDir,
		UseAppDir:       pullOptions.CreateAppDir,
		LocalPath:       pullOptions.LocalPath,
		CurrentCursor:   pullOptions.UpdateCursor,
		AppSlug:         pullOptions.AppSlug,
		AppSequence:     pullOptions.AppSequence,
		AppVersionLabel: pullOptions.AppVersionLabel,
		LocalRegistry: upstreamtypes.LocalRegistry{
			Host:      pullOptions.RewriteImageOptions.Host,
			Namespace: pullOptions.RewriteImageOptions.Namespace,
			Username:  pullOptions.RewriteImageOptions.Username,
			Password:  pullOptions.RewriteImageOptions.Password,
			ReadOnly:  pullOptions.RewriteImageOptions.IsReadOnly,
		},
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

		airgapAppFiles, err := ioutil.TempDir("", "airgap-kots")
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
		IncludeMinio:        pullOptions.IncludeMinio,
	}
	if err := upstream.WriteUpstream(u, writeUpstreamOptions); err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to write upstream")
	}
	log.FinishSpinner()

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(u.GetUpstreamDir(writeUpstreamOptions))
	if err != nil {
		return "", errors.Wrap(err, "failed to load kotskinds")
	}

	registrySettings := registrytypes.RegistrySettings{
		Hostname:   pullOptions.RewriteImageOptions.Host,
		Namespace:  pullOptions.RewriteImageOptions.Namespace,
		Username:   pullOptions.RewriteImageOptions.Username,
		Password:   pullOptions.RewriteImageOptions.Password,
		IsReadOnly: pullOptions.RewriteImageOptions.IsReadOnly,
	}

	pushImages := pullOptions.RewriteImageOptions.Host != ""

	needsConfig, err := kotsadmconfig.NeedsConfiguration(pullOptions.AppSlug, pullOptions.AppSequence, pullOptions.AirgapRoot != "", kotsKinds, registrySettings)
	if err != nil {
		return "", errors.Wrap(err, "failed to check if version needs configuration")
	}
	if needsConfig {
		if pullOptions.AirgapRoot != "" {
			// if this is an airgap install, we still need to process the images
			if _, err = processAirgapImages(pullOptions, pushImages, kotsKinds, fetchOptions.License, log); err != nil {
				return "", errors.Wrap(err, "failed to process airgap images")
			}
		}

		return "", ErrConfigNeeded
	}

	renderDir := pullOptions.RootDir
	if pullOptions.CreateAppDir {
		renderDir = filepath.Join(pullOptions.RootDir, u.Name)
	}

	newHelmCharts, err := kotsutil.LoadHelmChartsFromPath(renderDir)
	if err != nil {
		return "", errors.Wrap(err, "failed to load new helm charts")
	}

	if !pullOptions.SkipHelmChartCheck {
		prevHelmCharts, err := kotsutil.LoadHelmChartsFromPath(pullOptions.RootDir)
		if err != nil {
			return "", errors.Wrap(err, "failed to load previous helm charts")
		}

		for _, prevChart := range prevHelmCharts {
			if !prevChart.Spec.Exclude.IsEmpty() {
				isExcluded, err := prevChart.Spec.Exclude.Boolean()
				if err == nil && isExcluded {
					continue // this chart was excluded, so any changes to UseHelmInstall are irrelevant
				}
			}

			for _, newChart := range newHelmCharts {
				if prevChart.Spec.Chart.Name != newChart.Spec.Chart.Name {
					continue
				}
				if prevChart.Spec.UseHelmInstall != newChart.Spec.UseHelmInstall {
					return "", errors.Errorf("deployment method for chart %s has changed", newChart.Spec.Chart.Name)
				}
			}
		}
	}

	renderOptions := base.RenderOptions{
		SplitMultiDocYAML:       true,
		Namespace:               pullOptions.Namespace,
		LocalRegistryHost:       pullOptions.RewriteImageOptions.Host,
		LocalRegistryNamespace:  pullOptions.RewriteImageOptions.Namespace,
		LocalRegistryUsername:   pullOptions.RewriteImageOptions.Username,
		LocalRegistryPassword:   pullOptions.RewriteImageOptions.Password,
		LocalRegistryIsReadOnly: pullOptions.RewriteImageOptions.IsReadOnly,
		ExcludeKotsKinds:        pullOptions.ExcludeKotsKinds,
		Log:                     log,
		AppSlug:                 pullOptions.AppSlug,
		Sequence:                pullOptions.AppSequence,
		IsAirgap:                pullOptions.AirgapRoot != "",
	}
	log.ActionWithSpinner("Creating base")
	io.WriteString(pullOptions.ReportWriter, "Creating base\n")

	commonBase, helmBases, err := base.RenderUpstream(u, &renderOptions)
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

		newKotsKinds, err := kotsutil.LoadKotsKindsFromPath(u.GetUpstreamDir(writeUpstreamOptions))
		if err != nil {
			return "", errors.Wrap(err, "failed to load kotskinds")
		}
		newKotsKinds.Installation.Spec.YAMLErrors = files

		err = upstream.SaveInstallation(&newKotsKinds.Installation, u.GetUpstreamDir(writeUpstreamOptions))
		if err != nil {
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

	newKotsKinds, err := kotsutil.LoadKotsKindsFromPath(u.GetUpstreamDir(writeUpstreamOptions))
	if err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to load kotskinds")
	}

	err = crypto.InitFromString(newKotsKinds.Installation.Spec.EncryptionKey)
	if err != nil {
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
		NewHelmCharts:      newHelmCharts,
	}

	// the UseHelmInstall map blocks visibility into charts and subcharts when searching for private images
	// any chart name listed here will be skipped when writing midstream kustomization.yaml and pullsecret.yaml
	// when using Helm Install, each chart gets it's own kustomization and pullsecret yaml and MUST be skipped when processing higher level directories!
	// for writing Common Midstream, every chart and subchart is in this map as Helm Midstreams will be processed later in the code
	commonWriteMidstreamOptions.UseHelmInstall = map[string]bool{}
	for _, v := range newHelmCharts {
		chartBaseName := v.GetDirName()
		commonWriteMidstreamOptions.UseHelmInstall[chartBaseName] = v.Spec.UseHelmInstall
		if v.Spec.UseHelmInstall {
			subcharts, err := base.FindHelmSubChartsFromBase(writeBaseOptions.BaseDir, chartBaseName)
			if err != nil {
				log.FinishSpinnerWithError()
				return "", errors.Wrapf(err, "failed to find subcharts for parent chart %s", chartBaseName)
			}
			for _, subchart := range subcharts.SubCharts {
				commonWriteMidstreamOptions.UseHelmInstall[subchart] = v.Spec.UseHelmInstall
			}
		}
	}

	writeMidstreamOptions := commonWriteMidstreamOptions
	writeMidstreamOptions.MidstreamDir = filepath.Join(commonBase.GetOverlaysDir(writeBaseOptions), "midstream")
	writeMidstreamOptions.BaseDir = filepath.Join(u.GetBaseDir(writeUpstreamOptions), commonBase.Path)

	m, err := writeMidstream(writeMidstreamOptions, pullOptions, commonBase, fetchOptions.License, identityConfig, u.GetUpstreamDir(writeUpstreamOptions), pushImages, log)
	if err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to write common midstream")
	}

	helmMidstreams := []midstream.Midstream{}
	for _, helmBase := range helmBases {
		// we must look at the current chart for private images, but must ignore subcharts
		// to do this, we remove only the current helmBase name from the UseHelmInstall map to unblock visibility into the chart directory
		// this ensures only the current chart resources are added to kustomization.yaml and pullsecret.yaml
		// copy the bool setting in the map to restore it after this process loop
		previousUseHelmInstall := writeMidstreamOptions.UseHelmInstall[helmBase.Path]
		writeMidstreamOptions.UseHelmInstall[helmBase.Path] = false

		writeMidstreamOptions.MidstreamDir = filepath.Join(helmBase.GetOverlaysDir(writeBaseOptions), "midstream", helmBase.Path)
		writeMidstreamOptions.BaseDir = filepath.Join(u.GetBaseDir(writeUpstreamOptions), helmBase.Path)
		pushImages = false // never push images more than once

		helmBaseCopy := helmBase.DeepCopy()

		pullOptionsCopy := pullOptions
		pullOptionsCopy.Namespace = helmBaseCopy.Namespace

		helmMidstream, err := writeMidstream(writeMidstreamOptions, pullOptionsCopy, helmBaseCopy, fetchOptions.License, identityConfig, u.GetUpstreamDir(writeUpstreamOptions), pushImages, log)
		if err != nil {
			log.FinishSpinnerWithError()
			return "", errors.Wrapf(err, "failed to write helm midstream %s", helmBase.Path)
		}

		// add this chart back into UseHelmInstall to make sure it's not processed again
		writeMidstreamOptions.UseHelmInstall[helmBase.Path] = previousUseHelmInstall

		helmMidstreams = append(helmMidstreams, *helmMidstream)
	}

	err = removeUnusedHelmOverlays(writeMidstreamOptions.MidstreamDir, writeMidstreamOptions.BaseDir)
	if err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrapf(err, "failed to remove unused helm midstreams")
	}

	log.FinishSpinner()

	if err := writeDownstreams(pullOptions, commonBase.GetOverlaysDir(writeBaseOptions), m, helmMidstreams, log); err != nil {
		return "", errors.Wrap(err, "failed to write downstreams")
	}

	if includeAdminConsole {
		if err := writeArchiveAsConfigMap(pullOptions, u, u.GetBaseDir(writeUpstreamOptions)); err != nil {
			return "", errors.Wrap(err, "failed to write archive as config map")
		}
	}

	return filepath.Join(pullOptions.RootDir, u.Name), nil
}

func writeMidstream(writeMidstreamOptions midstream.WriteOptions, options PullOptions, b *base.Base, license *kotsv1beta1.License, identityConfig *kotsv1beta1.IdentityConfig, upstreamDir string, pushImages bool, log *logger.CLILogger) (*midstream.Midstream, error) {
	var images []kustomizetypes.Image
	var objects []k8sdoc.K8sDoc
	var pullSecretRegistries []string
	var pullSecretUsername string
	var pullSecretPassword string

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	newKotsKinds, err := kotsutil.LoadKotsKindsFromPath(upstreamDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load kotskinds from new upstream")
	}

	identitySpec, err := upstream.LoadIdentity(upstreamDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load identity")
	}

	// do not fail on being unable to get dockerhub credentials, since they're just used to increase the rate limit
	var dockerHubRegistryCreds registry.Credentials
	dockerhubSecret, _ := registry.GetDockerHubPullSecret(clientset, util.PodNamespace, options.Namespace, options.AppSlug)
	if dockerhubSecret != nil {
		dockerHubRegistryCreds, _ = registry.GetCredentialsForRegistryFromConfigJSON(dockerhubSecret.Data[".dockerconfigjson"], registry.DockerHubRegistryName)
	}

	if options.RewriteImages {
		// A target registry is configured. Rewrite all images and copy them (if necessary) to the configured registry.
		if options.RewriteImageOptions.IsReadOnly {
			log.ActionWithSpinner("Rewriting images")
			io.WriteString(options.ReportWriter, "Rewriting images\n")
		} else {
			log.ActionWithSpinner("Copying images")
			io.WriteString(options.ReportWriter, "Copying images\n")
		}

		if options.AirgapRoot == "" {
			// This is an online installation. Pull and rewrite images from online and copy them (if necessary) to the configured registry.
			rewriteResult, err := rewriteBaseImages(options, writeMidstreamOptions.BaseDir, newKotsKinds, license, dockerHubRegistryCreds, log)
			if err != nil {
				return nil, errors.Wrap(err, "failed to rewrite base images")
			}
			images = rewriteResult.Images
			newKotsKinds.Installation.Spec.KnownImages = rewriteResult.CheckedImages
		} else {
			// This is an airgapped installation. Copy and rewrite images from the airgap bundle to the configured registry.
			result, err := processAirgapImages(options, pushImages, newKotsKinds, license, log)
			if err != nil {
				return nil, errors.Wrap(err, "failed to process airgap images")
			}
			images = result.KustomizeImages
			newKotsKinds.Installation.Spec.KnownImages = result.KnownImages
		}

		objects = base.FindObjectsWithImages(b)

		// Use target registry credentials to create image pull secrets for all objects that have images.
		pullSecretRegistries = []string{options.RewriteImageOptions.Host}
		pullSecretUsername = options.RewriteImageOptions.Username
		pullSecretPassword = options.RewriteImageOptions.Password
		if pullSecretUsername == "" {
			pullSecretUsername, pullSecretPassword, err = registry.LoadAuthForRegistry(options.RewriteImageOptions.Host)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load registry auth for %q", options.RewriteImageOptions.Host)
			}
		}
	} else if license != nil {
		// A target registry is NOT configured. Find and rewrite private images to be proxied through proxy.replicated.com
		findResult, err := findPrivateImages(writeMidstreamOptions, b, newKotsKinds, license, dockerHubRegistryCreds)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find private images")
		}
		images = findResult.Images
		newKotsKinds.Installation.Spec.KnownImages = findResult.CheckedImages
		objects = findResult.Docs

		// Use license to create image pull secrets for all objects that have private images.
		pullSecretRegistries = registry.GetRegistryProxyInfo(license, &newKotsKinds.KotsApplication).ToSlice()
		pullSecretUsername = license.Spec.LicenseID
		pullSecretPassword = license.Spec.LicenseID
	}

	// For the newer style charts, create a new secret per chart as helm adds chart specific
	// details to annotations and labels to it.
	namePrefix := options.AppSlug
	for _, v := range writeMidstreamOptions.NewHelmCharts {
		if v.Spec.UseHelmInstall && filepath.Base(b.Path) != "." {
			namePrefix = fmt.Sprintf("%s-%s", options.AppSlug, filepath.Base(b.Path))
			break
		}
	}
	pullSecrets, err := registry.PullSecretForRegistries(
		pullSecretRegistries,
		pullSecretUsername,
		pullSecretPassword,
		options.Namespace,
		namePrefix,
	)
	if err != nil {
		return nil, errors.Wrap(err, "create pull secret")
	}
	pullSecrets.DockerHubSecret = dockerhubSecret

	if err := upstream.SaveInstallation(&newKotsKinds.Installation, upstreamDir); err != nil {
		return nil, errors.Wrap(err, "failed to save installation")
	}

	m, err := midstream.CreateMidstream(b, images, objects, &pullSecrets, identitySpec, identityConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create midstream")
	}

	if err := m.WriteMidstream(writeMidstreamOptions); err != nil {
		return nil, errors.Wrap(err, "failed to write common midstream")
	}

	return m, nil
}

// rewriteBaseImages Will rewrite images found in base and copy them (if necessary) to the configured registry.
func rewriteBaseImages(pullOptions PullOptions, baseDir string, kotsKinds *kotsutil.KotsKinds, license *kotsv1beta1.License, dockerHubRegistryCreds registry.Credentials, log *logger.CLILogger) (*base.RewriteImagesResult, error) {
	replicatedRegistryInfo := registry.GetRegistryProxyInfo(license, &kotsKinds.KotsApplication)

	rewriteImageOptions := base.RewriteImageOptions{
		BaseDir: baseDir,
		Log:     log,
		SourceRegistry: dockerregistrytypes.RegistryOptions{
			Endpoint:      replicatedRegistryInfo.Registry,
			ProxyEndpoint: replicatedRegistryInfo.Proxy,
		},
		DockerHubRegistry: dockerregistrytypes.RegistryOptions{
			Username: dockerHubRegistryCreds.Username,
			Password: dockerHubRegistryCreds.Password,
		},
		DestRegistry: dockerregistrytypes.RegistryOptions{
			Endpoint:  pullOptions.RewriteImageOptions.Host,
			Namespace: pullOptions.RewriteImageOptions.Namespace,
			Username:  pullOptions.RewriteImageOptions.Username,
			Password:  pullOptions.RewriteImageOptions.Password,
		},
		ReportWriter: pullOptions.ReportWriter,
		KotsKinds:    kotsKinds,
		CopyImages:   !pullOptions.RewriteImageOptions.IsReadOnly,
	}
	if license != nil {
		rewriteImageOptions.AppSlug = license.Spec.AppSlug
		rewriteImageOptions.SourceRegistry.Username = license.Spec.LicenseID
		rewriteImageOptions.SourceRegistry.Password = license.Spec.LicenseID
	}

	rewriteResult, err := base.RewriteImages(rewriteImageOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to rewrite images")
	}

	return rewriteResult, nil
}

// processAirgapImages Will rewrite images found in the airgap bundle/airgap root and copy them (if necessary) to the configured registry.
func processAirgapImages(pullOptions PullOptions, pushImages bool, kotsKinds *kotsutil.KotsKinds, license *kotsv1beta1.License, log *logger.CLILogger) (*upstream.ProcessAirgapImagesResult, error) {
	replicatedRegistryInfo := registry.GetRegistryProxyInfo(license, &kotsKinds.KotsApplication)

	processAirgapImageOptions := upstream.ProcessAirgapImagesOptions{
		RootDir:      pullOptions.RootDir,
		AirgapRoot:   pullOptions.AirgapRoot,
		AirgapBundle: pullOptions.AirgapBundle,
		CreateAppDir: pullOptions.CreateAppDir,
		PushImages:   !pullOptions.RewriteImageOptions.IsReadOnly && pushImages,
		Log:          log,
		ReplicatedRegistry: dockerregistrytypes.RegistryOptions{
			Endpoint:      replicatedRegistryInfo.Registry,
			ProxyEndpoint: replicatedRegistryInfo.Proxy,
		},
		ReportWriter: pullOptions.ReportWriter,
		DestinationRegistry: dockerregistrytypes.RegistryOptions{
			Endpoint:  pullOptions.RewriteImageOptions.Host,
			Namespace: pullOptions.RewriteImageOptions.Namespace,
			Username:  pullOptions.RewriteImageOptions.Username,
			Password:  pullOptions.RewriteImageOptions.Password,
		},
	}
	if license != nil {
		processAirgapImageOptions.ReplicatedRegistry.Username = license.Spec.LicenseID
		processAirgapImageOptions.ReplicatedRegistry.Password = license.Spec.LicenseID
	}

	imagesData, err := ioutil.ReadFile(filepath.Join(pullOptions.AirgapRoot, "images.json"))
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "failed to load images file")
	}
	if err == nil {
		var images []kustomizetypes.Image
		err := json.Unmarshal(imagesData, &images)
		if err != nil && !os.IsNotExist(err) {
			return nil, errors.Wrap(err, "failed to unmarshal images data")
		}
		processAirgapImageOptions.UseKnownImages = true
		processAirgapImageOptions.KnownImages = images
	}

	result, err := upstream.ProcessAirgapImages(processAirgapImageOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to process airgap images")
	}

	return result, nil
}

// findPrivateImages Finds and rewrites private images to be proxied through proxy.replicated.com
func findPrivateImages(writeMidstreamOptions midstream.WriteOptions, b *base.Base, kotsKinds *kotsutil.KotsKinds, license *kotsv1beta1.License, dockerHubRegistryCreds registry.Credentials) (*base.FindPrivateImagesResult, error) {
	replicatedRegistryInfo := registry.GetRegistryProxyInfo(license, &kotsKinds.KotsApplication)
	allPrivate := kotsKinds.KotsApplication.Spec.ProxyPublicImages

	findPrivateImagesOptions := base.FindPrivateImagesOptions{
		BaseDir: writeMidstreamOptions.BaseDir,
		AppSlug: license.Spec.AppSlug,
		ReplicatedRegistry: dockerregistrytypes.RegistryOptions{
			Endpoint:      replicatedRegistryInfo.Registry,
			ProxyEndpoint: replicatedRegistryInfo.Proxy,
		},
		DockerHubRegistry: dockerregistrytypes.RegistryOptions{
			Username: dockerHubRegistryCreds.Username,
			Password: dockerHubRegistryCreds.Password,
		},
		Installation:     &kotsKinds.Installation,
		AllImagesPrivate: allPrivate,
		HelmChartPath:    b.Path,
		UseHelmInstall:   writeMidstreamOptions.UseHelmInstall,
		KotsKindsImages:  kotsutil.GetImagesFromKotsKinds(kotsKinds),
	}
	findResult, err := base.FindPrivateImages(findPrivateImagesOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find private images")
	}

	return findResult, nil
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

func writeCombinedDownstreamBase(downstreamName string, bases []string, renderDir string) error {
	if _, err := os.Stat(renderDir); os.IsNotExist(err) {
		if err := os.MkdirAll(renderDir, 0744); err != nil {
			return errors.Wrap(err, "failed to mkdir")
		}
	}

	kustomization := kustomizetypes.Kustomization{
		TypeMeta: kustomizetypes.TypeMeta{
			APIVersion: "kustomize.config.k8s.io/v1beta1",
			Kind:       "Kustomization",
		},
		Bases: bases,
	}
	if err := k8sutil.WriteKustomizationToFile(kustomization, filepath.Join(renderDir, "kustomization.yaml")); err != nil {
		return errors.Wrap(err, "failed to write kustomization to file")
	}

	return nil
}

func ParseConfigValuesFromFile(filename string) (*kotsv1beta1.ConfigValues, error) {
	contents, err := ioutil.ReadFile(filename)
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
	contents, err := ioutil.ReadFile(filename)
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

	tempDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tempDir)

	err = archives.ExtractTGZArchiveFromReader(bytes.NewReader(appArchive), tempDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract app archive")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(tempDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read kots kinds")
	}

	s := k8sjson.NewYAMLSerializer(k8sjson.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(&kotsKinds.KotsApplication, &b); err != nil {
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
	contents, err := ioutil.ReadFile(filename)
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

			content, err := ioutil.ReadFile(path)
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
