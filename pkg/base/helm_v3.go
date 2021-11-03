package base

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	rspb "helm.sh/helm/v3/pkg/release"
	helmtime "helm.sh/helm/v3/pkg/time"
	k8syaml "sigs.k8s.io/yaml"
)

var (
	HelmV3ManifestNameRegex = regexp.MustCompile("^# Source: (.+)")
)

const NamespaceTemplateConst = "repl{{ Namespace}}"

func renderHelmV3(chartName string, chartPath string, vals map[string]interface{}, renderOptions *RenderOptions) ([]BaseFile, error) {
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
	if client.Namespace == "" {
		client.Namespace = NamespaceTemplateConst
	}

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

	baseFiles := []BaseFile{}

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
		baseFiles = append(baseFiles, BaseFile{
			Path:    manifestName,
			Content: []byte(manifest),
		})
	}

	// this secret should only be generated for installs that rely on us rendering yaml internally - not native helm installs
	// those generate their own secret
	if !renderOptions.UseHelmInstall {
		if renderOptions.Namespace == "" {
			rel.Namespace = namespace()
		}
		rel.Info.Status = rspb.StatusDeployed

		// override deployed times to avoid spurious diffs
		rel.Info.FirstDeployed, _ = helmtime.Parse(time.RFC3339, "1970-01-01T01:00:00")
		rel.Info.LastDeployed, _ = helmtime.Parse(time.RFC3339, "1970-01-01T01:00:00")

		helmReleaseSecretObj, err := newSecretsObject(rel)
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate helm secret")
		}

		renderedSecret, err := k8syaml.Marshal(helmReleaseSecretObj)
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate helm secret")
		}
		baseFiles = append(baseFiles, BaseFile{
			Path:    "chartHelmSecret.yaml",
			Content: renderedSecret,
		})
	}

	// insert namespace defined in the HelmChart spec
	baseFiles, err = kustomizeHelmNamespace(baseFiles, renderOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to insert helm namespace")
	}

	// maintain order
	return mergeBaseFiles(baseFiles), nil
}

func mergeBaseFiles(baseFiles []BaseFile) []BaseFile {
	merged := []BaseFile{}
	found := map[string]int{}
	for _, baseFile := range baseFiles {
		index, ok := found[baseFile.Path]
		if ok {
			merged[index].Content = append(merged[index].Content, []byte("\n---\n")...)
			merged[index].Content = append(merged[index].Content, baseFile.Content...)
		} else {
			found[baseFile.Path] = len(merged)
			merged = append(merged, baseFile)
		}
	}
	return merged
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

func namespace() string {
	// this is really only useful when called via the ffi function from kotsadm
	// because that namespace is not configurable otherwise
	if os.Getenv("DEV_NAMESPACE") != "" {
		return os.Getenv("DEV_NAMESPACE")
	}

	return util.PodNamespace
}
