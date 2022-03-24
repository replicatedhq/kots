package rewrite

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/downstream"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/upstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

type RewriteOptions struct {
	RootDir            string
	UpstreamURI        string
	UpstreamPath       string
	Downstreams        []string
	K8sNamespace       string
	Silent             bool
	CreateAppDir       bool
	ExcludeKotsKinds   bool
	Installation       *kotsv1beta1.Installation
	License            *kotsv1beta1.License
	ConfigValues       *kotsv1beta1.ConfigValues
	ReportWriter       io.Writer
	CopyImages         bool // can be false even if registry is not read-only
	IsAirgap           bool
	RegistryEndpoint   string
	RegistryUsername   string
	RegistryPassword   string
	RegistryNamespace  string
	RegistryIsReadOnly bool
	AppID              string
	AppSlug            string
	IsGitOps           bool
	AppSequence        int64
	ReportingInfo      *reportingtypes.ReportingInfo
	HTTPProxyEnvValue  string
	HTTPSProxyEnvValue string
	NoProxyEnvValue    string
}

func Rewrite(rewriteOptions RewriteOptions) error {
	log := logger.NewCLILogger()

	if rewriteOptions.Silent {
		log.Silence()
	}

	log.Initialize()

	if rewriteOptions.ReportWriter == nil {
		rewriteOptions.ReportWriter = ioutil.Discard
	}

	fetchOptions := &upstreamtypes.FetchOptions{
		RootDir:                  rewriteOptions.RootDir,
		LocalPath:                rewriteOptions.UpstreamPath,
		CurrentCursor:            rewriteOptions.Installation.Spec.UpdateCursor,
		CurrentVersionLabel:      rewriteOptions.Installation.Spec.VersionLabel,
		CurrentVersionIsRequired: rewriteOptions.Installation.Spec.IsRequired,
		EncryptionKey:            rewriteOptions.Installation.Spec.EncryptionKey,
		License:                  rewriteOptions.License,
		AppSequence:              rewriteOptions.AppSequence,
		AppSlug:                  rewriteOptions.AppSlug,
		LocalRegistry: upstreamtypes.LocalRegistry{
			Host:      rewriteOptions.RegistryEndpoint,
			Namespace: rewriteOptions.RegistryNamespace,
			Username:  rewriteOptions.RegistryUsername,
			Password:  rewriteOptions.RegistryPassword,
			ReadOnly:  rewriteOptions.RegistryIsReadOnly,
		},
		ReportingInfo:          rewriteOptions.ReportingInfo,
		SkipCompatibilityCheck: true, // we're rewriting an existing version, no need to check for compatibility
	}

	log.ActionWithSpinner("Pulling upstream")
	io.WriteString(rewriteOptions.ReportWriter, "Pulling upstream\n")
	u, err := upstream.FetchUpstream(rewriteOptions.UpstreamURI, fetchOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to load upstream")
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s clientset")
	}

	writeUpstreamOptions := upstreamtypes.WriteOptions{
		RootDir:              rewriteOptions.RootDir,
		CreateAppDir:         rewriteOptions.CreateAppDir,
		IncludeAdminConsole:  false,
		PreserveInstallation: true,
		IsOpenShift:          k8sutil.IsOpenShift(clientset),
	}
	if err := upstream.WriteUpstream(u, writeUpstreamOptions); err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to write upstream")
	}
	log.FinishSpinner()

	renderOptions := base.RenderOptions{
		SplitMultiDocYAML:       true,
		Namespace:               rewriteOptions.K8sNamespace,
		LocalRegistryHost:       rewriteOptions.RegistryEndpoint,
		LocalRegistryNamespace:  rewriteOptions.RegistryNamespace,
		LocalRegistryUsername:   rewriteOptions.RegistryUsername,
		LocalRegistryPassword:   rewriteOptions.RegistryPassword,
		LocalRegistryIsReadOnly: rewriteOptions.RegistryIsReadOnly,
		ExcludeKotsKinds:        rewriteOptions.ExcludeKotsKinds,
		Log:                     log,
		AppSlug:                 rewriteOptions.AppSlug,
		Sequence:                rewriteOptions.AppSequence,
		IsAirgap:                rewriteOptions.IsAirgap,
	}
	log.ActionWithSpinner("Creating base")
	io.WriteString(rewriteOptions.ReportWriter, "Creating base\n")

	commonBase, helmBases, err := base.RenderUpstream(u, &renderOptions)
	if err != nil {
		return errors.Wrap(err, "failed to render upstream")
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
			return errors.Wrap(err, "failed to load installation")
		}
		newKotsKinds.Installation.Spec.YAMLErrors = files

		err = upstream.SaveInstallation(&newKotsKinds.Installation, u.GetUpstreamDir(writeUpstreamOptions))
		if err != nil {
			return errors.Wrap(err, "failed to save installation")
		}
	}

	writeBaseOptions := base.WriteOptions{
		BaseDir:          u.GetBaseDir(writeUpstreamOptions),
		SkippedDir:       u.GetSkippedDir(writeUpstreamOptions),
		Overwrite:        true,
		ExcludeKotsKinds: rewriteOptions.ExcludeKotsKinds,
		IsHelmBase:       false,
	}
	if err := commonBase.WriteBase(writeBaseOptions); err != nil {
		return errors.Wrap(err, "failed to write common base")
	}

	for _, helmBase := range helmBases {
		helmBaseCopy := helmBase.DeepCopy()
		// strip namespace. helm render takes care of injecting the namespace
		helmBaseCopy.SetNamespace("")
		writeBaseOptions := base.WriteOptions{
			BaseDir:          u.GetBaseDir(writeUpstreamOptions),
			SkippedDir:       u.GetSkippedDir(writeUpstreamOptions),
			Overwrite:        true,
			ExcludeKotsKinds: rewriteOptions.ExcludeKotsKinds,
			IsHelmBase:       true,
		}
		if err := helmBaseCopy.WriteBase(writeBaseOptions); err != nil {
			return errors.Wrapf(err, "failed to write helm base %s", helmBaseCopy.Path)
		}
	}

	log.FinishSpinner()

	log.ActionWithSpinner("Creating midstreams")
	io.WriteString(rewriteOptions.ReportWriter, "Creating midstreams\n")

	builder, _, err := base.NewConfigContextTemplateBuilder(u, &renderOptions)
	if err != nil {
		return errors.Wrap(err, "failed to create new config context template builder")
	}

	err = crypto.InitFromString(rewriteOptions.Installation.Spec.EncryptionKey)
	if err != nil {
		return errors.Wrap(err, "failed to create cipher from installation spec")
	}

	var newHelmCharts []*kotsv1beta1.HelmChart
	newHelmCharts, err = kotsutil.LoadHelmChartsFromPath(rewriteOptions.UpstreamPath)
	if err != nil {
		return errors.Wrap(err, "failed to load new helm charts")
	}

	commonWriteMidstreamOptions := midstream.WriteOptions{
		AppSlug:            rewriteOptions.AppSlug,
		IsGitOps:           rewriteOptions.IsGitOps,
		IsOpenShift:        k8sutil.IsOpenShift(clientset),
		Builder:            *builder,
		HTTPProxyEnvValue:  rewriteOptions.HTTPProxyEnvValue,
		HTTPSProxyEnvValue: rewriteOptions.HTTPSProxyEnvValue,
		NoProxyEnvValue:    rewriteOptions.NoProxyEnvValue,
		NewHelmCharts:      newHelmCharts,
	}

	// the UseHelmInstall map blocks visibility into charts and subcharts when searching for private images
	// any chart name listed here will be skipped when writing midstream kustomization.yaml and pullsecret.yaml
	// when using Helm Install, each chart gets it's own kustomization and pullsecret yaml and MUST be skipped when processing higher level directories!
	// for writing Common Midstream, every chart and subchart is in this map as Helm Midstreams will be processed later in the code
	commonWriteMidstreamOptions.UseHelmInstall = map[string]bool{}
	for _, v := range newHelmCharts {
		chartBaseName := v.Spec.Chart.Name
		// the helmBase may have a chart name prefix removed - we must find the base name instead of the original chart name
		for _, helmBase := range helmBases {
			chartName := strings.Split(helmBase.Path, "/")[len(strings.Split(helmBase.Path, "/"))-1]
			if strings.HasSuffix(chartBaseName, chartName) {
				chartBaseName = chartName
			}
		}
		commonWriteMidstreamOptions.UseHelmInstall[chartBaseName] = v.Spec.UseHelmInstall
		if v.Spec.UseHelmInstall {
			subcharts, err := base.FindHelmSubChartsFromBase(writeBaseOptions.BaseDir, chartBaseName)
			if err != nil {
				return errors.Wrapf(err, "failed to find subcharts for parent chart %s", chartBaseName)
			}
			for _, subchart := range subcharts.SubCharts {
				commonWriteMidstreamOptions.UseHelmInstall[subchart] = v.Spec.UseHelmInstall
			}
		}
	}

	writeMidstreamOptions := commonWriteMidstreamOptions
	writeMidstreamOptions.MidstreamDir = filepath.Join(commonBase.GetOverlaysDir(writeBaseOptions), "midstream")
	writeMidstreamOptions.BaseDir = filepath.Join(u.GetBaseDir(writeUpstreamOptions), commonBase.Path)

	m, err := writeMidstream(writeMidstreamOptions, rewriteOptions, commonBase, fetchOptions.License, u.GetUpstreamDir(writeUpstreamOptions), log)
	if err != nil {
		return errors.Wrap(err, "failed to write common midstream")
	}

	helmMidstreams := []midstream.Midstream{}
	for _, helmBase := range helmBases {
		// we must look at the current chart for private images, but must ignore subcharts
		// to do this, we remove only the current helmBase name from the UseHelmInstall map to unblock visibility into the chart directory
		// this ensures only the current chart resources are added to kustomization.yaml and pullsecret.yaml
		chartName := strings.Split(helmBase.Path, "/")[len(strings.Split(helmBase.Path, "/"))-1]
		// copy the bool setting in the map to restore it after this process loop
		useHelmSetting := writeMidstreamOptions.UseHelmInstall[chartName]
		delete(writeMidstreamOptions.UseHelmInstall, chartName)

		writeMidstreamOptions.MidstreamDir = filepath.Join(helmBase.GetOverlaysDir(writeBaseOptions), "midstream", helmBase.Path)
		writeMidstreamOptions.BaseDir = filepath.Join(u.GetBaseDir(writeUpstreamOptions), helmBase.Path)

		helmBaseCopy := helmBase.DeepCopy()

		rewriteOptionsCopy := rewriteOptions
		rewriteOptionsCopy.K8sNamespace = helmBaseCopy.Namespace
		rewriteOptionsCopy.CopyImages = false // don't copy images more than once

		helmMidstream, err := writeMidstream(writeMidstreamOptions, rewriteOptionsCopy, helmBaseCopy, fetchOptions.License, u.GetUpstreamDir(writeUpstreamOptions), log)
		if err != nil {
			return errors.Wrapf(err, "failed to write helm midstream %s", helmBase.Path)
		}

		// add this chart back into UseHelmInstall to make sure it's not processed again
		writeMidstreamOptions.UseHelmInstall[chartName] = useHelmSetting

		helmMidstreams = append(helmMidstreams, *helmMidstream)
	}

	if err := writeDownstreams(rewriteOptions, commonBase.GetOverlaysDir(writeBaseOptions), m, helmMidstreams, log); err != nil {
		return errors.Wrap(err, "failed to write downstreams")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(rewriteOptions.RootDir)
	if err != nil {
		return errors.Wrap(err, "failed to load kotskinds")
	}

	err = store.GetStore().UpdateAppVersionInstallationSpec(rewriteOptions.AppID, rewriteOptions.AppSequence, kotsKinds.Installation)
	if err != nil {
		return errors.Wrap(err, "failed to updates installation spec")
	}

	log.FinishSpinner()

	return nil
}

