package kustomize

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	logs "log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sync"
	"time"

	"github.com/marccampbell/yaml-toolbox/pkg/splitter"
	"github.com/mholt/archiver/v3"
	"github.com/pkg/errors"
)

// struct with mutex and kustomizedFilesList map
type kustomizedFiles struct {
	mu   sync.Mutex
	list map[string]string
}

var (
	goTemplateRegex *regexp.Regexp
)

func init() {
	goTemplateRegex = regexp.MustCompile(`({{)|(}})`)
}

func RenderChartsArchive(versionArchive string, downstreamName string, kustomizeBinPath string) ([]byte, map[string]string, error) {
	archiveChartDir := filepath.Join(versionArchive, "overlays", "downstreams", downstreamName, "charts")
	logs.Printf("LG: archive directory: %v", archiveChartDir)
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

	kustomizedFilesList := map[string]string{}
	sourceChartsDir := filepath.Join(versionArchive, "base", "charts")
	metadataFiles := []string{"Chart.yaml", "Chart.lock"}

	filewalkStart := time.Now()
	fileCount := 0
	err = filepath.WalkDir(archiveChartDir,
		func(path string, info fs.DirEntry, err error) error {
			fileCount++
			return nil
		})
	filewalkDuration := time.Since(filewalkStart)
	logs.Printf("LG: Filepath walk only duration: %v File count: %v", filewalkDuration, fileCount)

	processArchiveStart := time.Now()
	totalPathsProcessed := 0
	wg := new(sync.WaitGroup)
	wg.Add(1)
	// new kustomizedFiles struct
	kustomizedFiles := &kustomizedFiles{
		list: make(map[string]string),
	}
	err = processArchive(archiveChartDir, sourceChartsDir, destChartsDir, kustomizeBinPath, kustomizedFiles, metadataFiles, wg, &totalPathsProcessed)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to process archive")
	}
	wg.Wait()
	processArchiveDuration := time.Since(processArchiveStart)
	logs.Printf("LG: Process archive duration: %v, totalPathsProcessed: %v", processArchiveDuration, totalPathsProcessed)
	logs.Printf("LG: Length of kustomize List: %v", len(kustomizedFiles.list))

	var totalDuration time.Duration
	totalPaths := 0
	startWalk := time.Now()
	// durations := []time.Duration{}
	err = filepath.WalkDir(archiveChartDir,
		func(path string, info fs.DirEntry, err error) error {
			// start := time.Now()
			totalPaths++
			if err != nil {
				return err
			}
			relPath, err := filepath.Rel(archiveChartDir, filepath.Dir(path))
			if err != nil {
				return errors.Wrapf(err, "failed to get %s relative path to %s", path, archiveChartDir)
			}

			//sleep for 5 seconds
			// time.Sleep(5 * time.Second)

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
				kustomizedFilesList[filename] = string(d)
			}

			err = saveHelmFile(destChartsDir, relPath, "all.yaml", archiveChartOutput)
			if err != nil {
				return errors.Wrapf(err, "failed to export content for %s", path)
			}
			// thisDuration := time.Since(start)
			// durations = append(durations, thisDuration)
			totalDuration += thisDuration
			return nil
		})
	avgDuration := totalDuration / time.Duration(totalPaths)
	logs.Printf("LG: Total paths: %d, total duration: %v, Average duration per path: %v", totalPaths, totalDuration, avgDuration)
	walkDuration := time.Since(startWalk)
	logs.Printf("LG: Duration of file tree walk: %v", walkDuration)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to walk charts directory")
	}
	logs.Printf("LG: Length of kustomizeList: %v", len(kustomizedFilesList))
	// logs.Printf("LG: file tree durations: ")
	// //log durations
	// for _, d := range durations {
	// 	logs.Printf("%v", d)
	// }

	// logs.Printf("LG: ------------------")

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

func processArchive(archiveChartDir string, sourceChartsDir string, destChartsDir string, kustomizeBinPath string, kustomizedFiles *kustomizedFiles, metadataFiles []string, wg *sync.WaitGroup, totalPaths *int) error {
	defer wg.Done()

	visit := func(path string, info fs.DirEntry, err error) error {
		// start := time.Now()
		//totalPaths++
		if err != nil {
			return err
		}

		// if info.IsDir() && path != archiveChartDir {
		// 	wg.Add(1)
		// 	go processArchive(path, sourceChartsDir, destChartsDir, kustomizeBinPath, kustomizedFilesList, metadataFiles, wg, totalPaths)
		// 	return filepath.SkipDir
		// }
		wg.Add(1)
		go processFile(archiveChartDir, sourceChartsDir, destChartsDir, kustomizeBinPath, kustomizedFiles, metadataFiles, wg, totalPaths, path, info)

		return nil
	}
	err := filepath.WalkDir(archiveChartDir, visit)
	// logs.Printf("LG: totalPaths: %d", totalPaths)
	if err != nil {
		return err
	}
	return nil
}

func processFile(archiveChartDir string, sourceChartsDir string, destChartsDir string, kustomizeBinPath string, kustomizedFiles *kustomizedFiles, metadataFiles []string, wg *sync.WaitGroup, totalPaths *int, path string, info fs.DirEntry) error {
	defer wg.Done()
	*totalPaths++
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
	kustomizedFiles.mu.Lock()
	for filename, d := range archiveFiles {
		kustomizedFiles.list[filename] = string(d)
	}
	kustomizedFiles.mu.Unlock()

	err = saveHelmFile(destChartsDir, relPath, "all.yaml", archiveChartOutput)
	if err != nil {
		return errors.Wrapf(err, "failed to export content for %s", path)
	}
	// thisDuration := time.Since(start)
	// totalDuration += thisDuration
	return nil
}
