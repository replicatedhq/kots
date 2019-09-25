package helm

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"

	"github.com/pkg/errors"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"sigs.k8s.io/kustomize/v3/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/v3/k8sdeps/transformer"
	"sigs.k8s.io/kustomize/v3/k8sdeps/validator"
	"sigs.k8s.io/kustomize/v3/pkg/commands/build"
	"sigs.k8s.io/kustomize/v3/pkg/fs"
	"sigs.k8s.io/kustomize/v3/pkg/plugins"
	"sigs.k8s.io/kustomize/v3/pkg/resmap"
	"sigs.k8s.io/kustomize/v3/pkg/resource"
)

type MakeHelmChartOptions struct {
	KotsAppDir       string
	KustomizationDir string
	RenderDir        string
}

func MakeHelmChart(options MakeHelmChartOptions) error {
	_, chartName := path.Split(options.KotsAppDir)

	kustomizeBuildOutput, err := ioutil.TempDir("", "kots")
	if err != nil {
		return errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(kustomizeBuildOutput)

	resourceFactory := resmap.NewFactory(resource.NewFactory(kunstruct.NewKunstructuredFactoryImpl()), transformer.NewFactoryImpl())
	buildOptions := build.NewOptions(options.KustomizationDir, kustomizeBuildOutput)

	pluginConfig := plugins.DefaultPluginConfig()
	pluginLoader := plugins.NewLoader(pluginConfig, resourceFactory)

	var b bytes.Buffer
	if err := buildOptions.RunBuild(&b, validator.NewKustValidator(), fs.MakeRealFS(), resourceFactory, transformer.NewFactoryImpl(), pluginLoader); err != nil {
		return errors.Wrap(err, "failed to run kustomize build")
	}

	// Look for
	// Create the chart
	// This is really raw
	chart := &chart.Metadata{
		Name:        chartName,
		Description: "A Helm chart for Kubernetes",
		Version:     "0.1.0",
		AppVersion:  "1.0",
		ApiVersion:  chartutil.ApiVersionV1,
	}

	_, err = chartutil.Create(chart, options.RenderDir)
	if err != nil {
		return errors.Wrap(err, "failed to render chart")
	}

	return nil
}
