package base

import (
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
				if err := os.MkdirAll(d, 0744); err != nil {
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

	var rendered map[string]string
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

	base, err := writeHelmBase(u.Name, rendered, renderOptions)
	if err != nil {
		return nil, errors.Wrapf(err, "write helm chart %s base", u.Name)
	}

	base.Path = "" // this will be added back later by renderReplicated
	return base, nil
}

func writeHelmBase(chartName string, fileMap map[string]string, renderOptions *RenderOptions) (*Base, error) {
	rest, crds, subCharts := splitHelmFiles(removeCommonPrefix(fileMap))

	base := &Base{
		Path: path.Join("charts", chartName),
	}
	for k, v := range rest {
		fileBaseFiles, err := writeHelmBaseFile(k, v, renderOptions)
		if err != nil {
			return nil, errors.Wrapf(err, "write helm base file %s", k)
		}
		base.Files = append(base.Files, fileBaseFiles...)
	}

	if len(crds) > 0 {
		crdsBase := Base{
			Path: "crds",
		}
		for k, v := range crds {
			fileBaseFiles, err := writeHelmBaseFile(k, v, renderOptions)
			if err != nil {
				return nil, errors.Wrapf(err, "write crds helm base file %s", k)
			}
			crdsBase.Files = append(crdsBase.Files, fileBaseFiles...)
		}
		base.Bases = append(base.Bases, crdsBase)
	}

	for subChartName, subChart := range subCharts {
		subChartBase, err := writeHelmBase(subChartName, subChart, renderOptions)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to write helm sub chart %s base", subChartName)
		}
		base.Bases = append(base.Bases, *subChartBase)
	}

	return base, nil
}

func splitHelmFiles(files map[string]string) (rest map[string]string, crds map[string]string, subCharts map[string]map[string]string) {
	subCharts = map[string]map[string]string{}
	crds = map[string]string{}
	rest = map[string]string{}
	for k, v := range files {
		dirPrefix := strings.SplitN(k, string(os.PathSeparator), 3)
		if dirPrefix[0] == "charts" && len(dirPrefix) == 3 {
			subChartName := dirPrefix[1]
			if subCharts[subChartName] == nil {
				subCharts[subChartName] = map[string]string{}
			}
			k = path.Join(dirPrefix[2:]...)
			subCharts[subChartName][k] = v
		} else if dirPrefix[0] == "crds" {
			k = path.Join(dirPrefix[1:]...)
			crds[k] = v
		} else {
			rest[k] = v
		}
	}
	return
}

func writeHelmBaseFile(name, content string, renderOptions *RenderOptions) ([]BaseFile, error) {
	fileStrings := []string{}
	if renderOptions.SplitMultiDocYAML {
		fileStrings = strings.Split(content, "\n---\n")
	} else {
		fileStrings = append(fileStrings, content)
	}

	baseFiles := []BaseFile{}

	for idx, fileString := range fileStrings {
		filename := name
		if len(fileStrings) > 1 {
			filename = strings.TrimSuffix(name, filepath.Ext(name))
			filename = fmt.Sprintf("%s-%d%s", filename, idx+1, filepath.Ext(name))
		}

		baseFile := BaseFile{
			Path:    filename,
			Content: []byte(fileString),
		}
		if err := baseFile.transpileHelmHooksToKotsHooks(); err != nil {
			return nil, errors.Wrap(err, "failed to transpile helm hooks to kots hooks")
		}

		baseFiles = append(baseFiles, baseFile)
	}

	return baseFiles, nil
}

// removeCommonPrefix will remove any common prefix from all files
func removeCommonPrefix(fileMap map[string]string) map[string]string {
	if len(fileMap) == 0 {
		return fileMap
	}

	commonPrefix := []string{}

	first := true
	for filepath := range fileMap {
		if first {
			firstFileDir, _ := path.Split(filepath)
			commonPrefix = strings.Split(firstFileDir, string(os.PathSeparator))

			first = false
			continue
		}
		d, _ := path.Split(filepath)
		dirs := strings.Split(d, string(os.PathSeparator))

		commonPrefix = util.CommonSlicePrefix(commonPrefix, dirs)
	}

	cleanedFileMap := map[string]string{}
	for filepath, content := range fileMap {
		d, f := path.Split(filepath)
		d2 := strings.Split(d, string(os.PathSeparator))

		d2 = d2[len(commonPrefix):]
		filepath = path.Join(path.Join(d2...), f)

		cleanedFileMap[filepath] = content
	}

	return cleanedFileMap
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
