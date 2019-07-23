package util

import (
	"os"
	"sort"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
	"github.com/spf13/afero"
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

func WritePostKustomizeFiles(debug log.Logger, FS afero.Afero, dest string, postKustomizeFiles []PostKustomizeFile) error {
	sort.Stable(PostKustomizeFileCollection(postKustomizeFiles))

	var joinedFinal string
	for _, file := range postKustomizeFiles {
		debug.Log("event", "marshal post kustomize file")
		fileB, err := MarshalIndent(2, file.Full)
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
