package kustomize

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/marccampbell/yaml-toolbox/pkg/splitter"
	"github.com/pkg/errors"
)

type WriteOptions struct {
	BaseDir          string
	OverlaysDir      string
	RenderedDir      string
	Downstreams      []string
	KustomizeBinPath string
}

func WriteRenderedApp(options WriteOptions) error {
	// cleanup existing rendered content if any
	_, err := os.Stat(options.RenderedDir)
	if err == nil {
		if err := os.RemoveAll(options.RenderedDir); err != nil {
			return errors.Wrap(err, "failed to remove existing rendered content")
		}
	}

	for _, downstreamName := range options.Downstreams {
		kustomizeBuildTarget := filepath.Join(options.OverlaysDir, "downstreams", downstreamName)

		renderedApp, err := exec.Command(options.KustomizeBinPath, "build", kustomizeBuildTarget).Output()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				err = fmt.Errorf("kustomize stderr: %q", string(ee.Stderr))
			}
			return errors.Wrap(err, "failed to run kustomize build")
		}

		renderedAppFiles, err := splitter.SplitYAML(renderedApp)
		if err != nil {
			return errors.Wrap(err, "failed to split yaml")
		}

		for filename, content := range renderedAppFiles {
			destPath := filepath.Join(options.RenderedDir, downstreamName, filename)
			if err := writeRenderedFile(destPath, []byte(content)); err != nil {
				return errors.Wrapf(err, "failed to write rendered app file %s", destPath)
			}
		}

		_, renderedChartsFiles, err := RenderChartsArchive(options.BaseDir, options.OverlaysDir, downstreamName, options.KustomizeBinPath)
		if err != nil {
			return errors.Wrap(err, "failed to render charts archive")
		}

		for relPath, content := range renderedChartsFiles {
			destPath := filepath.Join(options.RenderedDir, downstreamName, "charts", relPath)
			if err := writeRenderedFile(destPath, []byte(content)); err != nil {
				return errors.Wrapf(err, "failed to write rendered chart file %s", destPath)
			}
		}
	}

	return nil
}

func writeRenderedFile(destPath string, content []byte) error {
	parentDir := filepath.Dir(destPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return errors.Wrapf(err, "failed to mkdir %s", parentDir)
	}
	if err := ioutil.WriteFile(destPath, content, 0644); err != nil {
		return errors.Wrap(err, "failed to write file")
	}
	return nil
}

func GetRenderedApp(versionArchive string, downstreamName, kustomizeBinPath string) ([]byte, map[string][]byte, error) {
	// check if the app is already rendered
	renderedAppDir := filepath.Join(versionArchive, "rendered", downstreamName)
	if _, err := os.Stat(renderedAppDir); err == nil {
		allContent := [][]byte{}
		filesMap := map[string][]byte{}

		err := filepath.Walk(renderedAppDir,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				relPath, err := filepath.Rel(renderedAppDir, path)
				if err != nil {
					return errors.Wrapf(err, "failed to get relative path for %s", path)
				}

				// the charts directory includes helm charts to be installed using the helm cli,
				// and those are processed separately.
				if strings.Split(relPath, string(os.PathSeparator))[0] == "charts" {
					return nil
				}

				content, err := ioutil.ReadFile(path)
				if err != nil {
					return errors.Wrapf(err, "failed to read file %s", path)
				}

				allContent = append(allContent, content)
				filesMap[relPath] = content

				return nil
			})
		if err != nil {
			return nil, nil, errors.Wrap(err, "failed to walk dir")
		}

		return bytes.Join(allContent, []byte("\n---\n")), filesMap, nil
	}

	// older kots versions did not include the rendered app in the archive, so we have to render it
	kustomizeBuildTarget := filepath.Join(versionArchive, "overlays", "downstreams", downstreamName)

	allContent, err := exec.Command(kustomizeBinPath, "build", kustomizeBuildTarget).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, nil, fmt.Errorf("kustomize stderr: %q", string(ee.Stderr))
		}
		return nil, nil, errors.Wrap(err, "failed to run kustomize build")
	}

	filesMap, err := splitter.SplitYAML(allContent)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to split yaml")
	}

	return allContent, filesMap, nil
}
