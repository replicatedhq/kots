package base

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/releaseutil"
)

func renderHelmV3(chartName string, chartPath string, vals map[string]interface{}, renderOptions *RenderOptions) (map[string]string, error) {
	cfg := &action.Configuration{
		Log: renderOptions.Log.Debug,
	}
	client := action.NewInstall(cfg)
	client.DryRun = true
	client.ReleaseName = chartName
	client.Replace = true
	client.ClientOnly = true
	// client.IncludeCRDs = includeCrds
	client.Namespace = renderOptions.Namespace

	chartRequested, err := loader.Load(chartPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load chart")
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			return nil, errors.Wrap(err, "failed dependency check")
		}
	}

	rel, err := client.Run(chartRequested, vals)
	if err != nil {
		return nil, errors.Wrap(err, "failed to render chart")
	}

	var manifests bytes.Buffer
	fmt.Fprintln(&manifests, strings.TrimSpace(rel.Manifest))
	for _, m := range rel.Hooks {
		fmt.Fprintf(&manifests, "---\n# Source: %s\n%s\n", m.Path, m.Manifest)
	}

	return releaseutil.SplitManifests(manifests.String()), nil
}
