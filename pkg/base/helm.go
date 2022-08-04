package base

import (
	"bytes"
	"fmt"
	"io/ioutil"
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

type HelmSubCharts struct {
	ParentName string
	SubCharts  []string
}

type HelmName struct {
	Name string `yaml:"name"`
}
type HelmChartDependency struct {
	Alias string `yaml:"alias"`
	Name  string `yaml:"name"`
}
type HelmDependencies struct {
	Dependencies []HelmChartDependency `yaml:"dependencies"`
}

type HelmNameAndDependencies struct {
	HelmName
	HelmDependencies
}

type subChartBase struct {
	Name      string
	BaseFiles []BaseFile
}

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

	// This will be added back later by renderReplicated
	// I do not want to change the functionality of kots installing a helm chart
	base.Path = ""

	baseAliasMap := getBaseAliasMap(u)

	upstreamFiles := make(map[string][]byte)
	for _, upstreamFile := range u.Files {
		upstreamFiles[upstreamFile.Path] = upstreamFile.Content
	}

	upstreamAliasMap := getUpstreamAliasMap(u)

	nextBase := helmChartBaseAppendAdditionalFilesv2(*base, base.Path, upstreamFiles, baseAliasMap)
	nexterBase := helmChartBaseAppendMissingDependenciesv2(nextBase, upstreamFiles, upstreamAliasMap)

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

// level -> alias -> name
func getBaseAliasMap(u *upstreamtypes.Upstream) map[int]map[string]string {
	// map representing the chart aliases at each level in the dependency tree
	aliasMap := make(map[int]map[string]string)
	for _, upstreamFile := range u.Files {
		if strings.HasSuffix(upstreamFile.Path, "Chart.yaml") {
			level := strings.Count(upstreamFile.Path, "charts/") + 1
			chart := &HelmNameAndDependencies{}
			err := yaml.Unmarshal(upstreamFile.Content, chart)
			if err != nil {
				logger.Errorf("failed to unmarshal upstream file %s: %s", upstreamFile.Path, err)
				continue
			}
			for _, dep := range chart.Dependencies {
				if dep.Alias != "" {
					if _, ok := aliasMap[level]; !ok {
						aliasMap[level] = map[string]string{}
					}
					aliasMap[level][dep.Alias] = dep.Name
				}
			}
		}
	}

	return aliasMap
}

// level -> name -> []alias
func getUpstreamAliasMap(u *upstreamtypes.Upstream) map[int]map[string][]string {
	// map representing the chart aliases at each level in the dependency tree
	aliasMap := make(map[int]map[string][]string)
	for _, upstreamFile := range u.Files {
		if strings.HasSuffix(upstreamFile.Path, "Chart.yaml") {
			level := strings.Count(upstreamFile.Path, "charts"+string(os.PathSeparator)) + 1
			chart := &HelmNameAndDependencies{}
			err := yaml.Unmarshal(upstreamFile.Content, chart)
			if err != nil {
				logger.Errorf("failed to unmarshal upstream file %s: %s", upstreamFile.Path, err)
				continue
			}
			for _, dep := range chart.Dependencies {
				if dep.Alias != "" {
					if _, ok := aliasMap[level]; !ok {
						aliasMap[level] = map[string][]string{}
					}
					aliasMap[level][dep.Name] = append(aliasMap[level][dep.Name], dep.Alias)
				}
			}
		}
	}

	return aliasMap
}

func helmChartBaseAppendAdditionalFilesv2(parentBase Base, fullBasePath string, upstreamFiles map[string][]byte, aliasMap map[int]map[string]string) Base {
	re := regexp.MustCompile(`\/?charts\/`)
	levels := re.Split(fullBasePath, -1)
	for i, level := range levels {
		if _, ok := aliasMap[i]; ok {
			name := aliasMap[i][level]
			if name != "" {
				levels[i] = name
			}
		}
	}
	upstreamPath := strings.Join(levels, string(os.PathSeparator)+"charts"+string(os.PathSeparator))
	upstreamPath = strings.TrimPrefix(upstreamPath, string(os.PathSeparator))

	additionalFiles := []string{"Chart.yaml", "Chart.lock"}
	for _, additionalFile := range additionalFiles {
		for upstreamFilePath, upstreamFileContent := range upstreamFiles {
			if upstreamPath == "" && upstreamFilePath == path.Join(upstreamPath, additionalFile) {
				parentBase.AdditionalFiles = append(parentBase.AdditionalFiles, BaseFile{
					Path:    additionalFile,
					Content: upstreamFileContent,
				})
			} else if upstreamPath != "" && strings.HasSuffix(upstreamFilePath, path.Join(upstreamPath, additionalFile)) {
				parentBase.AdditionalFiles = append(parentBase.AdditionalFiles, BaseFile{
					Path:    additionalFile,
					Content: upstreamFileContent,
				})
			}
		}
		// // TODO: The below logic would be a better way to do this, but our tests have a bug in the `subchart-alias` test
		// // If we want to fix the bug, the above logic that determines the `upstreamPath` needs to be modified to account for file repositories
		// if content, ok := upstreamFiles[path.Join(upstreamPath, additionalFile)]; ok {
		// 	fmt.Printf("adding additional upstream file %s to base path %s\n", path.Join(upstreamPath, additionalFile), fullBasePath)
		// 	parentBase.AdditionalFiles = append(parentBase.AdditionalFiles, BaseFile{
		// 		Path:    additionalFile,
		// 		Content: content,
		// 	})
		// }
	}

	var nextBases []Base
	for _, base := range parentBase.Bases {
		base = helmChartBaseAppendAdditionalFilesv2(base, path.Join(parentBase.Path, base.Path), upstreamFiles, aliasMap)
		nextBases = append(nextBases, base)
	}
	parentBase.Bases = nextBases

	return parentBase
}

func helmChartBaseAppendMissingDependenciesv2(base Base, upstreamFiles map[string][]byte, aliasMap map[int]map[string][]string) Base {
	additionalBaseFiles := getAllAdditionalBaseFiles(base.Path, base)

	// iterate over the upstream files and add any missing dependencies to the base files
	for upstreamFilePath, upstreamFileContent := range upstreamFiles {
		if strings.HasSuffix(upstreamFilePath, "Chart.yaml") {
			upstreamPath := strings.TrimSuffix(upstreamFilePath, "Chart.yaml")
			upstreamPath = strings.TrimSuffix(upstreamPath, string(os.PathSeparator))

			basePaths := []string{}
			if upstreamPath == "" || strings.HasPrefix(upstreamPath, "charts") {
				re := regexp.MustCompile(`\/?charts\/`)
				levels := re.Split(upstreamPath, -1)
				basePathParts := make([][]string, len(levels))
				for i, level := range levels {
					foundAliases := false
					if _, ok := aliasMap[i]; ok {
						aliases := aliasMap[i][level]
						if len(aliases) > 0 {
							basePathParts[i] = aliases
							foundAliases = true
						}
					}
					if !foundAliases {
						basePathParts[i] = []string{level}
					}
				}
				basePaths = flattenBasePathParts(basePathParts)
			} else {
				// if the upstream path is not empty and doesn't start with "charts", then it could be a file repository
				// in this case, we just use the upstream path as the base path
				basePaths = []string{upstreamPath}
			}

			for _, basePath := range basePaths {
				if _, ok := additionalBaseFiles[path.Join(basePath, "Chart.yaml")]; !ok {
					logger.Infof("adding missing dependency %s to base path %s\n", upstreamFilePath, basePath)
					b := Base{
						Path: basePath,
						AdditionalFiles: []BaseFile{
							{
								Path:    "Chart.yaml",
								Content: upstreamFileContent,
							},
						},
					}
					base.Bases = append(base.Bases, b)
				}
			}
		}
	}

	return base
}

func flattenBasePathParts(basePathParts [][]string) []string {
	numParts := len(basePathParts)
	currentIndices := make([]int, numParts)
	paths := []string{}
	for true {
		parts := []string{}
		for i := 0; i < numParts; i++ {
			part := basePathParts[i][currentIndices[i]]
			parts = append(parts, part)
		}

		path := strings.Join(parts, string(os.PathSeparator)+"charts"+string(os.PathSeparator))
		path = strings.TrimPrefix(path, string(os.PathSeparator))
		paths = append(paths, path)

		next := numParts - 1
		for next >= 0 && currentIndices[next] == len(basePathParts[next])-1 {
			next--
		}

		if next < 0 {
			break
		}

		currentIndices[next]++

		for i := next + 1; i < numParts; i++ {
			currentIndices[i] = 0
		}
	}
	return paths
}

func getAllAdditionalBaseFiles(parentPath string, base Base) map[string][]byte {
	baseFiles := map[string][]byte{}
	for _, additionalFile := range base.AdditionalFiles {
		baseFiles[path.Join(parentPath, base.Path, additionalFile.Path)] = additionalFile.Content
	}
	for _, b := range base.Bases {
		subBaseFiles := getAllAdditionalBaseFiles(base.Path, b)
		for k, v := range subBaseFiles {
			baseFiles[k] = v
		}
	}
	return baseFiles
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

// Returns a list of HelmSubCharts, each of which contains the name of the parent chart and a list of subcharts
// Each item in the subcharts list is a string of repeating terms the form "charts/<chart name>".
// The first item is just the top level chart (TODO: this should be removed)
// For example:
//   - top-level-chart
//   - charts/top-level-chart
//   - charts/top-level-chart/charts/cool-sub-chart
func FindHelmSubChartsFromBase(baseDir, parentChartName string) (*HelmSubCharts, error) {
	charts := make([]string, 0)
	searchDir := filepath.Join(baseDir, "charts", parentChartName)

	// If dependencies in the chart are aliased, they will create new directories with the alias name
	// in the charts folder and need to be excluded when generating the pullsecrets.yaml. It feels like this
	// could replace the logic below that's doing the file tree walking but I'm unsure.
	parentChartPath := filepath.Join(searchDir, "Chart.yaml")
	parentChartRaw, err := ioutil.ReadFile(parentChartPath)
	if err == nil {
		parentChart := new(HelmDependencies)
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

			contents, err := ioutil.ReadFile(path)
			if err != nil {
				return errors.Wrap(err, "failed to read file")
			}

			// unmarshal just the name of the chart
			var chartInfo HelmName
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
