package base

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
	"helm.sh/helm/v4/pkg/action"
	"helm.sh/helm/v4/pkg/chart"
	chartutil "helm.sh/helm/v4/pkg/chart/common/util"
	"helm.sh/helm/v4/pkg/chart/v2/loader"
	"helm.sh/helm/v4/pkg/release/common"
	relv1 "helm.sh/helm/v4/pkg/release/v1"
	k8syaml "sigs.k8s.io/yaml"
)

func renderHelmV4(releaseName string, chartPath string, renderOptions *RenderOptions) ([]BaseFile, []BaseFile, error) {
	cfg := action.NewConfiguration()

	client := action.NewInstall(cfg)
	// DryRunClient replaces the Helm 3 combination of DryRun=true + ClientOnly=true.
	// It renders the chart without contacting the Kubernetes API server.
	client.DryRunStrategy = action.DryRunClient
	client.ReleaseName = releaseName
	client.Replace = true

	client.Namespace = renderOptions.Namespace
	if client.Namespace == "" {
		client.Namespace = NamespaceTemplateConst
	}

	chartRequested, err := loader.Load(chartPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to load chart")
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		// chart.Dependency is interface{}, so we need to convert []*v2chart.Dependency
		deps := make([]chart.Dependency, len(req))
		for i, d := range req {
			deps[i] = d
		}
		if err := action.CheckDependencies(chartRequested, deps); err != nil {
			return nil, nil, errors.Wrap(err, "failed dependency check")
		}
	}

	relIface, err := client.Run(chartRequested, renderOptions.HelmValues)
	if err != nil {
		return nil, nil, util.ActionableError{
			NoRetry: true,
			Message: fmt.Sprintf("helm v4 render failed with error: %v", err),
		}
	}

	rel, ok := relIface.(*relv1.Release)
	if !ok {
		return nil, nil, errors.New("helm v4 returned unexpected release type")
	}

	coalescedValues, err := chartutil.CoalesceValues(chartRequested, rel.Config)
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
		fmt.Fprintf(&manifests, "---\n# Source: %s\n%s\n", crd.Filename, string(crd.File.Data))
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
		rel.Info.Status = common.StatusDeployed

		// override deployed times to avoid spurious diffs.
		// Zero time.Time values are omitted from JSON marshaling by the v4 Info struct.
		rel.Info.FirstDeployed = time.Time{}
		rel.Info.LastDeployed = time.Time{}

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
