package apparchive

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	dockerregistrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kotsv1beta2 "github.com/replicatedhq/kotskinds/apis/kots/v1beta2"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/chartutil"
)

// GetV1Beta2ChartsArchive returns an archive of the v1beta2 charts to be deployed
func GetV1Beta2ChartsArchive(deployedVersionArchive string) ([]byte, error) {
	chartsDir := filepath.Join(deployedVersionArchive, "helm")
	if _, err := os.Stat(chartsDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to stat charts dir")
	}

	archive, err := util.TGZArchive(chartsDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create charts archive")
	}

	return archive, nil
}

// GetRenderedV1Beta2FileMap returns a map of the rendered v1beta2 charts to be deployed
func GetRenderedV1Beta2FileMap(deployedVersionArchive, downstream string) (map[string][]byte, error) {
	chartsDir := filepath.Join(deployedVersionArchive, "rendered", downstream, "helm")
	if _, err := os.Stat(chartsDir); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to stat charts dir")
	}

	filesMap, err := util.GetFilesMap(chartsDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get files map")
	}

	return filesMap, nil
}

type WriteV1Beta2HelmChartsOptions struct {
	Upstream             *upstreamtypes.Upstream
	WriteUpstreamOptions upstreamtypes.WriteOptions
	RenderOptions        *base.RenderOptions
	ProcessImageOptions  image.ProcessImageOptions
	KotsKinds            *kotsutil.KotsKinds
	Clientset            kubernetes.Interface
}

// WriteV1Beta2HelmCharts copies the upstream helm chart archive and rendered values to the helm directory and processes online images (if necessary)
func WriteV1Beta2HelmCharts(opts WriteV1Beta2HelmChartsOptions) error {
	// clear the previous helm dir before writing
	helmDir := opts.Upstream.GetHelmDir(opts.WriteUpstreamOptions)
	os.RemoveAll(helmDir)

	if opts.KotsKinds == nil || opts.KotsKinds.V1Beta2HelmCharts == nil {
		return nil
	}

	checkedImages := []kotsv1beta1.InstallationImage{}

	for _, v1Beta2Chart := range opts.KotsKinds.V1Beta2HelmCharts.Items {
		helmChart := v1Beta2Chart

		if !helmChart.Spec.Exclude.IsEmpty() {
			exclude, err := helmChart.Spec.Exclude.Boolean()
			if err != nil {
				return errors.Wrap(err, "failed to parse exclude boolean")
			}

			if exclude {
				continue
			}
		}

		chartDir := path.Join(helmDir, helmChart.GetDirName())
		if err := os.MkdirAll(chartDir, 0744); err != nil {
			return errors.Wrap(err, "failed to create chart dir")
		}

		archive, err := base.FindHelmChartArchiveInRelease(opts.Upstream.Files, &helmChart)
		if err != nil {
			return errors.Wrap(err, "failed to find helm chart archive in release")
		}

		archivePath := path.Join(chartDir, fmt.Sprintf("%s-%s.tgz", helmChart.Spec.Chart.Name, helmChart.Spec.Chart.ChartVersion))
		if err := os.WriteFile(archivePath, archive, 0644); err != nil {
			return errors.Wrap(err, "failed to write helm chart archive")
		}

		mergedValues := helmChart.Spec.Values
		if mergedValues == nil {
			mergedValues = map[string]kotsv1beta2.MappedChartValue{}
		}
		for _, optionalValues := range helmChart.Spec.OptionalValues {
			parsedBool, err := strconv.ParseBool(optionalValues.When)
			if err != nil {
				return errors.Wrap(err, "failed to parse when conditional on optional values")
			}
			if !parsedBool {
				continue
			}
			if optionalValues.RecursiveMerge {
				mergedValues = kotsv1beta2.MergeHelmChartValues(mergedValues, optionalValues.Values)
			} else {
				for k, v := range optionalValues.Values {
					mergedValues[k] = v
				}
			}
		}

		helmValues, err := helmChart.Spec.GetHelmValues(mergedValues)
		if err != nil {
			return errors.Wrap(err, "failed to get local values for chart")
		}

		valuesContent, err := yaml.Marshal(helmValues)
		if err != nil {
			return errors.Wrap(err, "failed to marshal values")
		}

		valuesPath := path.Join(chartDir, "values.yaml")
		if err := os.WriteFile(valuesPath, []byte(valuesContent), 0644); err != nil {
			return errors.Wrap(err, "failed to write values file")
		}

		if !opts.ProcessImageOptions.RewriteImages || opts.ProcessImageOptions.AirgapRoot != "" {
			// if an on-prem registry is not configured (which means it's an online installation)
			// there's no need to process/copy the images as they will be pulled from their original registries or through the replicated proxy.
			// if an on-prem registry is configured, but it's an airgap installation, we also don't need to process/copy the images
			// as they will be pushed from the airgap bundle.
			continue
		}

		result, err := processV1Beta2HelmChartImages(opts, &helmChart, chartDir)
		if err != nil {
			return errors.Wrap(err, "failed to process online images")
		}

		checkedImages = append(checkedImages, result.CheckedImages...)
	}

	if len(checkedImages) > 0 {
		opts.KotsKinds.Installation.Spec.KnownImages = append(opts.KotsKinds.Installation.Spec.KnownImages, checkedImages...)

		if err := kotsutil.SaveInstallation(&opts.KotsKinds.Installation, opts.Upstream.GetUpstreamDir(opts.WriteUpstreamOptions)); err != nil {
			return errors.Wrap(err, "failed to save installation")
		}
	}

	return nil
}

