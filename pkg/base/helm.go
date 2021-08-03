package base

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	"helm.sh/helm/v3/pkg/strvals"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

func RenderHelm(u *upstreamtypes.Upstream, renderOptions *RenderOptions) (*Base, error) {
	chartPath, err := ioutil.TempDir("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chart dir")
	}
	defer os.RemoveAll(chartPath)

	for _, file := range u.Files {
		p := path.Join(chartPath, file.Path)
		d, fileName := path.Split(p)
		if _, err := os.Stat(d); err != nil {
			if os.IsNotExist(err) {
				if err := os.MkdirAll(d, 0755); err != nil {
					return nil, errors.Wrap(err, "failed to mkdir for chart resource")
				}
			} else {
				return nil, errors.Wrap(err, "failed to check if dir exists")
			}
		}

		// check chart.yaml for Helm version if a helm version has not been explicitly provided
		if strings.EqualFold(fileName, "Chart.yaml") && renderOptions.HelmVersion == "" {
			renderOptions.HelmVersion, err = checkChartForVersion(&file)
			if err != nil {
				renderOptions.Log.Info("could not determine helm version (will use helm v2 by default): %v", err)
			} else {
				renderOptions.Log.Info("rendering with Helm %v", renderOptions.HelmVersion)
			}
		}

		if err := ioutil.WriteFile(p, file.Content, 0644); err != nil {
			return nil, errors.Wrap(err, "failed to write chart file")
		}
	}

	vals := renderOptions.HelmValues
	for _, value := range renderOptions.HelmOptions {
		if err := strvals.ParseInto(value, vals); err != nil {
			return nil, errors.Wrapf(err, "failed to parse helm value %q", value)
		}
	}

	var rendered []BaseFile
	switch strings.ToLower(renderOptions.HelmVersion) {
	case "v3":
		rendered, err = renderHelmV3(u.Name, chartPath, vals, renderOptions)
		if err != nil {
			return nil, errors.Wrap(err, "failed to render with helm v3")
		}
	case "v2", "":
		rendered, err = renderHelmV2(u.Name, chartPath, vals, renderOptions)
		if err != nil {
			return nil, errors.Wrap(err, "failed to render with helm v2")
		}
	default:
		return nil, errors.Errorf("unknown helmVersion %s", renderOptions.HelmVersion)
	}

	rendered = removeCommonPrefix(rendered) // TODO (ch35027): we should probably target the prefix here, maybe chartPath
	base, err := writeHelmBase(u.Name, rendered, renderOptions)
	if err != nil {
		return nil, errors.Wrapf(err, "write helm chart %s base", u.Name)
	}

	// This will be added back later by renderReplicated
	// I do not want to change the functionality of kots installing a helm chart
	base.Path = ""

	nextBase := helmChartBaseAppendAdditionalFiles(*base, u)

	return &nextBase, nil
}

func writeHelmBase(chartName string, baseFiles []BaseFile, renderOptions *RenderOptions) (*Base, error) {
	rest, crds, subCharts := splitHelmFiles(baseFiles)

	base := &Base{
		Path: path.Join("charts", chartName),
	}
	for _, baseFile := range rest {
		fileBaseFiles, err := writeHelmBaseFile(baseFile, renderOptions)
		if err != nil {
			return nil, errors.Wrapf(err, "write helm base file %s", baseFile.Path)
		}
		base.Files = append(base.Files, fileBaseFiles...)
	}

	if len(crds) > 0 {
		crdsBase := Base{
			Path: "crds",
		}
		for _, baseFile := range crds {
			fileBaseFiles, err := writeHelmBaseFile(baseFile, renderOptions)
			if err != nil {
				return nil, errors.Wrapf(err, "write crds helm base file %s", baseFile.Path)
			}
			crdsBase.Files = append(crdsBase.Files, fileBaseFiles...)
		}
		base.Bases = append(base.Bases, crdsBase)
	}

	for _, subChart := range subCharts {
		subChartBase, err := writeHelmBase(subChart.Name, subChart.BaseFiles, renderOptions)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to write helm sub chart %s base", subChart.Name)
		}
		base.Bases = append(base.Bases, *subChartBase)
	}

	return base, nil
}

type subChartBase struct {
	Name      string
	BaseFiles []BaseFile
}

