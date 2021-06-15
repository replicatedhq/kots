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

	rendered = removeCommonPrefix(rendered)
	base, err := writeHelmBase(u.Name, rendered, renderOptions)
	if err != nil {
		return nil, errors.Wrapf(err, "write helm chart %s base", u.Name)
	}

	base.Path = "" // this will be added back later by renderReplicated
	return base, nil
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

	for subChartName, subChart := range subCharts {
		subChartBase, err := writeHelmBase(subChartName, subChart, renderOptions)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to write helm sub chart %s base", subChartName)
		}
		base.Bases = append(base.Bases, *subChartBase)
	}

	return base, nil
}

func splitHelmFiles(baseFiles []BaseFile) (rest []BaseFile, crds []BaseFile, subCharts map[string][]BaseFile) {
	subCharts = map[string][]BaseFile{}
	for _, baseFile := range baseFiles {
		dirPrefix := strings.SplitN(baseFile.Path, string(os.PathSeparator), 3)
		if dirPrefix[0] == "charts" && len(dirPrefix) == 3 {
			subChartName := dirPrefix[1]
			subCharts[subChartName] = append(subCharts[subChartName], BaseFile{
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