type WriteRenderedV1Beta2HelmChartsOptions struct {
	HelmDir             string
	RenderedDir         string
	Log                 *logger.CLILogger
	Downstreams         []string
	KotsKinds           *kotsutil.KotsKinds
	ProcessImageOptions image.ProcessImageOptions
}

// WriteRenderedV1Beta2HelmCharts writes the rendered v1beta2 helm charts to the rendered directory for diffing
func WriteRenderedV1Beta2HelmCharts(opts WriteRenderedV1Beta2HelmChartsOptions) error {
	if opts.KotsKinds == nil || opts.KotsKinds.V1Beta2HelmCharts == nil {
		return nil
	}

	for _, downstream := range opts.Downstreams {
		for _, helmChart := range opts.KotsKinds.V1Beta2HelmCharts.Items {
			if !helmChart.Spec.Exclude.IsEmpty() {
				exclude, err := helmChart.Spec.Exclude.Boolean()
				if err != nil {
					return errors.Wrap(err, "failed to parse exclude boolean")
				}

				if exclude {
					continue
				}
			}

			// template the chart with the values to the rendered dir for the downstream
			renderedPath := path.Join(opts.RenderedDir, downstream, "helm", helmChart.GetDirName())
			chartDir := path.Join(opts.HelmDir, helmChart.GetDirName())
			valuesPath := path.Join(chartDir, "values.yaml")
			if err := templateV1Beta2HelmChartWithValuesToDir(&helmChart, chartDir, valuesPath, renderedPath, opts.Log.Debug); err != nil {
				return errors.Wrap(err, "failed to template helm chart for rendered dir")
			}
		}
	}

	return nil
}

