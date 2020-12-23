package base

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/releaseutil"
)

var (
	HelmV3ManifestNameRegex = regexp.MustCompile("^# Source: [^/]+/(.+)\n")
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
	client.IncludeCRDs = true
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

	resources := map[string][]string{}

	splitManifests := releaseutil.SplitManifests(manifests.String())
	for _, manifest := range splitManifests {
		submatch := HelmV3ManifestNameRegex.FindStringSubmatch(manifest)
		if len(submatch) == 0 {
			continue
		}
		manifestName := submatch[1]
		resources[manifestName] = append(resources[manifestName], HelmV3ManifestNameRegex.ReplaceAllString(manifest, ""))
	}

	multidocResources := map[string]string{}
	for manifestName, manifests := range resources {
		multidocResources[manifestName] = strings.Join(manifests, "\n---\n")
	}
	return multidocResources, nil
}
