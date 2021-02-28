package base

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
)

var (
	HelmV3ManifestNameRegex = regexp.MustCompile("^# Source: (.+)")
)

func renderHelmV3(chartName string, chartPath string, vals map[string]interface{}, renderOptions *RenderOptions) (map[string]string, error) {
	cfg := &action.Configuration{
		Log: renderOptions.Log.Debug,
	}
	client := action.NewInstall(cfg)
	client.DryRun = true
	client.ReleaseName = "RELEASE-NAME"
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

	splitManifests := splitManifests(manifests.String())
	manifestName := ""
	for _, manifest := range splitManifests {
		submatch := HelmV3ManifestNameRegex.FindStringSubmatch(manifest)
		if len(submatch) > 0 {
			// multi-doc manifests will not have the Source comment so use the previous name
			manifestName = strings.TrimPrefix(submatch[1], fmt.Sprintf("%s/", chartName))
		}
		if manifestName == "" {
			// if the manifest name is empty im not sure what to do with the doc
			continue
		} else if strings.TrimSpace(manifest) == "" {
			// filter out empty docs
			continue
		}
		resources[manifestName] = append(resources[manifestName], manifest)
	}

	multidocResources := map[string]string{}
	for manifestName, manifests := range resources {
		multidocResources[manifestName] = strings.Join(manifests, "\n---\n")
	}
	return multidocResources, nil
}

var sep = regexp.MustCompile("(?:^|\\s*\n)---\\s*")

func splitManifests(bigFile string) []string {
	res := []string{}
	// Making sure that any extra whitespace in YAML stream doesn't interfere in splitting documents correctly.
	bigFileTmp := strings.TrimSpace(bigFile)
	docs := sep.Split(bigFileTmp, -1)
	for _, d := range docs {
		d = strings.TrimSpace(d)
		res = append(res, d)
	}
	return res
}
