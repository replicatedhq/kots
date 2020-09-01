package k8sutil

import (
	"io/ioutil"
	"sort"
	"strings"

	"github.com/pkg/errors"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/yaml"
)

func ReadKustomizationFromFile(file string) (*kustomizetypes.Kustomization, error) {
	b, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read kustomization file")
	}

	k := kustomizetypes.Kustomization{}
	if err := yaml.Unmarshal(b, &k); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal kustomization")
	}

	return &k, nil
}

// implementing Len Swap and Less allows sorting the type directly
type kustPatches []kustomizetypes.PatchStrategicMerge

func (s kustPatches) Len() int {
	return len(s)
}
func (s kustPatches) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s kustPatches) Less(i, j int) bool {
	return strings.Compare(string(s[i]), string(s[j])) < 0
}

func WriteKustomizationToFile(kustomization kustomizetypes.Kustomization, file string) error {
	// we remove newTags from here... because...
	cleanedImages := []kustomizetypes.Image{}
	for _, image := range kustomization.Images {
		image.NewTag = ""
		cleanedImages = append(cleanedImages, image)
	}
	kustomization.Images = cleanedImages

	sort.Strings(kustomization.Bases)
	sort.Strings(kustomization.Resources)
	sort.Sort(kustPatches(kustomization.PatchesStrategicMerge))

	b, err := yaml.Marshal(kustomization)
	if err != nil {
		return errors.Wrap(err, "failed to marshal kustomization")
	}

	if err := ioutil.WriteFile(file, b, 0644); err != nil {
		return errors.Wrap(err, "failed to write kustomization file")
	}

	return nil
}
