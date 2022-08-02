package kustomize

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"

	"github.com/marccampbell/yaml-toolbox/pkg/splitter"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
)

var (
	goTemplateRegex *regexp.Regexp
)

func init() {
	goTemplateRegex = regexp.MustCompile(`({{)|(}})`)
}

func RenderChartsArchive(versionArchive string, downstreamName string, kustomizeBinPath string) ([]byte, map[string][]byte, error) {
	archiveChartDir := filepath.Join(versionArchive, "overlays", "downstreams", downstreamName, "charts")
	_, err := os.Stat(archiveChartDir)
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

	kustomizedFilesList := map[string][]byte{}
	sourceChartsDir := filepath.Join(versionArchive, "base", "charts")
	metadataFiles := []string{"Chart.yaml", "Chart.lock"}

	err = filepath.Walk(archiveChartDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			relPath, err := filepath.Rel(archiveChartDir, filepath.Dir(path))
			if err != nil {
				return errors.Wrapf(err, "failed to get %s relative path to %s", path, archiveChartDir)
			}

			for _, filename := range metadataFiles {
				err = copyHelmMetadataFile(sourceChartsDir, destChartsDir, relPath, filename)
				if err != nil {
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

			archiveFiles, err := splitter.SplitYAML(archiveChartOutput)
			if err != nil {
				return errors.Wrapf(err, "failed to split yaml result for %s", path)
			}
			for filename, d := range archiveFiles {
				kustomizedFilesList[filename] = d
			}

			err = saveHelmFile(destChartsDir, relPath, "all.yaml", archiveChartOutput)
			if err != nil {
				return errors.Wrapf(err, "failed to export content for %s", path)
			}
			return nil
		})
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to walk charts directory")
	}

	tempDir, err := ioutil.TempDir("", "helmkots")
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tempDir)
	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: true,
		},
	}
	if err := tarGz.Archive([]string{destChartsDir}, path.Join(tempDir, "helmcharts.tar.gz")); err != nil {
		return nil, nil, errors.Wrap(err, "failed to create tar gz")
	}

	archive, err := ioutil.ReadFile(path.Join(tempDir, "helmcharts.tar.gz"))
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read helm tar.gz file")
	}

	return archive, kustomizedFilesList, nil
}

func saveHelmFile(rootDir string, relDir string, filename string, content []byte) error {
	// We only get CRDs and templates YAML after kustomization
	destDir := filepath.Join(rootDir, relDir)
	if filepath.Base(relDir) != "crds" {
		destDir = filepath.Join(destDir, "templates")
		content = escapeGoTemplates(content)
	}

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return errors.Wrapf(err, "failed to mkdir for export chart %s", destDir)
	}

	exportFile := filepath.Join(destDir, filename)
	err := ioutil.WriteFile(exportFile, content, 0644)
	if err != nil {
		return errors.Wrapf(err, "failed to write file %s", exportFile)
	}

	return nil
}

func copyHelmMetadataFile(srcRootDir string, dstRootDir string, relPath string, filename string) error {
	fileContent, err := ioutil.ReadFile(filepath.Join(srcRootDir, relPath, filename))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrap(err, "failed to read file")
	}

	dstDir := filepath.Join(dstRootDir, relPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return errors.Wrap(err, "failed to create destination dir")
	}

	dstFilename := filepath.Join(dstDir, filename)
	err = ioutil.WriteFile(dstFilename, fileContent, 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write file")
	}

	return nil
}

// When saving templated file back into a chart, we need to escape Go templates so second Helm pass would ignore them.
// These are application templates that maybe used in application config files and Helm should ignore them.
// For example, original chart has this:
//		"legendFormat": "{{`{{`}} value {{`}}`}}",
// Rendered chart becomes:
//		"legendFormat": "{{ value }}",
// Repackaged chart should have this:
//		"legendFormat": "{{`{{`}} value {{`}}`}}",
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
