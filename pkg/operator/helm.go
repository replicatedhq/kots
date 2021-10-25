package operator

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
)

var (
	goTemplateRegex *regexp.Regexp
)

func init() {
	goTemplateRegex = regexp.MustCompile(`({{)|(}})`)
}

// When saving templated file back into a chart, we need to escape Go templates so second Helm pass would ignore them.
// These are appication templates that maybe used in application config files and Helm should ignore them.
// For example, origianl chart has this:
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

func renderChartsArchive(deployedVersionArchive string, name string, kustomizeBinPath string) ([]byte, error) {
	archiveChartDir := filepath.Join(deployedVersionArchive, "overlays", "downstreams", name, "charts")
	_, err := os.Stat(archiveChartDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	exportChartPath, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(exportChartPath)

	desrChartsDir := filepath.Join(exportChartPath, "charts")
	if _, err := os.Stat(desrChartsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(desrChartsDir, 0755); err != nil {
			return nil, errors.Wrap(err, "failed to mkdir for archive chart")
		}
	}

	sourceChartsDir := filepath.Join(deployedVersionArchive, "base", "charts")
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
				err = copyHelmMetadataFile(sourceChartsDir, desrChartsDir, relPath, filename)
				if err != nil {
					return errors.Wrapf(err, "failed to export file %s", filename)
				}
			}

			if info.Name() == "kustomization.yaml" {
				archiveChartOutput, err := exec.Command(kustomizeBinPath, "build", filepath.Dir(path)).Output()
				if err != nil {
					if ee, ok := err.(*exec.ExitError); ok {
						err = fmt.Errorf("kustomize %s: %q", path, string(ee.Stderr))
					}
					return errors.Wrapf(err, "failed to kustomize %s", path)
				}
				err = saveHelmFile(desrChartsDir, relPath, "all.yaml", archiveChartOutput)
				if err != nil {
					return errors.Wrapf(err, "failed to export content for %s", path)
				}
			}
			return nil
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to walk charts directory")
	}

	tempDir, err := ioutil.TempDir("", "helmkots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tempDir)
	tarGz := archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: true,
		},
	}
	if err := tarGz.Archive([]string{desrChartsDir}, path.Join(tempDir, "helmcharts.tar.gz")); err != nil {
		return nil, errors.Wrap(err, "failed to create tar gz")
	}

	archive, err := ioutil.ReadFile(path.Join(tempDir, "helmcharts.tar.gz"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to read helm tar.gz file")
	}

	return archive, nil
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
