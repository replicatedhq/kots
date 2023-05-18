package rewrite

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/apparchive"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/downstream"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/rendered"
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
	KotsApplication    *kotsv1beta1.Application
	ReportWriter       io.Writer
	CopyImages         bool // can be false even if registry is not read-only
	IsAirgap           bool
	RegistrySettings   registrytypes.RegistrySettings
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
	log := logger.NewCLILogger(os.Stdout)

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
		LocalRegistry:            rewriteOptions.RegistrySettings,
		ReportingInfo:            rewriteOptions.ReportingInfo,
		SkipCompatibilityCheck:   true, // we're rewriting an existing version, no need to check for compatibility
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
		IsGKEAutopilot:       k8sutil.IsGKEAutopilot(clientset),
	}
	if err = upstream.WriteUpstream(u, writeUpstreamOptions); err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to write upstream")
	}
	log.FinishSpinner()

	renderOptions := base.RenderOptions{
		SplitMultiDocYAML:       true,
		Namespace:               rewriteOptions.K8sNamespace,
		LocalRegistryHost:       rewriteOptions.RegistrySettings.Hostname,
		LocalRegistryNamespace:  rewriteOptions.RegistrySettings.Namespace,
		LocalRegistryUsername:   rewriteOptions.RegistrySettings.Username,
		LocalRegistryPassword:   rewriteOptions.RegistrySettings.Password,
		LocalRegistryIsReadOnly: rewriteOptions.RegistrySettings.IsReadOnly,
		ExcludeKotsKinds:        rewriteOptions.ExcludeKotsKinds,
		Log:                     log,
		AppSlug:                 rewriteOptions.AppSlug,
		Sequence:                rewriteOptions.AppSequence,
		IsAirgap:                rewriteOptions.IsAirgap,
	}
	log.ActionWithSpinner("Creating base")
	io.WriteString(rewriteOptions.ReportWriter, "Creating base\n")

	commonBase, helmBases, renderedKotsKindsMap, err := base.RenderUpstream(u, &renderOptions)
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

	v1Beta1HelmCharts, err := kotsutil.LoadV1Beta1HelmChartsFromPath(rewriteOptions.UpstreamPath)
	if err != nil {
		return errors.Wrap(err, "failed to load v1beta1 helm charts")
	}

	commonWriteMidstreamOptions := midstream.WriteOptions{
		AppSlug:            rewriteOptions.AppSlug,
		IsGitOps:           rewriteOptions.IsGitOps,
		IsOpenShift:        k8sutil.IsOpenShift(clientset),
		Builder:            *builder,
		HTTPProxyEnvValue:  rewriteOptions.HTTPProxyEnvValue,
		HTTPSProxyEnvValue: rewriteOptions.HTTPSProxyEnvValue,
		NoProxyEnvValue:    rewriteOptions.NoProxyEnvValue,
		NewHelmCharts:      v1Beta1HelmCharts,
	}

	// the UseHelmInstall map blocks visibility into charts and subcharts when searching for private images
	// any chart name listed here will be skipped when writing midstream kustomization.yaml and pullsecret.yaml
	// when using Helm Install, each chart gets it's own kustomization and pullsecret yaml and MUST be skipped when processing higher level directories!
	// for writing Common Midstream, every chart and subchart is in this map as Helm Midstreams will be processed later in the code
	commonWriteMidstreamOptions.UseHelmInstall = map[string]bool{}
	for _, v := range v1Beta1HelmCharts {
		chartBaseName := v.GetDirName()
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
	writeMidstreamOptions.MidstreamDir = filepath.Join(u.GetOverlaysDir(writeUpstreamOptions), "midstream")
	writeMidstreamOptions.BaseDir = filepath.Join(u.GetBaseDir(writeUpstreamOptions), commonBase.Path)

	processImageOptions := image.ProcessImageOptions{
		AppSlug:          rewriteOptions.AppSlug,
		Namespace:        rewriteOptions.K8sNamespace,
		RewriteImages:    rewriteOptions.RegistrySettings.Hostname != "",
		RegistrySettings: rewriteOptions.RegistrySettings,
		CopyImages:       rewriteOptions.CopyImages,
		RootDir:          rewriteOptions.RootDir,
		IsAirgap:         rewriteOptions.IsAirgap,
		AirgapRoot:       "",
		AirgapBundle:     "",
		PushImages:       rewriteOptions.RegistrySettings.Hostname != "",
		CreateAppDir:     false,
		ReportWriter:     rewriteOptions.ReportWriter,
	}

	upstreamDir := u.GetUpstreamDir(writeUpstreamOptions)
	identityConfig, err := upstream.LoadIdentityConfig(upstreamDir)
	if err != nil {
		return errors.Wrap(err, "failed to load identity config")
	}

	m, err := midstream.WriteMidstream(writeMidstreamOptions, processImageOptions, commonBase, rewriteOptions.License, identityConfig, upstreamDir, log)
	if err != nil {
		return errors.Wrap(err, "failed to write common midstream")
	}

	helmMidstreams := []midstream.Midstream{}
	for _, helmBase := range helmBases {
		// we must look at the current chart for private images, but must ignore subcharts
		// to do this, we remove only the current helmBase name from the UseHelmInstall map to unblock visibility into the chart directory
		// this ensures only the current chart resources are added to kustomization.yaml and pullsecret.yaml
		// chartName := strings.Split(helmBase.Path, "/")[len(strings.Split(helmBase.Path, "/"))-1]
		// copy the bool setting in the map to restore it after this process loop
		previousUseHelmInstall := writeMidstreamOptions.UseHelmInstall[helmBase.Path]
		writeMidstreamOptions.UseHelmInstall[helmBase.Path] = false

		writeMidstreamOptions.MidstreamDir = filepath.Join(u.GetOverlaysDir(writeUpstreamOptions), "midstream", helmBase.Path)
		writeMidstreamOptions.BaseDir = filepath.Join(u.GetBaseDir(writeUpstreamOptions), helmBase.Path)

		helmBaseCopy := helmBase.DeepCopy()

		processImageOptionsCopy := processImageOptions
		processImageOptionsCopy.Namespace = helmBaseCopy.Namespace
		processImageOptionsCopy.CopyImages = false // don't copy images more than once

		helmMidstream, err := midstream.WriteMidstream(writeMidstreamOptions, processImageOptionsCopy, helmBaseCopy, rewriteOptions.License, identityConfig, upstreamDir, log)
		if err != nil {
			return errors.Wrapf(err, "failed to write helm midstream %s", helmBase.Path)
		}

		// add this chart back into UseHelmInstall to make sure it's not processed again
		writeMidstreamOptions.UseHelmInstall[helmBase.Path] = previousUseHelmInstall

		helmMidstreams = append(helmMidstreams, *helmMidstream)
	}

	renderedKotsKinds, err := kotsutil.KotsKindsFromMap(renderedKotsKindsMap)
	if err != nil {
		return errors.Wrap(err, "failed to load rendered kotskinds from map")
	}

	writeV1Beta2HelmChartsOpts := apparchive.WriteV1Beta2HelmChartsOptions{
		Upstream:            u,
		RenderOptions:       &renderOptions,
		ProcessImageOptions: processImageOptions,
		HelmDir:             u.GetHelmDir(writeUpstreamOptions),
		KotsKinds:           renderedKotsKinds,
		Clientset:           clientset,
	}

	if err := apparchive.WriteV1Beta2HelmCharts(writeV1Beta2HelmChartsOpts); err != nil {
		return errors.Wrap(err, "failed to write helm v1beta2 charts")
	}

	if err := writeDownstreams(rewriteOptions, u.GetOverlaysDir(writeUpstreamOptions), m, helmMidstreams, log); err != nil {
		return errors.Wrap(err, "failed to write downstreams")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(rewriteOptions.UpstreamPath)
	if err != nil {
		return errors.Wrap(err, "failed to load kotskinds")
	}

	err = store.GetStore().UpdateAppVersionInstallationSpec(rewriteOptions.AppID, rewriteOptions.AppSequence, kotsKinds.Installation)
	if err != nil {
		return errors.Wrap(err, "failed to update installation spec")
	}

	installationBytes, err := ioutil.ReadFile(filepath.Join(u.GetUpstreamDir(writeUpstreamOptions), "userdata", "installation.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to read installation file")
	}

	installationFilename := kotsutil.GenUniqueKotsKindFilename(renderedKotsKindsMap, "installation")
	renderedKotsKindsMap[installationFilename] = []byte(installationBytes)

	if err := kotsutil.WriteKotsKinds(renderedKotsKindsMap, u.GetKotsKindsDir(writeUpstreamOptions)); err != nil {
		return errors.Wrap(err, "failed to write kots base")
	}

	if err := rendered.WriteRenderedApp(&rendered.WriteOptions{
		BaseDir:             u.GetBaseDir(writeUpstreamOptions),
		OverlaysDir:         u.GetOverlaysDir(writeUpstreamOptions),
		RenderedDir:         u.GetRenderedDir(writeUpstreamOptions),
		Downstreams:         rewriteOptions.Downstreams,
		KustomizeBinPath:    kotsKinds.GetKustomizeBinaryPath(),
		HelmDir:             u.GetHelmDir(writeUpstreamOptions),
		Log:                 log,
		KotsKinds:           renderedKotsKinds,
		ProcessImageOptions: processImageOptions,
		Clientset:           clientset,
	}); err != nil {
		return errors.Wrap(err, "failed to write rendered")
	}

	log.FinishSpinner()

	return nil
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
