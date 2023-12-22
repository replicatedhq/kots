package apparchive

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/util"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

type AppWriteOptions struct {
	BaseDir          string
	OverlaysDir      string
	RenderedDir      string
	Downstreams      []string
	KustomizeBinPath string
}

func WriteRenderedApp(options AppWriteOptions) error {
	// cleanup existing rendered content if any
	_, err := os.Stat(options.RenderedDir)
	if err == nil {
		if err := os.RemoveAll(options.RenderedDir); err != nil {
			return errors.Wrap(err, "failed to remove existing rendered content")
		}
	}

	for _, downstreamName := range options.Downstreams {
		kustomizeBuildTarget := filepath.Join(options.OverlaysDir, "downstreams", downstreamName)

		baseKustomization, err := k8sutil.ReadKustomizationFromFile(filepath.Join(options.BaseDir, "kustomization.yaml"))
		if err != nil {
			return errors.Wrap(err, "failed to read base kustomization")
		}

		if err := baseKustomization.CheckEmpty(); err != nil {
			baseKustomization.MetaData = &kustomizetypes.ObjectMeta{
				Annotations: map[string]string{
					"kots.io/kustomization": "base",
				},
			}
			if err := k8sutil.WriteKustomizationToFile(*baseKustomization, filepath.Join(options.BaseDir, "kustomization.yaml")); err != nil {
				return errors.Wrap(err, "failed to write base kustomization")
			}
		}

		renderedApp, err := exec.Command(options.KustomizeBinPath, "build", kustomizeBuildTarget).Output()
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				err = fmt.Errorf("kustomize stderr: %q", string(ee.Stderr))
			}
			return errors.Wrap(err, "failed to run kustomize build")
		}

		renderedAppFiles, err := util.SplitYAML(renderedApp)
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

				// the charts and helm directories include v1beta1 and v1beta2 helm charts, respectively,
				// to be installed using the helm cli and are processed separately.
				if strings.Split(relPath, string(os.PathSeparator))[0] == "charts" {
					return nil
				}
				if strings.Split(relPath, string(os.PathSeparator))[0] == "helm" {
					return nil
				}

				content, err := os.ReadFile(path)
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

	baseDir := filepath.Join(versionArchive, "base")
	filter := filterChartsInBasePath(baseDir)
	if err := cleanBaseApp(baseDir, filter); err != nil {
		return nil, nil, errors.Wrap(err, "failed to clean base app")
	}

	// older kots versions did not include the rendered app in the archive, so we have to render it
	kustomizeBuildTarget := filepath.Join(versionArchive, "overlays", "downstreams", downstreamName)

	kustomization, err := k8sutil.ReadKustomizationFromFile(filepath.Join(baseDir, "kustomization.yaml"))
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to read base kustomization")
	}

	if err := kustomization.CheckEmpty(); err != nil {
		kustomization.MetaData = &kustomizetypes.ObjectMeta{
			Annotations: map[string]string{
				"kots.io/kustomization": "base",
			},
		}
		if err := k8sutil.WriteKustomizationToFile(*kustomization, filepath.Join(baseDir, "kustomization.yaml")); err != nil {
			return nil, nil, errors.Wrap(err, "failed to write base kustomization")
		}
	}

	allContent, err := exec.Command(kustomizeBinPath, "build", kustomizeBuildTarget).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, nil, fmt.Errorf("kustomize stderr: %q", string(ee.Stderr))
		}
		return nil, nil, errors.Wrap(err, "failed to run kustomize build")
	}

	filesMap, err := util.SplitYAML(allContent)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to split yaml")
	}

	return allContent, filesMap, nil
}

// cleanBaseApp iterates over the base files and removes any files with nil map entries
// this does not include helm charts, which are processed separately.
// an optional filter can be passed to skip files that should not be removed.
//
// workaround for: https://github.com/kubernetes-sigs/kustomize/issues/5050
func cleanBaseApp(baseDir string, filter func(path string) (bool, error)) error {
	if _, err := os.Stat(baseDir); err == nil {
		err := filepath.Walk(baseDir,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				if filter != nil {
					shouldFilter, err := filter(path)
					if err != nil {
						return errors.Wrapf(err, "failed to filter path %s", path)
					}
					if shouldFilter {
						return nil
					}
				}

				content, err := os.ReadFile(path)
				if err != nil {
					return errors.Wrapf(err, "failed to read file %s", path)
				}

				_, manifest := base.GetGVKWithNameAndNs(content, "")
				if manifest.APIVersion == "" || manifest.Kind == "" || manifest.Metadata.Name == "" {
					// ignore invalid resources
					return nil
				}

				if manifest.Kind == "CustomResourceDefinition" {
					// ignore crds
					return nil
				}

				newContent, err := kotsutil.RemoveNilFieldsFromYAML(content)
				if err != nil {
					return errors.Wrapf(err, "failed to remove empty mapping fields from %s", path)
				}

				if err := os.WriteFile(path, newContent, 0644); err != nil {
					return errors.Wrapf(err, "failed to write file %s", path)
				}

				return nil
			})
		if err != nil {
			return errors.Wrap(err, "failed to walk dir")
		}
	}

	return nil
}

// filterChartsInBasePath returns a filter that can be passed to cleanBaseApp
// to skip files in the charts directory
func filterChartsInBasePath(basePath string) func(path string) (bool, error) {
	return func(path string) (bool, error) {
		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return false, errors.Wrapf(err, "failed to get relative path for %s", path)
		}

		if strings.Split(relPath, string(os.PathSeparator))[0] == "charts" {
			return true, nil
		}

		return false, nil
	}
}
