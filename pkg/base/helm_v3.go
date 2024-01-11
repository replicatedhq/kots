package base

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	rspb "helm.sh/helm/v3/pkg/release"
	helmtime "helm.sh/helm/v3/pkg/time"
	k8syaml "sigs.k8s.io/yaml"
)

var (
	HelmV3ManifestNameRegex = regexp.MustCompile("^# Source: (.+)")
)

const NamespaceTemplateConst = "repl{{ Namespace}}"

func renderHelmV3(releaseName string, chartPath string, renderOptions *RenderOptions) ([]BaseFile, []BaseFile, error) {
	cfg := &action.Configuration{
		Log: renderOptions.Log.Debug,
	}
	client := action.NewInstall(cfg)
	client.DryRun = true
	client.ReleaseName = releaseName
	client.Replace = true
	client.ClientOnly = true

	client.Namespace = renderOptions.Namespace
	if client.Namespace == "" {
		client.Namespace = NamespaceTemplateConst
	}

	chartRequested, err := loader.Load(chartPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to load chart")
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			return nil, nil, errors.Wrap(err, "failed dependency check")
		}
	}

	rel, err := client.Run(chartRequested, renderOptions.HelmValues)
	if err != nil {
		return nil, nil, util.ActionableError{
			NoRetry: true,
			Message: fmt.Sprintf("helm v3 render failed with error: %v", err),
		}
	}

	coalescedValues, err := chartutil.CoalesceValues(rel.Chart, rel.Config)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to coalesce values")
	}

	valuesContent, err := k8syaml.Marshal(coalescedValues)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to marshal rendered values")
	}

	baseFiles := []BaseFile{}
	additionalFiles := []BaseFile{
		{
			Path:    "values.yaml",
			Content: valuesContent,
		},
	}

	var manifests bytes.Buffer
	fmt.Fprintln(&manifests, strings.TrimSpace(rel.Manifest))
	// add hooks
	for _, m := range rel.Hooks {
		fmt.Fprintf(&manifests, "---\n# Source: %s\n%s\n", m.Path, m.Manifest)
	}
	// add crds
	for _, crd := range chartRequested.CRDObjects() {
		fmt.Fprintf(&manifests, "---\n# Source: %s\n%s\n", crd.Filename, string(crd.File.Data[:]))
	}

	splitManifests := splitManifests(manifests.String())
	manifestName := ""
	for _, manifest := range splitManifests {
		submatch := HelmV3ManifestNameRegex.FindStringSubmatch(manifest)
		if len(submatch) > 0 {
			// multi-doc manifests will not have the Source comment so use the previous name
			manifestName = strings.TrimPrefix(submatch[1], fmt.Sprintf("%s/", chartRequested.Name()))
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

	// Don't change the classic style rendering ie, picking all the files within charts, subdirs
	// and do a single apply. This will not work for Native helm expects uniquely named image pullsecrets.
	// helm maintains strict ownership of secretnames for each subcharts to add Release metadata for each chart
	if !renderOptions.UseHelmInstall {
		baseFiles = removeCommonPrefix(baseFiles)
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
			return nil, nil, errors.Wrap(err, "failed to generate helm secret")
		}

		renderedSecret, err := k8syaml.Marshal(helmReleaseSecretObj)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to generate helm secret")
		}
		baseFiles = append(baseFiles, BaseFile{
			Path:    "chartHelmSecret.yaml",
			Content: renderedSecret,
		})
	}

	// insert namespace defined in the HelmChart spec
	baseFiles, err = kustomizeHelmNamespace(baseFiles, renderOptions)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to insert helm namespace")
	}

	// ensure order
	merged := mergeBaseFiles(baseFiles)
	sort.Slice(merged, func(i, j int) bool {
		return 0 > strings.Compare(merged[i].Path, merged[j].Path)
	})
	return merged, additionalFiles, nil
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
		d = strings.TrimSpace(d) + "\n"
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