func splitHelmFiles(baseFiles []BaseFile) (rest []BaseFile, crds []BaseFile, subCharts []subChartBase) {
	subChartsIndex := map[string]int{}
	for _, baseFile := range baseFiles {
		dirPrefix := strings.SplitN(baseFile.Path, string(os.PathSeparator), 3)
		if dirPrefix[0] == "charts" && len(dirPrefix) == 3 {
			subChartName := dirPrefix[1]
			index, ok := subChartsIndex[subChartName]
			if !ok {
				index = len(subCharts)
				subCharts = append(subCharts, subChartBase{Name: subChartName})
				subChartsIndex[subChartName] = index
			}
			subCharts[index].BaseFiles = append(subCharts[index].BaseFiles, BaseFile{
				Path:    path.Join(dirPrefix[2:]...),
				Content: baseFile.Content,
			})
		} else if dirPrefix[0] == "crds" {
			crds = append(crds, BaseFile{
				Path:    path.Join(dirPrefix[1:]...),
				Content: baseFile.Content,
			})
		} else {
			rest = append(rest, baseFile)
		}
	}
	return
}

func writeHelmBaseFile(baseFile BaseFile, renderOptions *RenderOptions) ([]BaseFile, error) {
	multiDoc := [][]byte{}
	if renderOptions.SplitMultiDocYAML {
		multiDoc = bytes.Split(baseFile.Content, []byte("\n---\n"))
	} else {
		multiDoc = append(multiDoc, baseFile.Content)
	}

	baseFiles := []BaseFile{}

	for idx, content := range multiDoc {
		filename := baseFile.Path
		if len(multiDoc) > 1 {
			filename = strings.TrimSuffix(baseFile.Path, filepath.Ext(baseFile.Path))
			filename = fmt.Sprintf("%s-%d%s", filename, idx+1, filepath.Ext(baseFile.Path))
		}

		baseFile := BaseFile{
			Path:    filename,
			Content: content,
		}
		if err := baseFile.transpileHelmHooksToKotsHooks(); err != nil {
			return nil, errors.Wrap(err, "failed to transpile helm hooks to kots hooks")
		}

		baseFiles = append(baseFiles, baseFile)
	}

	return baseFiles, nil
}

// removeCommonPrefix will remove any common prefix from all files
func removeCommonPrefix(baseFiles []BaseFile) []BaseFile {
	if len(baseFiles) == 0 {
		return baseFiles
	}

	commonPrefix := []string{}

	first := true
	for _, baseFile := range baseFiles {
		if first {
			firstFileDir, _ := path.Split(baseFile.Path)
			commonPrefix = strings.Split(firstFileDir, string(os.PathSeparator))

			first = false
			continue
		}
		d, _ := path.Split(baseFile.Path)
		dirs := strings.Split(d, string(os.PathSeparator))

		commonPrefix = util.CommonSlicePrefix(commonPrefix, dirs)
	}

	cleanedBaseFiles := []BaseFile{}
	for _, baseFile := range baseFiles {
		d, f := path.Split(baseFile.Path)
		d2 := strings.Split(d, string(os.PathSeparator))

		d2 = d2[len(commonPrefix):]
		cleanedBaseFiles = append(cleanedBaseFiles, BaseFile{
			Path:    path.Join(path.Join(d2...), f),
			Content: baseFile.Content,
		})
	}

	return cleanedBaseFiles
}

func helmChartBaseAppendAdditionalFiles(base Base, u *upstreamtypes.Upstream) Base {
	for _, upstreamFile := range u.Files {
		if upstreamFile.Path == path.Join(base.Path, "Chart.yaml") {
			base.AdditionalFiles = append(base.AdditionalFiles, BaseFile{
				Path:    "Chart.yaml",
				Content: upstreamFile.Content,
			})
		}
		if upstreamFile.Path == path.Join(base.Path, "Chart.lock") {
			base.AdditionalFiles = append(base.AdditionalFiles, BaseFile{
				Path:    "Chart.lock",
				Content: upstreamFile.Content,
			})
		}
	}

	var nextBases []Base
	for _, base := range base.Bases {
		base = helmChartBaseAppendAdditionalFiles(base, u)
		nextBases = append(nextBases, base)
	}
	base.Bases = nextBases

	return base
}

