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
	"github.com/replicatedhq/kots/pkg/downstream"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotslicense "github.com/replicatedhq/kots/pkg/license"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	"github.com/replicatedhq/kots/pkg/upstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

type PullOptions struct {
	HelmRepoURI             string
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
	HelmVersion             string
	HelmOptions             []string
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
	IdentityPostgresConfig  *kotsv1beta1.IdentityPostgresConfig
	SkipCompatibilityCheck  bool
}

type RewriteImageOptions struct {
	ImageFiles string
	Host       string
	Namespace  string
	Username   string
	Password   string
	IsReadOnly bool
}

// PullApplicationMetadata will return the application metadata yaml, if one is
// available for the upstream
func PullApplicationMetadata(upstreamURI string, versionLabel string) ([]byte, error) {
	u, err := url.ParseRequestURI(upstreamURI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse uri")
	}

	// metadata is only currently supported on licensed apps
	if u.Scheme != "replicated" {
		return nil, nil
	}

	data, err := upstream.GetApplicationMetadata(u, versionLabel)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get application metadata")
	}

	return data, nil
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
		HelmRepoURI:     pullOptions.HelmRepoURI,
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

	verifiedLicense, err := kotslicense.VerifySignature(fetchOptions.License)
	if err != nil {
		return "", errors.Wrap(err, "failed to verify signature")
	}
	fetchOptions.License = verifiedLicense

	if pullOptions.LicenseEndpointOverride != "" && fetchOptions.License != nil {
		fetchOptions.License.Spec.Endpoint = pullOptions.LicenseEndpointOverride
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

		airgap, err := FindAirgapMetaInDir(pullOptions.AirgapRoot)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse license from file")
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
		return "", errors.Wrap(err, "failed to fetch upstream")
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
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
		return "", errors.Wrap(err, "failed to write upstream")
	}
	log.FinishSpinner()

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
		HelmVersion:             pullOptions.HelmVersion,
		HelmOptions:             pullOptions.HelmOptions,
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
			return "", errors.Wrapf(err, "failed to write helm base %s", helmBaseCopy.Path)
		}
	}

	log.FinishSpinner()

	log.ActionWithSpinner("Creating midstreams")
	io.WriteString(pullOptions.ReportWriter, "Creating midstreams\n")

	builder, _, err := base.NewConfigContextTemplateBuilder(u, &renderOptions)
	if err != nil {
		return "", errors.Wrap(err, "failed to create new config context template builder")
	}

	newKotsKinds, err := kotsutil.LoadKotsKindsFromPath(u.GetUpstreamDir(writeUpstreamOptions))
	if err != nil {
		return "", errors.Wrap(err, "failed to load kotskinds")
	}

	err = crypto.InitFromString(newKotsKinds.Installation.Spec.EncryptionKey)
	if err != nil {
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
	pushImages := pullOptions.RewriteImageOptions.Host != ""

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

	m, err := writeMidstream(writeMidstreamOptions, pullOptions, u, commonBase, fetchOptions.License, identityConfig, u.GetUpstreamDir(writeUpstreamOptions), pushImages, log)
	if err != nil {
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

		helmMidstream, err := writeMidstream(writeMidstreamOptions, pullOptionsCopy, u, helmBaseCopy, fetchOptions.License, identityConfig, u.GetUpstreamDir(writeUpstreamOptions), pushImages, log)
		if err != nil {
			return "", errors.Wrapf(err, "failed to write helm midstream %s", helmBase.Path)
		}

		// add this chart back into UseHelmInstall to make sure it's not processed again
		writeMidstreamOptions.UseHelmInstall[helmBase.Path] = previousUseHelmInstall

		helmMidstreams = append(helmMidstreams, *helmMidstream)
	}

	err = removeUnusedHelmOverlays(writeMidstreamOptions.MidstreamDir, writeMidstreamOptions.BaseDir)
	if err != nil {
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

func writeMidstream(writeMidstreamOptions midstream.WriteOptions, options PullOptions, u *upstreamtypes.Upstream, b *base.Base, license *kotsv1beta1.License, identityConfig *kotsv1beta1.IdentityConfig, upstreamDir string, pushImages bool, log *logger.CLILogger) (*midstream.Midstream, error) {
	var pullSecrets registry.ImagePullSecrets
	var images []kustomizetypes.Image
	var objects []k8sdoc.K8sDoc

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	replicatedRegistryInfo := registry.ProxyEndpointFromLicense(license)

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

		if options.RewriteImageOptions.IsReadOnly {
			log.ActionWithSpinner("Rewriting private images")
			io.WriteString(options.ReportWriter, "Rewriting private images\n")
		} else {
			log.ActionWithSpinner("Copying private images")
			io.WriteString(options.ReportWriter, "Copying private images\n")
		}

		// TODO (ethan): rewrite dex image?

		// Rewrite all images
		if options.RewriteImageOptions.ImageFiles == "" {
			newKotsKinds, err := kotsutil.LoadKotsKindsFromPath(upstreamDir)
			if err != nil {
				return nil, errors.Wrap(err, "failed to load kotskinds from new upstream")
			}

			writeUpstreamImageOptions := base.WriteUpstreamImageOptions{
				BaseDir: writeMidstreamOptions.BaseDir,
				Log:     log,
				SourceRegistry: registry.RegistryOptions{
					Endpoint:      replicatedRegistryInfo.Registry,
					ProxyEndpoint: replicatedRegistryInfo.Proxy,
				},
				DockerHubRegistry: registry.RegistryOptions{
					Username: dockerHubRegistryCreds.Username,
					Password: dockerHubRegistryCreds.Password,
				},
				ReportWriter: options.ReportWriter,
				KotsKinds:    newKotsKinds,
				CopyImages:   !options.RewriteImageOptions.IsReadOnly,
			}
			if license != nil {
				writeUpstreamImageOptions.AppSlug = license.Spec.AppSlug
				writeUpstreamImageOptions.SourceRegistry.Username = license.Spec.LicenseID
				writeUpstreamImageOptions.SourceRegistry.Password = license.Spec.LicenseID
			}

			if options.RewriteImageOptions.Host != "" {
				writeUpstreamImageOptions.DestRegistry = registry.RegistryOptions{
					Endpoint:  options.RewriteImageOptions.Host,
					Namespace: options.RewriteImageOptions.Namespace,
					Username:  options.RewriteImageOptions.Username,
					Password:  options.RewriteImageOptions.Password,
				}
			}

			copyResult, err := base.ProcessUpstreamImages(writeUpstreamImageOptions)
			if err != nil {
				return nil, errors.Wrap(err, "failed to write upstream images")
			}
			images = copyResult.Images

			newKotsKinds.Installation.Spec.KnownImages = copyResult.CheckedImages

			err = upstream.SaveInstallation(&newKotsKinds.Installation, upstreamDir)
			if err != nil {
				return nil, errors.Wrap(err, "failed to save installation")
			}
		}

		// If the request includes a rewrite image options host name, then also
		// push the images
		if options.RewriteImageOptions.Host != "" {
			processUpstreamImageOptions := upstream.ProcessUpstreamImagesOptions{
				RootDir:      options.RootDir,
				ImagesDir:    imagesDirFromOptions(u, options),
				AirgapBundle: options.AirgapBundle,
				CreateAppDir: options.CreateAppDir,
				PushImages:   !options.RewriteImageOptions.IsReadOnly && pushImages,
				Log:          log,
				ReplicatedRegistry: registry.RegistryOptions{
					Endpoint:      replicatedRegistryInfo.Registry,
					ProxyEndpoint: replicatedRegistryInfo.Proxy,
				},
				ReportWriter: options.ReportWriter,
				DestinationRegistry: registry.RegistryOptions{
					Endpoint:  options.RewriteImageOptions.Host,
					Namespace: options.RewriteImageOptions.Namespace,
					Username:  options.RewriteImageOptions.Username,
					Password:  options.RewriteImageOptions.Password,
				},
			}
			if license != nil {
				processUpstreamImageOptions.ReplicatedRegistry.Username = license.Spec.LicenseID
				processUpstreamImageOptions.ReplicatedRegistry.Password = license.Spec.LicenseID
			}

			var upstreamImages *upstream.ProcessUpstreamImageResult
			if images == nil { // don't do ProcessUpstreamImages if we already copied them
				imagesData, err := ioutil.ReadFile(filepath.Join(options.AirgapRoot, "images.json"))
				if err != nil && !os.IsNotExist(err) {
					return nil, errors.Wrap(err, "failed to load images file")
				}

				if err == nil {
					err := json.Unmarshal(imagesData, &images)
					if err != nil && !os.IsNotExist(err) {
						return nil, errors.Wrap(err, "failed to unmarshal images data")
					}
					processUpstreamImageOptions.UseKnownImages = true
					processUpstreamImageOptions.KnownImages = images
				}

				upstreamImages, err = upstream.ProcessUpstreamImages(u, processUpstreamImageOptions)
				if err != nil {
					return nil, errors.Wrap(err, "failed to push upstream images")
				}
			}

			affectedObjects := base.FindObjectsWithImages(b)

			registryUser := options.RewriteImageOptions.Username
			registryPass := options.RewriteImageOptions.Password
			if registryUser == "" {
				registryUser, registryPass, err = registry.LoadAuthForRegistry(options.RewriteImageOptions.Host)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to load registry auth for %q", options.RewriteImageOptions.Host)
				}
			}

			namePrefix := options.AppSlug
			// For the newer style charts, create a new secret per chart as helm adds chart specific
			// details to annotations and labels to it.
			for _, v := range writeMidstreamOptions.NewHelmCharts {
				if v.Spec.UseHelmInstall && filepath.Base(b.Path) != "." {
					namePrefix = fmt.Sprintf("%s-%s", options.AppSlug, filepath.Base(b.Path))
					break
				}
			}
			pullSecrets, err = registry.PullSecretForRegistries(
				[]string{options.RewriteImageOptions.Host},
				registryUser,
				registryPass,
				options.Namespace,
				namePrefix,
			)
			if err != nil {
				return nil, errors.Wrap(err, "create pull secret")
			}

			if upstreamImages != nil {
				images = upstreamImages.KustomizeImages

				newKotsKinds, err := kotsutil.LoadKotsKindsFromPath(upstreamDir)
				if err != nil {
					return nil, errors.Wrap(err, "failed to load kotskinds from new upstream")
				}
				newKotsKinds.Installation.Spec.KnownImages = upstreamImages.KnownImages
				err = upstream.SaveInstallation(&newKotsKinds.Installation, upstreamDir)
				if err != nil {
					return nil, errors.Wrap(err, "failed to save installation for private registry")
				}
			}
			objects = affectedObjects
		}
	} else if license != nil {
		newKotsKinds, err := kotsutil.LoadKotsKindsFromPath(upstreamDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load kotskinds")
		}

		allPrivate := newKotsKinds.KotsApplication.Spec.ProxyPublicImages

		// Rewrite private images
		findPrivateImagesOptions := base.FindPrivateImagesOptions{
			BaseDir: writeMidstreamOptions.BaseDir,
			AppSlug: license.Spec.AppSlug,
			ReplicatedRegistry: registry.RegistryOptions{
				Endpoint:      replicatedRegistryInfo.Registry,
				ProxyEndpoint: replicatedRegistryInfo.Proxy,
			},
			DockerHubRegistry: registry.RegistryOptions{
				Username: dockerHubRegistryCreds.Username,
				Password: dockerHubRegistryCreds.Password,
			},
			Installation:     &newKotsKinds.Installation,
			AllImagesPrivate: allPrivate,
			HelmChartPath:    b.Path,
			UseHelmInstall:   writeMidstreamOptions.UseHelmInstall,
			KotsKindsImages:  kotsutil.GetImagesFromKotsKinds(newKotsKinds),
		}
		findResult, err := base.FindPrivateImages(findPrivateImagesOptions)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find private images")
		}

		newKotsKinds.Installation.Spec.KnownImages = findResult.CheckedImages
		err = upstream.SaveInstallation(&newKotsKinds.Installation, upstreamDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to save installation")
		}

		// TODO (ethan): proxy dex image

		// Note that there maybe no rewritten images if only replicated private images are being used.
		// We still need to create the secret in that case.
		if len(findResult.Docs) > 0 {
			namePrefix := options.AppSlug
			// For the newer style charts, create a new secret per chart as helm adds chart specific
			// details to annotations and labels to it.
			for _, v := range writeMidstreamOptions.NewHelmCharts {
				if v.Spec.UseHelmInstall && filepath.Base(b.Path) != "." {
					namePrefix = fmt.Sprintf("%s-%s", options.AppSlug, filepath.Base(b.Path))
					break
				}
			}
			pullSecrets, err = registry.PullSecretForRegistries(
				replicatedRegistryInfo.ToSlice(),
				license.Spec.LicenseID,
				license.Spec.LicenseID,
				options.Namespace,
				namePrefix,
			)
			if err != nil {
				return nil, errors.Wrap(err, "create pull secret")
			}
		}
		images = findResult.Images
		objects = findResult.Docs
	}

	pullSecrets.DockerHubSecret = dockerhubSecret
	m, err := midstream.CreateMidstream(b, images, objects, &pullSecrets, identitySpec, identityConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create midstream")
	}

	if err := m.WriteMidstream(writeMidstreamOptions); err != nil {
		return nil, errors.Wrap(err, "failed to write common midstream")
	}

	return m, nil
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

func GetAppMetadataFromAirgap(airgapArchive string) ([]byte, error) {
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
		errors.Wrap(err, "failed to encode metadata")
	}

	return b.Bytes(), nil
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

func FindAirgapMetaInDir(root string) (*kotsv1beta1.Airgap, error) {
	files, err := ioutil.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to read airgap directory content")
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		contents, err := ioutil.ReadFile(filepath.Join(root, file.Name()))
		if err != nil {
			// TODO: log
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		decoded, gvk, err := decode(contents, nil, nil)
		if err != nil {
			// TODO: log
			continue
		}

		if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Airgap" {
			continue
		}

		airgap := decoded.(*kotsv1beta1.Airgap)
		return airgap, nil
	}

	return nil, nil
}

func imagesDirFromOptions(upstream *upstreamtypes.Upstream, pullOptions PullOptions) string {
	if pullOptions.RewriteImageOptions.ImageFiles != "" {
		return pullOptions.RewriteImageOptions.ImageFiles
	}

	if pullOptions.CreateAppDir {
		return filepath.Join(pullOptions.RootDir, upstream.Name, "images")
	}

	return filepath.Join(pullOptions.RootDir, "images")
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
