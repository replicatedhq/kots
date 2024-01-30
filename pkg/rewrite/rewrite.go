package rewrite

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/apparchive"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/binaries"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/downstream"
	imagetypes "github.com/replicatedhq/kots/pkg/image/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	"github.com/replicatedhq/kots/pkg/rendered"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/upstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
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
		RootDir:                         rewriteOptions.RootDir,
		LocalPath:                       rewriteOptions.UpstreamPath,
		CurrentCursor:                   rewriteOptions.Installation.Spec.UpdateCursor,
		CurrentVersionLabel:             rewriteOptions.Installation.Spec.VersionLabel,
		CurrentVersionIsRequired:        rewriteOptions.Installation.Spec.IsRequired,
		CurrentReplicatedRegistryDomain: rewriteOptions.Installation.Spec.ReplicatedRegistryDomain,
		CurrentReplicatedProxyDomain:    rewriteOptions.Installation.Spec.ReplicatedProxyDomain,
		CurrentReplicatedChartNames:     rewriteOptions.Installation.Spec.ReplicatedChartNames,
		EncryptionKey:                   rewriteOptions.Installation.Spec.EncryptionKey,
		License:                         rewriteOptions.License,
		AppSequence:                     rewriteOptions.AppSequence,
		AppSlug:                         rewriteOptions.AppSlug,
		LocalRegistry:                   rewriteOptions.RegistrySettings,
		ReportingInfo:                   rewriteOptions.ReportingInfo,
		SkipCompatibilityCheck:          true, // we're rewriting an existing version, no need to check for compatibility
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
		IsAirgap:             rewriteOptions.IsAirgap,
		KotsadmID:            k8sutil.GetKotsadmID(clientset),
		AppID:                rewriteOptions.AppID,
	}
	if err = upstream.WriteUpstream(u, writeUpstreamOptions); err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to write upstream")
	}
	log.FinishSpinner()

	renderOptions := base.RenderOptions{
		SplitMultiDocYAML: true,
		Namespace:         rewriteOptions.K8sNamespace,
		RegistrySettings:  rewriteOptions.RegistrySettings,
		ExcludeKotsKinds:  rewriteOptions.ExcludeKotsKinds,
		Log:               log,
		AppSlug:           rewriteOptions.AppSlug,
		Sequence:          rewriteOptions.AppSequence,
		IsAirgap:          rewriteOptions.IsAirgap,
	}
	log.ActionWithSpinner("Rendering KOTS custom resources")
	io.WriteString(rewriteOptions.ReportWriter, "Rendering KOTS custom resources\n")

	renderedKotsKindsMap, err := base.RenderKotsKinds(u, &renderOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to render upstream")
	}

	renderedKotsKinds, err := kotsutil.KotsKindsFromMap(renderedKotsKindsMap)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to load rendered kotskinds from map")
	}
	log.FinishSpinner()

	log.ActionWithSpinner("Creating base")
	io.WriteString(rewriteOptions.ReportWriter, "Creating base\n")

	commonBase, helmBases, err := base.RenderUpstream(u, &renderOptions, renderedKotsKinds)
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

		renderedKotsKinds.Installation.Spec.YAMLErrors = files

		if err := kotsutil.SaveInstallation(&renderedKotsKinds.Installation, u.GetUpstreamDir(writeUpstreamOptions)); err != nil {
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

	if err := crypto.InitFromString(rewriteOptions.Installation.Spec.EncryptionKey); err != nil {
		return errors.Wrap(err, "failed to create cipher from installation spec")
	}

	identityConfig, err := upstream.LoadIdentityConfig(u.GetUpstreamDir(writeUpstreamOptions))
	if err != nil {
		return errors.Wrap(err, "failed to load identity config")
	}

	var v1Beta1HelmCharts []*kotsv1beta1.HelmChart
	if renderedKotsKinds.V1Beta1HelmCharts != nil {
		for _, v1Beta1Chart := range renderedKotsKinds.V1Beta1HelmCharts.Items {
			kc := v1Beta1Chart
			v1Beta1HelmCharts = append(v1Beta1HelmCharts, &kc)
		}
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
		License:            rewriteOptions.License,
		KotsKinds:          renderedKotsKinds,
		IdentityConfig:     identityConfig,
		UpstreamDir:        u.GetUpstreamDir(writeUpstreamOptions),
		Log:                log,
	}

	processImageOptions := imagetypes.ProcessImageOptions{
		AppSlug:          rewriteOptions.AppSlug,
		Namespace:        rewriteOptions.K8sNamespace,
		RewriteImages:    rewriteOptions.RegistrySettings.Hostname != "",
		RegistrySettings: rewriteOptions.RegistrySettings,
		CopyImages:       rewriteOptions.CopyImages,
		RootDir:          rewriteOptions.RootDir,
		IsAirgap:         rewriteOptions.IsAirgap,
		AirgapRoot:       "",
		AirgapBundle:     "",
		CreateAppDir:     false,
		ReportWriter:     rewriteOptions.ReportWriter,
	}

	writeMidstreamOptions := commonWriteMidstreamOptions
	writeMidstreamOptions.MidstreamDir = filepath.Join(u.GetOverlaysDir(writeUpstreamOptions), "midstream")
	writeMidstreamOptions.BaseDir = filepath.Join(u.GetBaseDir(writeUpstreamOptions), commonBase.Path)
	writeMidstreamOptions.ProcessImageOptions = processImageOptions
	writeMidstreamOptions.Base = commonBase

	m, err := midstream.WriteMidstream(writeMidstreamOptions)
	if err != nil {
		return errors.Wrap(err, "failed to write common midstream")
	}

	helmMidstreams := []midstream.Midstream{}
	for _, helmBase := range helmBases {
		helmBaseCopy := helmBase.DeepCopy()

		processImageOptionsCopy := processImageOptions
		processImageOptionsCopy.Namespace = helmBaseCopy.Namespace
		if processImageOptions.IsAirgap {
			// don't copy images if airgap, as all images would've been pushed from the airgap bundle.
			processImageOptionsCopy.CopyImages = false
		}

		writeMidstreamOptions.MidstreamDir = filepath.Join(u.GetOverlaysDir(writeUpstreamOptions), "midstream", helmBaseCopy.Path)
		writeMidstreamOptions.BaseDir = filepath.Join(u.GetBaseDir(writeUpstreamOptions), helmBaseCopy.Path)
		writeMidstreamOptions.ProcessImageOptions = processImageOptionsCopy
		writeMidstreamOptions.Base = helmBaseCopy

		helmMidstream, err := midstream.WriteMidstream(writeMidstreamOptions)
		if err != nil {
			return errors.Wrapf(err, "failed to write helm midstream %s", helmBase.Path)
		}

		helmMidstreams = append(helmMidstreams, *helmMidstream)
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
		return errors.Wrap(err, "failed to write helm v1beta2 charts")
	}

	if err := writeDownstreams(rewriteOptions, u.GetOverlaysDir(writeUpstreamOptions), m, helmMidstreams, log); err != nil {
		return errors.Wrap(err, "failed to write downstreams")
	}

	if err := store.GetStore().UpdateAppVersionInstallationSpec(rewriteOptions.AppID, rewriteOptions.AppSequence, renderedKotsKinds.Installation); err != nil {
		return errors.Wrap(err, "failed to update installation spec")
	}

	// installation spec gets updated during the process, ensure the map has the latest version
	installationBytes, err := os.ReadFile(filepath.Join(u.GetUpstreamDir(writeUpstreamOptions), "userdata", "installation.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to read installation file")
	}
	renderedKotsKindsMap["userdata/installation.yaml"] = installationBytes

	if err := rendered.WriteRenderedApp(&rendered.WriteOptions{
		BaseDir:             u.GetBaseDir(writeUpstreamOptions),
		OverlaysDir:         u.GetOverlaysDir(writeUpstreamOptions),
		RenderedDir:         u.GetRenderedDir(writeUpstreamOptions),
		Downstreams:         rewriteOptions.Downstreams,
		KustomizeBinPath:    binaries.GetKustomizeBinPath(),
		HelmDir:             u.GetHelmDir(writeUpstreamOptions),
		Log:                 log,
		KotsKinds:           renderedKotsKinds,
		ProcessImageOptions: processImageOptions,
		Clientset:           clientset,
	}); err != nil {
		return errors.Wrap(err, "failed to write rendered")
	}

	// preflights may also be included within helm chart templates, so load any from the rendered dir
	tsKinds, err := kotsutil.LoadTSKindsFromPath(u.GetRenderedDir(writeUpstreamOptions))
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to load troubleshoot kinds from path: %s", u.GetRenderedDir(writeUpstreamOptions)))
	}

	if tsKinds.PreflightsV1Beta2 != nil {
		var renderedPreflight *troubleshootv1beta2.Preflight
		for _, v := range tsKinds.PreflightsV1Beta2 {
			renderedPreflight = troubleshootpreflight.ConcatPreflightSpec(renderedPreflight, &v)
		}

		if renderedPreflight != nil {
			renderedPreflightBytes, err := kotsutil.MarshalRuntimeObject(renderedPreflight)
			if err != nil {
				return errors.Wrap(err, "failed to marshal rendered preflight")
			}
			renderedKotsKindsMap["helm-preflight.yaml"] = renderedPreflightBytes
		}
	}

	if err := kotsutil.WriteKotsKinds(renderedKotsKindsMap, u.GetKotsKindsDir(writeUpstreamOptions)); err != nil {
		return errors.Wrap(err, "failed to write the rendered kots kinds")
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