func checkChartForVersion(file *upstreamtypes.UpstreamFile) (string, error) {
	var chartValues map[string]interface{}

	err := yaml.Unmarshal(file.Content, &chartValues)
	if err != nil {
		return "", errors.Wrap(err, "failed to unmarshal chart.yaml")
	}
	// note: helm API v2 is equivilent to Helm V3
	if version, ok := chartValues["apiVersion"]; ok && strings.EqualFold(version.(string), "v2") {
		return "v3", nil
	}

	// if no determination is made, assume v2
	return "v2", nil
}

// insert namespace if it's defined in the spec and not already present in the manifests
func kustomizeHelmNamespace(baseFiles []BaseFile, renderOptions *RenderOptions) ([]BaseFile, error) {
	if renderOptions.Namespace == "" {
		return baseFiles, nil
	}

	chartsPath, err := ioutil.TempDir("", "charts")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(chartsPath)

	var updatedBaseFiles []BaseFile
	var kustomizeResources []string
	var kustomizePatches []kustomizetypes.PatchStrategicMerge
	resources := map[string]BaseFile{}
	foundGVKNamesMap := map[string]bool{}
	for _, baseFile := range baseFiles {
		// write temp files for manifests that need a namespace
		gvk, manifest := GetGVKWithNameAndNs(baseFile.Content, renderOptions.Namespace)
		if manifest.APIVersion == "" || manifest.Kind == "" || manifest.Metadata.Name == "" {
			updatedBaseFiles = append(updatedBaseFiles, baseFile)
			continue // ignore invalid resources
		}

		// ignore crds
		if manifest.Kind == "CustomResourceDefinition" {
			updatedBaseFiles = append(updatedBaseFiles, baseFile)
			continue
		}

		if manifest.Metadata.Namespace == "" {
			name := filepath.Base(baseFile.Path)
			tmpFile, err := ioutil.TempFile(chartsPath, name)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to write temp file %v", tmpFile.Name())
			}
			defer tmpFile.Close()

			if _, err := tmpFile.Write(baseFile.Content); err != nil {
				return nil, errors.Wrapf(err, "failed to write temp file %v content", tmpFile.Name())
			}

			if err := tmpFile.Close(); err != nil {
				return nil, errors.Wrapf(err, "failed to close temp file %v", tmpFile.Name())
			}

			if !foundGVKNamesMap[gvk] || gvk == "" {
				resources[gvk] = baseFile
				kustomizeResources = append(kustomizeResources, tmpFile.Name())
				foundGVKNamesMap[gvk] = true
			} else {
				kustomizePatches = append(kustomizePatches, kustomizetypes.PatchStrategicMerge(tmpFile.Name()))
			}
		} else {
			updatedBaseFiles = append(updatedBaseFiles, baseFile)
			continue // don't bother kustomizing the yaml if namespace already exists
		}
	}

	// write kustomization
	kustomization := kustomizetypes.Kustomization{
		TypeMeta: kustomizetypes.TypeMeta{
			APIVersion: "kustomize.config.k8s.io/v1beta1",
			Kind:       "Kustomization",
		},
		Namespace:             renderOptions.Namespace,
		Resources:             kustomizeResources,
		PatchesStrategicMerge: kustomizePatches,
	}
	b, err := yaml.Marshal(kustomization)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal kustomization")
	}

	err = ioutil.WriteFile(filepath.Join(chartsPath, "kustomization.yaml"), b, 0644)
	if err != nil {
		return nil, errors.Wrap(err, "failed to write kustomization file")
	}

	fsys := filesys.MakeFsOnDisk()
	k := krusty.MakeKustomizer(krusty.MakeDefaultOptions())
	m, err := k.Run(fsys, chartsPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to kustomize %s", chartsPath)
	}
	updatedManifests, err := m.AsYaml()
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert kustomize output to yaml")
	}

	splitManifests := splitManifests(string(updatedManifests))
	for _, manifest := range splitManifests {
		if len(manifest) == 0 {
			continue
		}

		gvk, _ := GetGVKWithNameAndNs([]byte(manifest), renderOptions.Namespace)
		if _, ok := resources[gvk]; !ok {
			return nil, errors.Wrapf(err, "failed to replace base %v", gvk)
		}

		baseFile := resources[gvk]
		baseFile.Content = []byte(manifest)
		updatedBaseFiles = append(updatedBaseFiles, baseFile)
	}

	return updatedBaseFiles, nil
}
