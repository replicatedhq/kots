package util

import (
	"os"
	"sort"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
	yaml "gopkg.in/yaml.v2"
)

type PostKustomizeFile struct {
	Order   int
	Minimal MinimalK8sYaml
	Full    interface{}
}

type PostKustomizeFileCollection []PostKustomizeFile

func (c PostKustomizeFileCollection) Len() int {
	return len(c)
}

func (c PostKustomizeFileCollection) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}

func (c PostKustomizeFileCollection) Less(i, j int) bool {
	return c[i].Order < c[j].Order
}

type ListK8sYaml struct {
	APIVersion string        `json:"apiVersion" yaml:"apiVersion"`
	Kind       string        `json:"kind" yaml:"kind" hcl:"kind"`
	Items      []interface{} `json:"items" yaml:"items"`
}

func RebuildListYaml(debug log.Logger, lists []List, kustomizedYamlFiles []PostKustomizeFile) ([]PostKustomizeFile, error) {
	yamlMap := make(map[MinimalK8sYaml]PostKustomizeFile)

	for _, PostKustomizeFile := range kustomizedYamlFiles {
		yamlMap[PostKustomizeFile.Minimal] = PostKustomizeFile
	}

	fullReconstructedRendered := make([]PostKustomizeFile, 0)
	for _, list := range lists {
		var allListItems []interface{}
		for _, item := range list.Items {
			if pkFile, exists := yamlMap[item]; exists {
				delete(yamlMap, item)
				allListItems = append(allListItems, pkFile.Full)
			}
		}

		// don't render empty lists
		if len(allListItems) == 0 {
			continue
		}

		debug.Log("event", "reconstruct list")
		reconstructedList := ListK8sYaml{
			APIVersion: list.APIVersion,
			Kind:       "List",
			Items:      allListItems,
		}

		postKustomizeList := PostKustomizeFile{
			Minimal: MinimalK8sYaml{
				Kind: "List",
			},
			Full: reconstructedList,
		}

		fullReconstructedRendered = append(fullReconstructedRendered, postKustomizeList)
	}

	for nonListYamlMinimal, pkFile := range yamlMap {
		fullReconstructedRendered = append(fullReconstructedRendered, PostKustomizeFile{
			Order:   pkFile.Order,
			Minimal: nonListYamlMinimal,
			Full:    pkFile.Full,
		})
	}

	return fullReconstructedRendered, nil
}

func WritePostKustomizeFiles(debug log.Logger, FS afero.Afero, dest string, postKustomizeFiles []PostKustomizeFile) error {

	sort.Stable(PostKustomizeFileCollection(postKustomizeFiles))

	var joinedFinal string
	for _, file := range postKustomizeFiles {
		debug.Log("event", "marshal post kustomize file")
		fileB, err := yaml.Marshal(file.Full)
		if err != nil {
			return errors.Wrapf(err, "marshal file %s", file.Minimal.Metadata.Name)
		}

		if joinedFinal != "" {
			joinedFinal += "---\n" + string(fileB)
		} else {
			joinedFinal += string(fileB)
		}
	}

	debug.Log("event", "write post kustomize files", "dest", dest)
	if err := FS.WriteFile(dest, []byte(joinedFinal), os.FileMode(0644)); err != nil {
		return errors.Wrapf(err, "write kustomized and post processed yaml at %s", dest)
	}

	return nil
}
