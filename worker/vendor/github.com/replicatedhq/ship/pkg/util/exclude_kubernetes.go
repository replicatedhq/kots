package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	yaml "gopkg.in/yaml.v3"
	"sigs.k8s.io/kustomize/pkg/patch"
	"sigs.k8s.io/kustomize/pkg/resid"
	"sigs.k8s.io/kustomize/pkg/types"
)

// calls ExcludeKubernetesResource for each excluded resource. Returns after the first error.
func ExcludeKubernetesResources(fs afero.Afero, basePath string, overlaysPath string, excludedResources []string) error {
	for _, excludedResource := range excludedResources {
		excludedIDs, err := ExcludeKubernetesResource(fs, basePath, excludedResource)
		if err != nil {
			return errors.Wrapf(err, "excluding %s from %s", excludedResource, basePath)
		}
		for _, excludedID := range excludedIDs {
			err = ExcludeKubernetesPatch(fs, overlaysPath, excludedID)
			if err != nil {
				return errors.Wrapf(err, "excluding %s of %s from %s", excludedID.String(), excludedResource, basePath)
			}
		}
	}
	return nil
}

// exclude the provided kubernetes resource file from the kustomization.yaml at basePath, or from bases imported from that.
func ExcludeKubernetesResource(fs afero.Afero, basePath string, excludedResource string) ([]resid.ResId, error) {
	kustomization, err := getKustomization(fs, basePath)
	if err != nil {
		return nil, errors.Wrapf(err, "get kustomization for %s", basePath)
	}

	excludedResource = strings.TrimPrefix(excludedResource, string(filepath.Separator))

	newResources := []string{}
	var excludedResourceBytes []byte
	for _, existingResource := range kustomization.Resources {
		if existingResource != excludedResource {
			newResources = append(newResources, existingResource)
		} else {
			excludedResourceBytes, err = fs.ReadFile(filepath.Join(basePath, excludedResource))
			if err != nil {
				return nil, errors.Wrapf(err, "read to-be-excluded resource file")
			}
		}
	}

	if len(newResources) != len(kustomization.Resources) {
		// parse to-be-excluded resource file

		excludedResources, err := NewKubernetesResources(excludedResourceBytes)
		if err != nil {
			return nil, errors.Wrapf(err, "parse to-be-excluded resource file")
		}

		kustomization.Resources = newResources
		// write updated kustomization to disk - resource has been removed

		err = writeKustomization(fs, basePath, kustomization)
		if err != nil {
			return nil, errors.Wrapf(err, "persist kustomization for %s", basePath)
		}
		return ResIDs(excludedResources), nil
	}

	// check if the resource is already removed from this dir
	alreadyRemoved, err := fs.Exists(filepath.Join(basePath, excludedResource))
	if err != nil {
		return nil, errors.Wrapf(err, "check if %s exists in %s", excludedResource, basePath)
	}
	if alreadyRemoved {
		// the file to be removed exists within this base dir, and not within the kustomization yaml
		excludedResourceBytes, err = fs.ReadFile(filepath.Join(basePath, excludedResource))
		if err != nil {
			return nil, errors.Wrapf(err, "read already-excluded resource file")
		}

		excludedResources, err := NewKubernetesResources(excludedResourceBytes)
		if err != nil {
			return nil, errors.Wrapf(err, "parse already-excluded resource file")
		}

		return ResIDs(excludedResources), nil
	}

	for _, newBase := range kustomization.Bases {
		newBase = filepath.Clean(filepath.Join(basePath, newBase))
		cleanBase := strings.ReplaceAll(newBase, string(filepath.Separator), "-")

		if strings.HasPrefix(excludedResource, cleanBase) {
			updatedResource := strings.TrimPrefix(excludedResource, cleanBase)
			updatedResource = strings.TrimPrefix(updatedResource, string(filepath.Separator))

			return ExcludeKubernetesResource(fs, newBase, updatedResource)
		}
	}

	return nil, fmt.Errorf("unable to find resource %s in %s or its bases", excludedResource, basePath)
}

// for the provided base and all subbases, check each strategic merge patch and json patch
// if they match the resid provided, remove them
func ExcludeKubernetesPatch(fs afero.Afero, basePath string, excludedResource resid.ResId) error {
	kustomization, err := getKustomization(fs, basePath)
	if err != nil {
		return errors.Wrapf(err, "get kustomization for %s", basePath)
	}

	newJSONPatches := []patch.Json6902{}
	for _, jsonPatch := range kustomization.PatchesJson6902 {
		if excludedResource.Gvk().Equals(jsonPatch.Target.Gvk) {
			if jsonPatch.Target.Name == excludedResource.Name() {
				// don't add to new patch list
				continue
			}
		}
		newJSONPatches = append(newJSONPatches, jsonPatch)
	}
	kustomization.PatchesJson6902 = newJSONPatches

	newMergePatches := []patch.StrategicMerge{}
	for _, mergePatch := range kustomization.PatchesStrategicMerge {
		matches, err := mergePatchMatches(fs, basePath, string(mergePatch), excludedResource)
		if err != nil {
			return errors.Wrapf(err, "check if patch matches excluded resource")
		}

		if !matches {
			newMergePatches = append(newMergePatches, mergePatch)
		}
	}
	kustomization.PatchesStrategicMerge = newMergePatches

	for _, base := range kustomization.Bases {
		err = ExcludeKubernetesPatch(fs, filepath.Join(basePath, base), excludedResource)
		if err != nil {
			return errors.Wrapf(err, "exclude kubernetes patch %s from base %s of %s", excludedResource.String(), base, basePath)
		}
	}

	err = writeKustomization(fs, basePath, kustomization)
	if err != nil {
		return errors.Wrapf(err, "persist kustomization for %s", basePath)
	}

	return nil
}