func templateV1Beta2HelmChartWithValuesToDir(helmChart *kotsv1beta2.HelmChart, chartDir, valuesPath, outputDir string, log func(string, ...interface{})) error {
	cfg := &action.Configuration{
		Log: log,
	}
	client := action.NewInstall(cfg)
	client.DryRun = true
	client.ReleaseName = helmChart.GetReleaseName()
	client.Replace = true
	client.ClientOnly = true
	client.IncludeCRDs = true

	client.Namespace = helmChart.Spec.Namespace
	if client.Namespace == "" {
		client.Namespace = util.PodNamespace
	}

	chartPath := path.Join(chartDir, fmt.Sprintf("%s-%s.tgz", helmChart.Spec.Chart.Name, helmChart.Spec.Chart.ChartVersion))

	chartRequested, err := loader.Load(chartPath)
	if err != nil {
		return errors.Wrap(err, "failed to load chart")
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			return errors.Wrap(err, "failed dependency check")
		}
	}

	values, err := chartutil.ReadValuesFile(valuesPath)
	if err != nil {
		return errors.Wrap(err, "failed to read values file")
	}

	rel, err := client.Run(chartRequested, values)
	if err != nil {
		return errors.Wrap(err, "failed to run helm install")
	}

	var manifests bytes.Buffer
	fmt.Fprintln(&manifests, strings.TrimSpace(rel.Manifest))
	for _, m := range rel.Hooks {
		fmt.Fprintf(&manifests, "---\n# Source: %s\n%s\n", m.Path, m.Manifest)
	}
	// add hooks
	for _, m := range rel.Hooks {
		fmt.Fprintf(&manifests, "---\n# Source: %s\n%s\n", m.Path, m.Manifest)
	}

	if err := os.MkdirAll(outputDir, 0744); err != nil {
		return errors.Wrap(err, "failed to create rendered path")
	}

	err = os.WriteFile(filepath.Join(outputDir, "all.yaml"), manifests.Bytes(), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write all.yaml")
	}

	return nil
}

func processV1Beta2HelmChartImages(opts WriteV1Beta2HelmChartsOptions, helmChart *kotsv1beta2.HelmChart, chartDir string) (*base.RewriteImagesResult, error) {
	// template the chart with the builder values to a temp dir and then process images
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("kots-images-%s", helmChart.GetDirName()))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir for image processing")
	}
	defer os.RemoveAll(tmpDir)

	builderHelmValues, err := helmChart.GetBuilderValues()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get builder values for chart")
	}

	builderValuesContent, err := yaml.Marshal(builderHelmValues)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal builder values")
	}

	builderValuesPath := path.Join(tmpDir, "builder-values.yaml")
	if err := os.WriteFile(builderValuesPath, builderValuesContent, 0644); err != nil {
		return nil, errors.Wrap(err, "failed to write builder values file")
	}

	templatedOutputDir := path.Join(tmpDir, helmChart.GetDirName())
	if err := os.Mkdir(templatedOutputDir, 0755); err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir for image processing")
	}

	if err := templateV1Beta2HelmChartWithValuesToDir(helmChart, chartDir, builderValuesPath, templatedOutputDir, opts.RenderOptions.Log.Debug); err != nil {
		return nil, errors.Wrap(err, "failed to template helm chart for image processing")
	}

	var dockerHubRegistryCreds registry.Credentials
	dockerhubSecret, _ := registry.GetDockerHubPullSecret(opts.Clientset, util.PodNamespace, opts.ProcessImageOptions.Namespace, opts.ProcessImageOptions.AppSlug)
	if dockerhubSecret != nil {
		dockerHubRegistryCreds, _ = registry.GetCredentialsForRegistryFromConfigJSON(dockerhubSecret.Data[".dockerconfigjson"], registry.DockerHubRegistryName)
	}

	var destRegistry *dockerregistrytypes.RegistryOptions
	if opts.ProcessImageOptions.RewriteImages {
		destRegistry = &dockerregistrytypes.RegistryOptions{
			Endpoint:  opts.ProcessImageOptions.RegistrySettings.Hostname,
			Namespace: opts.ProcessImageOptions.RegistrySettings.Namespace,
			Username:  opts.ProcessImageOptions.RegistrySettings.Username,
			Password:  opts.ProcessImageOptions.RegistrySettings.Password,
		}
	}

	baseImages, err := image.FindImagesInDir(templatedOutputDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to find base images")
	}

	kotsKindsImages, err := kotsutil.GetImagesFromKotsKinds(opts.KotsKinds, destRegistry)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get images from kots kinds")
	}

	result, err := midstream.RewriteBaseImages(opts.ProcessImageOptions, baseImages, kotsKindsImages, opts.KotsKinds, opts.KotsKinds.License, dockerHubRegistryCreds, opts.RenderOptions.Log)
	if err != nil {
		return nil, errors.Wrap(err, "failed to rewrite base images")
	}

	return result, nil
}
