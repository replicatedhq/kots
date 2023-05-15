package apparchive

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
)

var (
	goTemplateRegex *regexp.Regexp
)

func init() {
	goTemplateRegex = regexp.MustCompile(`({{)|(}})`)
}

func GetRenderedV1Beta1ChartsArchive(versionArchive string, downstreamName, kustomizeBinPath string) ([]byte, map[string][]byte, error) {
	renderedChartsDir := filepath.Join(versionArchive, "rendered", downstreamName, "charts")
	if _, err := os.Stat(renderedChartsDir); err == nil {
		// charts are already rendered, so we can just tar.gz them up
		filesMap, err := util.GetFilesMap(renderedChartsDir)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to get files map")
		}

		archive, err := util.TGZArchive(renderedChartsDir)
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to create tar.gz")
		}

		return archive, filesMap, nil
	}

	// older kots versions did not include the rendered charts in the app archive, so we have to render them
	baseDir := filepath.Join(versionArchive, "base")
	overlaysDir := filepath.Join(versionArchive, "overlays")

	chartsDir := filepath.Join(baseDir, "charts")
	if err := cleanBaseApp(chartsDir, nil); err != nil {
		return nil, nil, errors.Wrap(err, "failed to clean base app")
	}

	archive, filesMap, err := RenderChartsArchive(baseDir, overlaysDir, downstreamName, kustomizeBinPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to render charts archive")
	}

	return archive, filesMap, nil
}

func RenderChartsArchive(baseDir string, overlaysDir string, downstreamName string, kustomizeBinPath string) ([]byte, map[string][]byte, error) {
	archiveChartsDir := filepath.Join(overlaysDir, "downstreams", downstreamName, "charts")
	_, err := os.Stat(archiveChartsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, errors.Wrap(err, "failed to stat charts directory")
	}

	exportChartPath, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(exportChartPath)

	destChartsDir := filepath.Join(exportChartPath, "charts")
	if _, err := os.Stat(destChartsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(destChartsDir, 0755); err != nil {
			return nil, nil, errors.Wrap(err, "failed to mkdir for archive chart")
		}
	}

	renderedFilesMap := map[string][]byte{}
	sourceChartsDir := filepath.Join(baseDir, "charts")
	metadataFiles := []string{"Chart.yaml", "Chart.lock"}

	err = filepath.Walk(archiveChartsDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			relPath, err := filepath.Rel(archiveChartsDir, filepath.Dir(path))
			if err != nil {
				return errors.Wrapf(err, "failed to get %s relative path to %s", path, archiveChartsDir)
			}

			for _, filename := range metadataFiles {
				content, err := ioutil.ReadFile(filepath.Join(sourceChartsDir, relPath, filename))
				if err != nil {
					if os.IsNotExist(err) {
						continue
					}
					return errors.Wrapf(err, "failed to read file %s", filename)
				}
				if err := writeHelmFile(destChartsDir, relPath, filename, content, renderedFilesMap); err != nil {
					return errors.Wrapf(err, "failed to export file %s", filename)
				}
			}

			if info.Name() != "kustomization.yaml" {
				return nil
			}

			srcPath := filepath.Join(sourceChartsDir, relPath)
			_, err = os.Stat(srcPath)
			if err != nil && !os.IsNotExist(err) {
				return errors.Wrapf(err, "failed to os stat file %s", srcPath)
			}
			if os.IsNotExist(err) {
				return nil // source chart does not exist in base
			}

			archiveChartOutput, err := exec.Command(kustomizeBinPath, "build", filepath.Dir(path)).Output()
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					err = fmt.Errorf("kustomize %s: %q", path, string(ee.Stderr))
				}
				return errors.Wrapf(err, "failed to kustomize %s", path)
			}

			archiveFiles, err := util.SplitYAML(archiveChartOutput)
			if err != nil {
				return errors.Wrapf(err, "failed to split yaml result for %s", path)
			}

			for filename, content := range archiveFiles {
				destRelPath := relPath

				if filepath.Base(destRelPath) != "crds" {
					destRelPath = filepath.Join(destRelPath, "templates")
					content = escapeGoTemplates(content)
				}

				if err := writeHelmFile(destChartsDir, destRelPath, filename, content, renderedFilesMap); err != nil {
					return errors.Wrapf(err, "failed to write file %s", filename)
				}
			}

			return nil
		})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to walk charts directory")
	}

	archive, err := util.TGZArchive(destChartsDir)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create tar.gz")
	}

	return archive, renderedFilesMap, nil
}

func writeHelmFile(dstRootDir string, relPath string, filename string, content []byte, renderedFilesMap map[string][]byte) error {
	dstDir := filepath.Join(dstRootDir, relPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create destination dir")
	}

	destFilePath := filepath.Join(dstDir, filename)
	if err := ioutil.WriteFile(destFilePath, content, 0644); err != nil {
		return errors.Wrap(err, "failed to write file")
	}

	renderedFilesMap[filepath.Join(relPath, filename)] = content

	return nil
}

// When saving templated file back into a chart, we need to escape Go templates so second Helm pass would ignore them.
// These are application templates that maybe used in application config files and Helm should ignore them.
// For example, original chart has this:
//
//	"legendFormat": "{{`{{`}} value {{`}}`}}",
//
// Rendered chart becomes:
//
//	"legendFormat": "{{ value }}",
//
// Repackaged chart should have this:
//
//	"legendFormat": "{{`{{`}} value {{`}}`}}",
func escapeGoTemplates(content []byte) []byte {
	replace := func(in []byte) []byte {
		if string(in) == "{{" {
			return []byte("{{`{{`}}")
		}
		if string(in) == "}}" {
			return []byte("{{`}}`}}")
		}
		return in
	}

	return goTemplateRegex.ReplaceAllFunc(content, replace)
}
