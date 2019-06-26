package util

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/afero"
	yaml "gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/pkg/types"
)

// calls ExcludeKubernetesResource for each excluded resource. Returns after the first error.
func ExcludeKubernetesResources(fs afero.Afero, basePath string, excludedResources []string) error {
	for _, excludedResource := range excludedResources {
		err := ExcludeKubernetesResource(fs, basePath, excludedResource)
		if err != nil {
			return errors.Wrapf(err, "excluding %s from %s", excludedResource, basePath)
		}
	}
	return nil
}

// exclude the provided kubernetes resource file from the kustomization.yaml at basePath, or from bases imported from that.
func ExcludeKubernetesResource(fs afero.Afero, basePath string, excludedResource string) error {
	kustomizeYaml, err := fs.ReadFile(filepath.Join(basePath, "kustomization.yaml"))
	if err != nil {
		return errors.Wrapf(err, "read kustomization yaml in %s", basePath)
	}

	kustomization := types.Kustomization{}
	err = yaml.Unmarshal(kustomizeYaml, &kustomization)
	if err != nil {
		return errors.Wrapf(err, "unmarshal kustomization yaml from %s", basePath)
	}

	excludedResource = strings.TrimPrefix(excludedResource, string(filepath.Separator))

	newResources := []string{}
	for _, existingResource := range kustomization.Resources {
		if existingResource != excludedResource {
			newResources = append(newResources, existingResource)
		}
	}

	if len(newResources) != len(kustomization.Resources) {
		kustomization.Resources = newResources

		// write updated kustomization to disk - resource has been removed
		kustomizeYaml, err = yaml.Marshal(kustomization)
		if err != nil {
			return errors.Wrapf(err, "marshal kustomization yaml from %s", basePath)
		}

		err = fs.WriteFile(filepath.Join(basePath, "kustomization.yaml"), kustomizeYaml, 0644)
		if err != nil {
			return errors.Wrapf(err, "write kustomization yaml to %s", basePath)
		}
		return nil
	}

	// check if the resource is already removed from this dir
	alreadyRemoved := false
	err = fs.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			if err != nil {
				return err
			}
			relPath, err := filepath.Rel(basePath, path)
			if err != nil {
				return errors.Wrapf(err, "get relative path to %s from %s", path, basePath)
			}
			if relPath == excludedResource {
				alreadyRemoved = true
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "walk files in %s", basePath)
	}
	if alreadyRemoved {
		// the file to be removed exists within this base dir, and not within the kustomization yaml
		return nil
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

	return fmt.Errorf("unable to find resource %s in %s or its bases", excludedResource, basePath)
}

// TODO add a function to do the opposite of ExcludeKubernetesResource
