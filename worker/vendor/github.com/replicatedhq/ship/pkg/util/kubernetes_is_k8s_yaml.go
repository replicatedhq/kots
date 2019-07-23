package util

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	yaml "gopkg.in/yaml.v3"
)

func ShouldAddFileToBase(fs *afero.Afero, excludedBases []string, targetPath string) bool {
	if filepath.Ext(targetPath) != ".yaml" && filepath.Ext(targetPath) != ".yml" {
		return false
	}

	for _, base := range excludedBases {
		basePathWOLeading := strings.TrimPrefix(base, "/")
		if basePathWOLeading == targetPath {
			return false
		}
	}

	if !IsK8sYaml(fs, targetPath) {
		return false
	}

	return !strings.HasSuffix(targetPath, "kustomization.yaml") &&
		!strings.HasSuffix(targetPath, "Chart.yaml") &&
		!strings.HasSuffix(targetPath, "values.yaml")
}

func IsK8sYaml(fs *afero.Afero, target string) bool {
	fileContents, err := fs.ReadFile(target)
	if err != nil {
		// if we cannot read a file, we assume that it is valid k8s yaml
		return true
	}

	var allMinimalYaml []MinimalK8sYaml
	dec := yaml.NewDecoder(bytes.NewReader(fileContents))
	for {
		minimal := MinimalK8sYaml{}
		err := dec.Decode(&minimal)
		if err == io.EOF {
			break
		} else if err != nil {
			// if we cannot unmarshal the file, it is not valid k8s yaml
			return false
		}
		allMinimalYaml = append(allMinimalYaml, minimal)
	}

	foundAcceptableDoc := false

	// if any of the documents is valid k8s yaml, we keep the file
	for _, minimal := range allMinimalYaml {
		if minimal.Kind == "" {
			// if there is not a kind, it is not valid k8s yaml
			continue
		}

		// k8s yaml must have a name OR be a list type
		if minimal.Metadata.Name != "" || strings.HasSuffix(minimal.Kind, "List") {
			foundAcceptableDoc = true
		}
	}

	return foundAcceptableDoc
}
