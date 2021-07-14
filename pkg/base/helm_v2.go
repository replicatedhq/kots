package base

import (
	"io/ioutil"
	golog "log"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	"k8s.io/helm/pkg/timeconv"
)

func renderHelmV2(chartName string, chartPath string, vals map[string]interface{}, renderOptions *RenderOptions) ([]BaseFile, error) {
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

	rendered, err := renderutil.Render(c, config, renderOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render chart")
	}

	// need to split base files before inserting namespace
	baseFiles := []BaseFile{}
	for name, content := range rendered {
		splitManifests := splitManifests(content)
		for _, manifest := range splitManifests {
			if strings.TrimSpace(manifest) == "" {
				// filter out empty docs
				continue
			}
			baseFiles = append(baseFiles, BaseFile{
				Path:    name,
				Content: []byte(manifest),
			})
		}
	}

	// insert namespace defined in the HelmChart spec
	baseFiles, err = insertHelmNamespace(baseFiles, renderOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to insert helm namespace")
	}

	// maintain order
	return mergeBaseFiles(baseFiles), nil
}
