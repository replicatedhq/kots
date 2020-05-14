package base

import (
	"io/ioutil"
	golog "log"
	"os"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	"k8s.io/helm/pkg/timeconv"
)

func renderHelmV2(chartName string, chartPath string, vals map[string]interface{}, renderOptions *RenderOptions) (map[string]string, error) {
	marshalledVals, err := yaml.Marshal(vals)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal helm values")
	}

	config := &chart.Config{Raw: string(marshalledVals), Values: map[string]*chart.Value{}}

	c, err := chartutil.Load(chartPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load chart")
	}

	renderOpts := renderutil.Options{
		ReleaseOptions: chartutil.ReleaseOptions{
			Name:      chartName,
			IsInstall: true,
			IsUpgrade: false,
			Time:      timeconv.Now(),
			Namespace: renderOptions.Namespace,
		},
		KubeVersion: "1.16.0",
	}

	// Silence the go logger because helm will complain about some of our template strings
	golog.SetOutput(ioutil.Discard)
	defer golog.SetOutput(os.Stdout)
	return renderutil.Render(c, config, renderOpts)
}
