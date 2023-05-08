package helmdeploy

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
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	"gopkg.in/yaml.v2"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"k8s.io/client-go/kubernetes"
	"k8s.io/helm/pkg/chartutil"
)

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
			return errors.Wrap(err, "failed to render local values for chart")
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
	}

	return nil
}

type WriteOptions struct {
	HelmDir             string
	RenderedDir         string
	Log                 *logger.CLILogger
	Downstreams         []string
	KotsKinds           *kotsutil.KotsKinds
	ProcessImageOptions midstream.ProcessImageOptions
	Clientset           kubernetes.Interface
}

// WriteRenderedHelmCharts writes the rendered helm chart to the rendered directory
func WriteRenderedHelmCharts(opts WriteOptions) error {
	if opts.KotsKinds == nil || opts.KotsKinds.V1Beta2HelmCharts == nil {
		return nil
	}

	for _, downstream := range opts.Downstreams {
		for _, helmChart := range opts.KotsKinds.V1Beta2HelmCharts.Items {
			renderedPath, err := renderHelmChart(opts, downstream, &helmChart)
			if err != nil {
				return errors.Wrap(err, "failed to render helm chart")
			}

			if err := processImages(opts, renderedPath); err != nil {
				return errors.Wrap(err, "failed to process images")
			}
		}
	}

	return nil
}

func renderHelmChart(opts WriteOptions, downstream string, helmChart *kotsv1beta2.HelmChart) (string, error) {
	cfg := &action.Configuration{
		Log: opts.Log.Debug,
	}
	client := action.NewInstall(cfg)
	client.DryRun = true
	client.ReleaseName = helmChart.GetReleaseName()
	client.Replace = true
	client.ClientOnly = true
	client.IncludeCRDs = true

	chartDir := path.Join(opts.HelmDir, helmChart.GetDirName())

	client.Namespace = helmChart.Spec.Namespace
	if client.Namespace == "" {
		client.Namespace = util.PodNamespace
	}

	chartPath := path.Join(chartDir, fmt.Sprintf("%s-%s.tgz", helmChart.Spec.Chart.Name, helmChart.Spec.Chart.ChartVersion))
	valuesPath := path.Join(chartDir, "values.yaml")

	chartRequested, err := loader.Load(chartPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to load chart")
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			return "", errors.Wrap(err, "failed dependency check")
		}
	}

	values, err := chartutil.ReadValuesFile(valuesPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read values file")
	}

	rel, err := client.Run(chartRequested, values)
	if err != nil {
		return "", errors.Wrap(err, "failed to run helm install")
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

	renderedPath := path.Join(opts.RenderedDir, downstream, "helm", helmChart.GetDirName())
	if err := os.MkdirAll(renderedPath, 0744); err != nil {
		return "", errors.Wrap(err, "failed to create rendered path")
	}

	err = os.WriteFile(filepath.Join(renderedPath, "all.yaml"), manifests.Bytes(), 0644)
	if err != nil {
		return "", errors.Wrap(err, "failed to write all.yaml")
	}

	return renderedPath, nil
}

// processImages will pull any public images from online and copy them to the configured private registry
func processImages(opts WriteOptions, renderedPath string) error {
	if !opts.ProcessImageOptions.RewriteImages || opts.ProcessImageOptions.AirgapRoot != "" {
		return nil
	}

	var dockerHubRegistryCreds registry.Credentials
	dockerhubSecret, _ := registry.GetDockerHubPullSecret(opts.Clientset, util.PodNamespace, opts.ProcessImageOptions.Namespace, opts.ProcessImageOptions.AppSlug)
	if dockerhubSecret != nil {
		dockerHubRegistryCreds, _ = registry.GetCredentialsForRegistryFromConfigJSON(dockerhubSecret.Data[".dockerconfigjson"], registry.DockerHubRegistryName)
	}

	_, err := midstream.RewriteBaseImages(opts.ProcessImageOptions, renderedPath, opts.KotsKinds, opts.KotsKinds.License, dockerHubRegistryCreds, opts.Log)
	if err != nil {
		return errors.Wrap(err, "failed to rewrite base images")
	}

	return nil
}