func writeMidstream(writeMidstreamOptions midstream.WriteOptions, options RewriteOptions, b *base.Base, license *kotsv1beta1.License, upstreamDir string, log *logger.CLILogger) (*midstream.Midstream, error) {
	var pullSecrets *registry.ImagePullSecrets
	var images []kustomizetypes.Image
	var objects []k8sdoc.K8sDoc

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	replicatedRegistryInfo := registry.ProxyEndpointFromLicense(options.License)

	identitySpec, err := upstream.LoadIdentity(upstreamDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load identity")
	}

	identityConfig, err := upstream.LoadIdentityConfig(upstreamDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load identity config")
	}

	// do not fail on being unable to get dockerhub credentials, since they're just used to increase the rate limit
	dockerHubRegistryCreds, _ := registry.GetDockerHubCredentials(clientset, options.K8sNamespace)

	// TODO (ethan): rewrite dex image?

	if options.CopyImages || options.RegistryEndpoint != "" {
		// When CopyImages is set, we copy images, rewrite all images, and use registry
		// settings to create secrets for all objects that have images.
		// When only registry endpoint is set, we don't need to copy images, but still
		// need to rewrite them and create secrets.

		newKotsKinds, err := kotsutil.LoadKotsKindsFromPath(upstreamDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load kotskinds from new upstream")
		}

		writeUpstreamImageOptions := base.WriteUpstreamImageOptions{
			BaseDir:      writeMidstreamOptions.BaseDir,
			ReportWriter: options.ReportWriter,
			Log:          log,
			SourceRegistry: registry.RegistryOptions{
				Endpoint:      replicatedRegistryInfo.Registry,
				ProxyEndpoint: replicatedRegistryInfo.Proxy,
			},
			DestRegistry: registry.RegistryOptions{
				Endpoint:  options.RegistryEndpoint,
				Namespace: options.RegistryNamespace,
				Username:  options.RegistryUsername,
				Password:  options.RegistryPassword,
			},
			DockerHubRegistry: registry.RegistryOptions{
				Username: dockerHubRegistryCreds.Username,
				Password: dockerHubRegistryCreds.Password,
			},
			KotsKinds:  newKotsKinds,
			IsAirgap:   options.IsAirgap,
			CopyImages: options.CopyImages,
		}

		if license != nil {
			writeUpstreamImageOptions.AppSlug = license.Spec.AppSlug
			writeUpstreamImageOptions.SourceRegistry.Username = license.Spec.LicenseID
			writeUpstreamImageOptions.SourceRegistry.Password = license.Spec.LicenseID
		}

		copyResult, err := base.ProcessUpstreamImages(writeUpstreamImageOptions)
		if err != nil {
			return nil, errors.Wrap(err, "failed to write upstream images")
		}

		newKotsKinds.Installation.Spec.KnownImages = copyResult.CheckedImages
		err = upstream.SaveInstallation(&newKotsKinds.Installation, upstreamDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to save installation")
		}
		affectedObjects := base.FindObjectsWithImages(b)

		registryUser := options.RegistryUsername
		registryPass := options.RegistryPassword
		if registryUser == "" {
			// this will only work when envoked from CLI where `docker login` command has been executed
			registryUser, registryPass, err = registry.LoadAuthForRegistry(options.RegistryEndpoint)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load registry auth for %q", options.RegistryEndpoint)
			}
		}
		namePrefix := options.AppSlug
		// For the newer style charts, create a new secret per chart as helm adds chart specific
		// details to annotations and labels to it.
		for _, v := range writeMidstreamOptions.NewHelmCharts {
			if filepath.Base(b.Path) == v.Spec.Chart.Name && v.Spec.UseHelmInstall == true {
				namePrefix = fmt.Sprintf("%s-%s", options.AppSlug, filepath.Base(b.Path))
				break
			}
		}
		pullSecrets, err = registry.PullSecretForRegistries(
			[]string{options.RegistryEndpoint},
			registryUser,
			registryPass,
			options.K8sNamespace,
			namePrefix,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create private registry pull secrets")
		}

		images = copyResult.Images
		objects = affectedObjects
	} else {
		kotsKinds, err := kotsutil.LoadKotsKindsFromPath(upstreamDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load kotskonds")
		}

		allPrivate := kotsKinds.KotsApplication.Spec.ProxyPublicImages

		// When CopyImages is not set, we only rewrite private images and use license to create secrets
		// for all objects that have private images
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
			Installation:     options.Installation,
			AllImagesPrivate: allPrivate,
			UseHelmInstall:   writeMidstreamOptions.UseHelmInstall,
		}
		findResult, err := base.FindPrivateImages(findPrivateImagesOptions)
		if err != nil {
			return nil, errors.Wrap(err, "failed to find private images")
		}

		kotsKinds.Installation.Spec.KnownImages = findResult.CheckedImages
		err = upstream.SaveInstallation(&kotsKinds.Installation, upstreamDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to save installation")
		}

		if len(findResult.Docs) > 0 {
			namePrefix := options.AppSlug
			// For the newer style charts, create a new secret per chart as helm adds chart specific
			// details to annotations and labels to it.
			for _, v := range writeMidstreamOptions.NewHelmCharts {
				if filepath.Base(b.Path) == v.Spec.Chart.Name && v.Spec.UseHelmInstall == true {
					namePrefix = fmt.Sprintf("%s-%s", options.AppSlug, filepath.Base(b.Path))
					break
				}
			}

			replicatedRegistryInfo := registry.ProxyEndpointFromLicense(options.License)
			pullSecrets, err = registry.PullSecretForRegistries(
				replicatedRegistryInfo.ToSlice(),
				options.License.Spec.LicenseID,
				options.License.Spec.LicenseID,
				options.K8sNamespace,
				namePrefix,
			)
			if err != nil {
				return nil, errors.Wrap(err, "failed to create Replicated registry pull secret")
			}
		}

		images = findResult.Images
		objects = findResult.Docs
	}

	m, err := midstream.CreateMidstream(b, images, objects, pullSecrets, identitySpec, identityConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create midstream")
	}

	if err := m.WriteMidstream(writeMidstreamOptions); err != nil {
		return nil, errors.Wrap(err, "failed to write common midstream")
	}

	return m, nil
}

func writeDownstreams(options RewriteOptions, overlaysDir string, m *midstream.Midstream, helmMidstreams []midstream.Midstream, log *logger.CLILogger) error {
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
			helmMidstream := mid

			d, err := downstream.CreateDownstream(&helmMidstream)
			if err != nil {
				return errors.Wrapf(err, "failed to create downstream %s for midstream %s", downstreamName, mid.Base.Path)
			}

			writeDownstreamOptions := downstream.WriteOptions{
				DownstreamDir: filepath.Join(overlaysDir, "downstreams", downstreamName, mid.Base.Path),
				MidstreamDir:  filepath.Join(overlaysDir, "midstream", mid.Base.Path),
			}
			if err := d.WriteDownstream(writeDownstreamOptions); err != nil {
				return errors.Wrapf(err, "failed to write downstream %s for midstream %s", downstreamName, mid.Base.Path)
			}
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
