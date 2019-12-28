package base

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kustomizetypes "sigs.k8s.io/kustomize/v3/pkg/types"
)

type WriteOptions struct {
	BaseDir          string
	Overwrite        bool
	ExcludeKotsKinds bool
}

func (b *Base) WriteBase(options WriteOptions) error {
	renderDir := options.BaseDir

	_, err := os.Stat(renderDir)
	if err == nil {
		if options.Overwrite {
			if err := os.RemoveAll(renderDir); err != nil {
				return errors.Wrap(err, "failed to remove previous content in base")
			}
		} else {
			return fmt.Errorf("directory %s already exists", renderDir)
		}
	}

	if _, err := os.Stat(renderDir); os.IsNotExist(err) {
		if err := os.MkdirAll(renderDir, 0744); err != nil {
			return errors.Wrap(err, "failed to mkdir for base root")
		}
	}

	foundGVKNames := [][]byte{}
	kustomizeResources := []string{}
	kustomizePatches := []kustomizetypes.PatchStrategicMerge{}
	for _, file := range b.Files {
		writeToBase := file.ShouldBeIncludedInBaseFilesystem(options.ExcludeKotsKinds)
		writeToKustomization := file.ShouldBeIncludedInBaseKustomization(options.ExcludeKotsKinds)

		if !writeToBase && !writeToKustomization {
			continue
		}

		if writeToKustomization {
			found := false
			thisGVKName := GetGVKWithNameHash(file.Content)
			for _, gvkName := range foundGVKNames {
				if bytes.Compare(gvkName, thisGVKName) == 0 {
					found = true
				}
			}

			if !found || thisGVKName == nil {
				kustomizeResources = append(kustomizeResources, path.Join(".", file.Path))
				foundGVKNames = append(foundGVKNames, thisGVKName)
			} else {
				kustomizePatches = append(kustomizePatches, kustomizetypes.PatchStrategicMerge(path.Join(".", file.Path)))
			}
		}

		if writeToBase {
			fileRenderPath := path.Join(renderDir, file.Path)
			d, _ := path.Split(fileRenderPath)
			if _, err := os.Stat(d); os.IsNotExist(err) {
				if err := os.MkdirAll(d, 0744); err != nil {
					return errors.Wrap(err, "failed to mkdir")
				}
			}

			if err := ioutil.WriteFile(fileRenderPath, file.Content, 0644); err != nil {
				return errors.Wrap(err, "failed to write base file")
			}
		}
	}

	kustomization := kustomizetypes.Kustomization{
		TypeMeta: kustomizetypes.TypeMeta{
			APIVersion: "kustomize.config.k8s.io/v1beta1",
			Kind:       "Kustomization",
		},
		Resources:             kustomizeResources,
		PatchesStrategicMerge: kustomizePatches,
	}

	if err := k8sutil.WriteKustomizationToFile(&kustomization, path.Join(renderDir, "kustomization.yaml")); err != nil {
		return errors.Wrap(err, "failed to write kustomization to file")
	}

	return nil
}

func (b *Base) GetOverlaysDir(options WriteOptions) string {
	renderDir := options.BaseDir

	return path.Join(renderDir, "..", "overlays")
}
