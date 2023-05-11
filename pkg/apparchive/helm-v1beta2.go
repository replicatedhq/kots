package apparchive

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	kotsv1beta2 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta2"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
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

// WriteV1Beta2HelmCharts copies the upstream helm chart archive and rendered values to the helm directory
func WriteV1Beta2HelmCharts(u *upstreamtypes.Upstream, renderOptions *base.RenderOptions, helmDir string, helmCharts []*kotsv1beta2.HelmChart) error {
	for _, helmChart := range helmCharts {
		chartDir := path.Join(helmDir, helmChart.GetDirName())
		if err := os.MkdirAll(chartDir, 0744); err != nil {
			return errors.Wrap(err, "failed to create chart dir")
		}

		archive, err := base.FindHelmChartArchiveInRelease(u.Files, helmChart)
		if err != nil {
			return errors.Wrap(err, "failed to find helm chart archive in release")
		}

		archivePath := path.Join(chartDir, fmt.Sprintf("%s-%s.tgz", helmChart.Spec.Chart.Name, helmChart.Spec.Chart.ChartVersion))
		if err := ioutil.WriteFile(archivePath, archive, 0644); err != nil {
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

		builder, _, err := base.NewConfigContextTemplateBuilder(u, renderOptions)
		if err != nil {
			return errors.Wrap(err, "failed to create config context template builder")
		}

		renderedValuesContent, err := builder.RenderTemplate(fmt.Sprintf("%s-values", helmChart.GetDirName()), string(valuesContent))
		if err != nil {
			return errors.Wrap(err, "failed to render values")
		}

		valuesPath := path.Join(chartDir, "values.yaml")
		if err := ioutil.WriteFile(valuesPath, []byte(renderedValuesContent), 0644); err != nil {
			return errors.Wrap(err, "failed to write values file")
		}

		builderValues := helmChart.Spec.Builder
		if builderValues == nil {
			builderValues = map[string]kotsv1beta2.MappedChartValue{}
		}

		builderHelmValues, err := helmChart.Spec.GetHelmValues(builderValues)
		if err != nil {
			return errors.Wrap(err, "failed to get builder values for chart")
		}

		builderValuesContent, err := yaml.Marshal(builderHelmValues)
		if err != nil {
			return errors.Wrap(err, "failed to marshal builder values")
		}

		renderedBuilderValuesContent, err := builder.RenderTemplate(fmt.Sprintf("%s-builder-values", helmChart.GetDirName()), string(builderValuesContent))
		if err != nil {
			return errors.Wrap(err, "failed to render builder values")
		}

		builderValuesPath := path.Join(chartDir, "builder-values.yaml")
		if err := ioutil.WriteFile(builderValuesPath, []byte(renderedBuilderValuesContent), 0644); err != nil {
			return errors.Wrap(err, "failed to write builder values file")
		}
	}

	return nil
}

type HelmWriteOptions struct {
	HelmDir             string
	RenderedDir         string
	Log                 *logger.CLILogger
	Downstreams         []string
	KotsKinds           *kotsutil.KotsKinds
	ProcessImageOptions image.ProcessImageOptions
	Clientset           kubernetes.Interface
}

// WriteRenderedHelmCharts writes the rendered helm chart to the rendered directory and processes the images
func WriteRenderedHelmCharts(opts HelmWriteOptions) error {
	if opts.KotsKinds == nil || opts.KotsKinds.V1Beta2HelmCharts == nil {
		return nil
	}

	for _, downstream := range opts.Downstreams {
		for _, helmChart := range opts.KotsKinds.V1Beta2HelmCharts.Items {
			// template the chart with the values to the rendered dir for the downstream
			renderedPath := path.Join(opts.RenderedDir, downstream, "helm", helmChart.GetDirName())
			if err := templateHelmChartWithValuesToDir(opts.HelmDir, &helmChart, "values.yaml", renderedPath, opts.Log.Debug); err != nil {
				return errors.Wrap(err, "failed to template helm chart for rendered dir")
			}

			if !opts.ProcessImageOptions.RewriteImages || opts.ProcessImageOptions.AirgapRoot != "" {
				// if an on-prem registry is not configured (which means it's an online installation)
				// there's no need to process/copy the images as they will be pulled from their original registries or through the replicated proxy.
				// if an on-prem registry is configured, but it's an airgap installation, we also don't need to process/copy the images
				// as they will be pushed from the airgap bundle.
				continue
			}

			// template the chart with the builder values to a temp dir and then process images
			imageProcessingPath, err := os.MkdirTemp("", fmt.Sprintf("kots-images-%s", helmChart.GetDirName()))
			if err != nil {
				return errors.Wrap(err, "failed to create temp dir for image processing")
			}
			defer os.RemoveAll(imageProcessingPath)

			if err := templateHelmChartWithValuesToDir(opts.HelmDir, &helmChart, "builder-values.yaml", imageProcessingPath, opts.Log.Debug); err != nil {
				return errors.Wrap(err, "failed to template helm chart for image processing")
			}

			if err := processImages(opts, imageProcessingPath); err != nil {
				return errors.Wrap(err, "failed to process images")
			}
		}
	}

	return nil
}

func templateHelmChartWithValuesToDir(helmDir string, helmChart *kotsv1beta2.HelmChart, valuesFile, outputDir string, log func(string, ...interface{})) error {
	cfg := &action.Configuration{
		Log: log,
	}
	client := action.NewInstall(cfg)
	client.DryRun = true
	client.ReleaseName = helmChart.GetReleaseName()
	client.Replace = true
	client.ClientOnly = true
	client.IncludeCRDs = true

	chartDir := path.Join(helmDir, helmChart.GetDirName())

	client.Namespace = helmChart.Spec.Namespace
	if client.Namespace == "" {
		client.Namespace = util.PodNamespace
	}

	chartPath := path.Join(chartDir, fmt.Sprintf("%s-%s.tgz", helmChart.Spec.Chart.Name, helmChart.Spec.Chart.ChartVersion))
	valuesPath := path.Join(chartDir, valuesFile)

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

// processImages will pull all images (public and private) from online and copy them to the configured private registry
func processImages(opts HelmWriteOptions, renderedPath string) error {
	var dockerHubRegistryCreds registry.Credentials
	dockerhubSecret, _ := registry.GetDockerHubPullSecret(opts.Clientset, util.PodNamespace, opts.ProcessImageOptions.Namespace, opts.ProcessImageOptions.AppSlug)
	if dockerhubSecret != nil {
		dockerHubRegistryCreds, _ = registry.GetCredentialsForRegistryFromConfigJSON(dockerhubSecret.Data[".dockerconfigjson"], registry.DockerHubRegistryName)
	}

	_, err := image.RewriteBaseImages(opts.ProcessImageOptions, renderedPath, opts.KotsKinds, opts.KotsKinds.License, dockerHubRegistryCreds, opts.Log)
	if err != nil {
		return errors.Wrap(err, "failed to rewrite base images")
	}

	return nil
}
