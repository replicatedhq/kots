package base

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	"helm.sh/helm/v3/pkg/strvals"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/krusty"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

func RenderHelm(u *upstreamtypes.Upstream, renderOptions *RenderOptions) (*Base, error) {
	chartPath, err := os.MkdirTemp("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chart dir")
	}
	defer os.RemoveAll(chartPath)

	for _, file := range u.Files {
		p := path.Join(chartPath, file.Path)
		d, _ := path.Split(p)
		if _, err := os.Stat(d); err != nil {
			if os.IsNotExist(err) {
				if err := os.MkdirAll(d, 0755); err != nil {
					return nil, errors.Wrap(err, "failed to mkdir for chart resource")
				}
			} else {
				return nil, errors.Wrap(err, "failed to check if dir exists")
			}
		}

		if err := os.WriteFile(p, file.Content, 0644); err != nil {
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
	var additional []BaseFile
	switch strings.ToLower(renderOptions.HelmVersion) {
	case "v3", "":
		rendered, additional, err = renderHelmV3(u.Name, chartPath, vals, renderOptions)
		if err != nil {
			return nil, errors.Wrap(err, "failed to render with helm v3")
		}
	case "v2":
		rendered, additional, err = renderHelmV2(u.Name, chartPath, vals, renderOptions)
		if err != nil {
			return nil, errors.Wrap(err, "failed to render with helm v2")
		}
	default:
		return nil, errors.Errorf("unknown helmVersion %s", renderOptions.HelmVersion)
	}

	// Don't change the classic style rendering ie, picking all the files within charts, subdirs
	// and do a single apply. This will not work for Native helm expects uniquely named image pullsecrets.
	// helm maintains strict ownership of secretnames for each subcharts to add Release metadata for each chart
	if !renderOptions.UseHelmInstall {
		rendered = removeCommonPrefix(rendered)
	}

	base, err := writeHelmBase(u.Name, rendered, renderOptions)
	if err != nil {
		return nil, errors.Wrapf(err, "write helm chart %s base", u.Name)
	}

	base.AdditionalFiles = append(base.AdditionalFiles, additional...)

	// This will be added back later by renderReplicated
	// I do not want to change the functionality of kots installing a helm chart
	base.Path = ""

	upstreamFileMap := make(map[string][]byte)
	for _, upstreamFile := range u.Files {
		upstreamFileMap[upstreamFile.Path] = upstreamFile.Content
	}

	upstreamToBasePathsMap := getUpstreamToBasePathsMap(upstreamFileMap)
	baseToUpstreamPathMap := getBaseToUpstreamPathMap(upstreamFileMap, upstreamToBasePathsMap)

	// since we already rendered the helm chart using dry-run mode, we need to add back the Chart.yaml files so that helm install can work
	nextBase := helmChartBaseAppendAdditionalFiles(*base, base.Path, upstreamFileMap, baseToUpstreamPathMap)
	nexterBase := helmChartBaseAppendMissingDependencies(nextBase, u.Files, upstreamToBasePathsMap, baseToUpstreamPathMap)

	return &nexterBase, nil
}

func writeHelmBase(chartName string, baseFiles []BaseFile, renderOptions *RenderOptions) (*Base, error) {
	rest, crds, subCharts := splitHelmFiles(baseFiles)

	base := &Base{
		Path: path.Join("charts", chartName),
	}
	if renderOptions.UseHelmInstall {
		base.Namespace = renderOptions.Namespace
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

// shouldMapUpstreamPath returns true if it's a Chart.yaml file and it exists in one of:
// - the root of the chart (the parent chart)
// - a sub-chart in a 'charts' directory
func shouldMapUpstreamPath(upstreamPath string) bool {
	parts := strings.Split(upstreamPath, string(os.PathSeparator))
	if fileName := parts[len(parts)-1]; fileName != "Chart.yaml" {
		return false
	}
	if len(parts) == 1 {
		// this is the parent chart
		return true
	}
	if len(parts) < 3 { // charts/<subchart>/Chart.yaml
		return false
	}
	parentDir := parts[len(parts)-3] // charts/<subchart>/Chart.yaml -> charts ... this applies to all nested subcharts too
	return parentDir == "charts"
}

// creates a map of the upstream chart paths and their cooresponding base paths
func getUpstreamToBasePathsMap(upstreamFiles map[string][]byte) map[string][]string {
	upstreamToBasePathsMap := make(map[string][]string)
	for upstreamFilePath := range upstreamFiles {
		if !shouldMapUpstreamPath(upstreamFilePath) {
			continue
		}
		upstreamPath := strings.TrimSuffix(upstreamFilePath, "Chart.yaml")        // charts/subchart/Chart.yaml -> charts/subchart/
		upstreamPath = strings.TrimSuffix(upstreamPath, string(os.PathSeparator)) // charts/subchart/ -> charts/subchart
		basePaths, err := helmChartUpstreamPathToBasePaths(upstreamPath, upstreamFiles)
		if err != nil {
			logger.Errorf("failed to get base paths for upstream path %s: %v", upstreamFilePath, err)
		}
		upstreamToBasePathsMap[upstreamPath] = basePaths
	}
	return upstreamToBasePathsMap
}

// creates a map of the base paths and their cooresponding upstream paths
func getBaseToUpstreamPathMap(upstreamFileMap map[string][]byte, upstreamToBasePathsMap map[string][]string) map[string]string {
	baseToUpstreamPathMap := make(map[string]string)
	for upstreamFilePath, basePaths := range upstreamToBasePathsMap {
		for _, basePath := range basePaths {
			baseToUpstreamPathMap[basePath] = upstreamFilePath
		}
	}
	return baseToUpstreamPathMap
}

// adds the Chart.yaml and Chart.lock files from the upstream to the base charts that were rendered
func helmChartBaseAppendAdditionalFiles(base Base, fullBasePath string, upstreamFiles map[string][]byte, baseToUpstreamPathMap map[string]string) Base {
	if upstreamPath, ok := baseToUpstreamPathMap[fullBasePath]; ok {
		additionalFiles := []string{"Chart.yaml", "Chart.lock"}
		for _, additionalFile := range additionalFiles {
			additionalFilePath := path.Join(upstreamPath, additionalFile)
			if content, ok := upstreamFiles[additionalFilePath]; ok {
				base.AdditionalFiles = append(base.AdditionalFiles, BaseFile{
					Path:    additionalFile,
					Content: content,
				})
			}
		}
	} else {
		// not an error since the map only contains information about charts and nested subcharts, not all files.
		logger.Debugf("upstream path not found for base path '%s'", fullBasePath)
	}

	var nextBases []Base
	for _, nextBase := range base.Bases {
		nextBase = helmChartBaseAppendAdditionalFiles(nextBase, path.Join(fullBasePath, nextBase.Path), upstreamFiles, baseToUpstreamPathMap)
		nextBases = append(nextBases, nextBase)
	}
	base.Bases = nextBases

	return base
}

func helmChartUpstreamPathToBasePaths(upstreamPath string, upstreamFileMap map[string][]byte) ([]string, error) {
	charts := pathToCharts(upstreamPath)
	basePaths := []string{""}
	parentChartPath := ""
	// iterate over the subcharts and find any that are aliased by the parent chart
	for i := 1; i < len(charts); i++ {
		subchart := charts[i]
		aliases := []string{}

		// check if the chart is aliased by its parent
		parentChartYaml := filepath.Join(parentChartPath, "Chart.yaml")
		if content, ok := upstreamFileMap[parentChartYaml]; ok {
			deps := new(HelmChartDependencies)
			if err := yaml.Unmarshal(content, deps); err != nil {
				return nil, errors.Wrapf(err, "failed to unmarshal %s", parentChartYaml)
			}
			foundSubchartInDeps := false
			for _, dep := range deps.Dependencies {
				if dep.Name != subchart {
					continue
				}
				foundSubchartInDeps = true
				if dep.Alias != "" {
					aliases = append(aliases, dep.Alias)
				} else {
					aliases = append(aliases, dep.Name)
				}
			}
			if !foundSubchartInDeps {
				// subchart is not managed via dependencies
				aliases = append(aliases, subchart)
			}
		} else {
			return nil, errors.Errorf("failed to find upstream file %s", parentChartYaml)
		}

		newBasePaths := []string{}
		for _, basePath := range basePaths {
			for _, alias := range aliases {
				newBasePath := path.Join(basePath, "charts", alias)
				newBasePaths = append(newBasePaths, newBasePath)
			}
		}
		basePaths = newBasePaths

		parentChartPath = path.Join(parentChartPath, "charts", subchart)
	}

	return basePaths, nil
}

// Takes an input chart path and returns a list that represents the dependency tree for the chart.
// The top-level chart is represented by an empty string.
// // "" => [""]
// // "charts/mariadb" => ["", "mariadb"]
// // "charts/mariadb/charts/common" => ["", "mariadb", "common"]
func pathToCharts(path string) []string {
	re := regexp.MustCompile(`\/?charts\/`)
	return re.Split(path, -1)
}

// look for any sub-chart dependencies that were not rendered and add their Chart.yaml to the base files
func helmChartBaseAppendMissingDependencies(base Base, upstreamFiles []upstreamtypes.UpstreamFile, upstreamToBasePathsMap map[string][]string, baseToUpstreamPathMap map[string]string) Base {
	allBasePaths := getAllBasePaths("", base)

	// create a map of the upstream paths that have been rendered
	renderedUpstreamPaths := map[string]bool{}
	for _, basePath := range allBasePaths {
		if upstreamPath, ok := baseToUpstreamPathMap[basePath]; ok {
			renderedUpstreamPaths[upstreamPath] = true
		}
	}

	for _, upstreamFile := range upstreamFiles {
		if !shouldMapUpstreamPath(upstreamFile.Path) {
			continue
		}
		upstreamPath := strings.TrimSuffix(upstreamFile.Path, "Chart.yaml")
		upstreamPath = strings.TrimSuffix(upstreamPath, string(os.PathSeparator))
		if _, ok := renderedUpstreamPaths[upstreamPath]; ok {
			continue // already rendered this upstream path
		}
		basePaths := upstreamToBasePathsMap[upstreamPath]
		for _, basePath := range basePaths {
			if basePath == "" {
				continue // empty sub-base path is not allowed
			}
			b := Base{
				Path: basePath,
				AdditionalFiles: []BaseFile{
					{
						Path:    "Chart.yaml",
						Content: upstreamFile.Content,
					},
				},
			}
			base.Bases = append(base.Bases, b)
		}
	}

	return base
}

func getAllBasePaths(prefix string, base Base) []string {
	basePaths := []string{path.Join(prefix, base.Path)}
	for _, b := range base.Bases {
		basePaths = append(basePaths, getAllBasePaths(path.Join(prefix, base.Path), b)...)
	}
	return basePaths
}

// insert namespace if it's defined in the spec and not already present in the manifests
func kustomizeHelmNamespace(baseFiles []BaseFile, renderOptions *RenderOptions) ([]BaseFile, error) {
	if renderOptions.Namespace == "" {
		return baseFiles, nil
	}

	chartsPath, err := os.MkdirTemp("", "charts")
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

		// Only add the namespace if the object itself isn't a namespace.
		if manifest.Metadata.Namespace == "" && !(manifest.Kind == "Namespace" && manifest.APIVersion == "v1") {
			name := filepath.Base(baseFile.Path)
			tmpFile, err := os.CreateTemp(chartsPath, name)
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

	err = os.WriteFile(filepath.Join(chartsPath, "kustomization.yaml"), b, 0644)
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
		if strings.TrimSpace(manifest) == "" {
			// filter out empty docs
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

type HelmSubCharts struct {
	ParentName string
	SubCharts  []string
}

type HelmChartDependency struct {
	Alias      string `yaml:"alias"`
	Name       string `yaml:"name"`
	Repository string `yaml:"repository"`
}
type HelmChartDependencies struct {
	Dependencies []HelmChartDependency `yaml:"dependencies"`
}

// Returns a list of HelmSubCharts, each of which contains the name of the parent chart and a list of subcharts
// Each item in the subcharts list is a string of repeating terms the form "charts/<chart name>".
// The first item is just the top level chart (TODO: this should be removed)
// For example:
//   - top-level-chart
//   - charts/top-level-chart
//   - charts/top-level-chart/charts/cool-sub-chart
func FindHelmSubChartsFromBase(baseDir, parentChartName string) (*HelmSubCharts, error) {
	type helmName struct {
		Name string `yaml:"name"`
	}

	charts := make([]string, 0)
	searchDir := filepath.Join(baseDir, "charts", parentChartName)

	// If dependencies in the chart are aliased, they will create new directories with the alias name
	// in the charts folder and need to be excluded when generating the pullsecrets.yaml. It feels like this
	// could replace the logic below that's doing the file tree walking but I'm unsure.
	parentChartPath := filepath.Join(searchDir, "Chart.yaml")
	parentChartRaw, err := os.ReadFile(parentChartPath)
	if err == nil {
		parentChart := new(HelmChartDependencies)
		err = yaml.Unmarshal(parentChartRaw, parentChart)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal parent chart %s", parentChartPath)
		}
		for _, dep := range parentChart.Dependencies {
			if dep.Alias != "" {
				charts = append(charts, dep.Alias)
			} else {
				charts = append(charts, dep.Name)
			}
		}
	}

	err = filepath.Walk(searchDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// ignore anything that's not a chart yaml
			if info.Name() != "Chart.yaml" {
				return nil
			}

			contents, err := os.ReadFile(path)
			if err != nil {
				return errors.Wrap(err, "failed to read file")
			}

			// unmarshal just the name of the chart
			var chartInfo helmName
			err = yaml.Unmarshal(contents, &chartInfo)
			if err != nil {
				return nil
			}

			if chartInfo.Name == "" {
				// probably not a valid chart file
				return nil
			}

			// use directory names because they are unique
			chartName, err := filepath.Rel(baseDir, filepath.Dir(path))
			if err != nil {
				return errors.Wrap(err, "failed to get chart name from path")
			}

			charts = append(charts, chartName)

			return nil
		})
	if err != nil {
		if !strings.Contains(err.Error(), "no such file or directory") {
			return nil, errors.Wrap(err, "failed to walk upstream dir")
		}
	}

	return &HelmSubCharts{
		ParentName: parentChartName,
		SubCharts:  charts,
	}, nil
}