// UnExcludeKubernetesResource finds a deleted resource in a child of the basePath and includes it again
func UnExcludeKubernetesResource(fs afero.Afero, basePath string, unExcludedResource string) error {
	kustomization, err := getKustomization(fs, basePath)
	if err != nil {
		return errors.Wrapf(err, "get kustomization for %s", basePath)
	}

	unExcludedResource = strings.TrimPrefix(unExcludedResource, string(filepath.Separator))

	// check if the resource is already included, if it is there is nothing left to do
	for _, existingResource := range kustomization.Resources {
		if existingResource == unExcludedResource {
			return nil
		}
	}

	resourceLength := len(kustomization.Resources)

	err = fs.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrapf(err, "walk %s", path)
		}
		relPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return errors.Wrapf(err, "get relative path to %s from %s", path, basePath)
		}

		if relPath == unExcludedResource {
			kustomization.Resources = append(kustomization.Resources, unExcludedResource)
		}

		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "walk files in %s", basePath)
	}

	if resourceLength != len(kustomization.Resources) {
		// write updated kustomization to disk - resource has been reincluded
		err = writeKustomization(fs, basePath, kustomization)
		if err != nil {
			return errors.Wrapf(err, "persist kustomization for %s", basePath)
		}
		return nil
	}

	for _, newBase := range kustomization.Bases {
		newBase = filepath.Clean(filepath.Join(basePath, newBase))
		cleanBase := strings.ReplaceAll(newBase, string(filepath.Separator), "-")

		if strings.HasPrefix(unExcludedResource, cleanBase) {
			updatedResource := strings.TrimPrefix(unExcludedResource, cleanBase)
			updatedResource = strings.TrimPrefix(updatedResource, string(filepath.Separator))

			return UnExcludeKubernetesResource(fs, newBase, updatedResource)
		}
	}

	return fmt.Errorf("unable to find resource %s in %s or its bases", unExcludedResource, basePath)
}

func getKustomization(fs afero.Afero, basePath string) (*types.Kustomization, error) {
	exists, err := fs.Exists(filepath.Join(basePath, "kustomization.yaml"))
	if err != nil {
		return nil, errors.Wrapf(err, "check kustomization yaml in %s", basePath)
	}
	if !exists {
		return nil, fmt.Errorf("kustomization in %s does not exist", basePath)
	}

	kustomizeYaml, err := fs.ReadFile(filepath.Join(basePath, "kustomization.yaml"))
	if err != nil {
		return nil, errors.Wrapf(err, "read kustomization yaml in %s", basePath)
	}

	kustomization := types.Kustomization{}
	err = yaml.Unmarshal(kustomizeYaml, &kustomization)
	if err != nil {
		return nil, errors.Wrapf(err, "unmarshal kustomization yaml from %s", basePath)
	}
	return &kustomization, nil
}

func writeKustomization(fs afero.Afero, basePath string, kustomization *types.Kustomization) error {
	kustomizeYaml, err := MarshalIndent(2, kustomization)
	if err != nil {
		return errors.Wrapf(err, "marshal kustomization yaml from %s", basePath)
	}

	err = fs.WriteFile(filepath.Join(basePath, "kustomization.yaml"), kustomizeYaml, 0644)
	if err != nil {
		return errors.Wrapf(err, "write kustomization yaml to %s", basePath)
	}
	return nil
}

// checks if the merge patch file at a given path matches the provided resource
func mergePatchMatches(fs afero.Afero, basePath string, mergePatch string, excludedResource resid.ResId) (bool, error) {
	// read contents of resource and convert it to ResID form
	patchBytes, err := fs.ReadFile(filepath.Join(basePath, string(mergePatch)))
	if err != nil {
		return false, errors.Wrapf(err, "read %s in %s to exclude patches for %s", string(mergePatch), basePath, excludedResource.String())
	}

	patchResources, err := NewKubernetesResources(patchBytes)
	if err != nil {
		return false, errors.Wrapf(err, "parse %s in %s to exclude patches for %s", string(mergePatch), basePath, excludedResource.String())
	}

	patchIDs := ResIDs(patchResources)

	for _, patchID := range patchIDs {
		if patchID.GvknEquals(excludedResource) {
			// this file of patches touches the excluded resource, and should be discarded
			return true, nil
		}
	}
	return false, nil
}
