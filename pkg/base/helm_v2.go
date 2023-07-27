package base

import (
	"fmt"
	"io"
	golog "log"
	"os"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/renderutil"
	"k8s.io/helm/pkg/timeconv"
	k8syaml "sigs.k8s.io/yaml"
)

func renderHelmV2(chartName string, chartPath string, vals map[string]interface{}, renderOptions *RenderOptions) ([]BaseFile, []BaseFile, error) {
	marshalledVals, err := yaml.Marshal(vals)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to marshal helm values")
	}

	config := &chart.Config{Raw: string(marshalledVals), Values: map[string]*chart.Value{}}

	c, err := chartutil.Load(chartPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to load chart")
	}

	coalescedValues, err := chartutil.CoalesceValues(c, config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to coalesce values")
	}

	valuesContent, err := k8syaml.Marshal(coalescedValues)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to marshal rendered values")
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
	if renderOpts.ReleaseOptions.Namespace == "" {
		renderOpts.ReleaseOptions.Namespace = NamespaceTemplateConst
	}

	// Silence the go logger because helm will complain about some of our template strings
	golog.SetOutput(io.Discard)
	defer golog.SetOutput(os.Stdout)

	rendered, err := renderutil.Render(c, config, renderOpts)
	if err != nil {
		return nil, nil, util.ActionableError{
			NoRetry: true,
			Message: fmt.Sprintf("helm v2 render failed with error: %v", err),
		}
	}

	baseFiles := []BaseFile{}
	additionalFiles := []BaseFile{
		{
			Path:    "values.yaml",
			Content: valuesContent,
		},
	}

	// need to split base files before inserting namespace
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
	baseFiles, err = kustomizeHelmNamespace(baseFiles, renderOptions)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to insert helm namespace")
	}

	// maintain order
	return mergeBaseFiles(baseFiles), additionalFiles, nil
}
