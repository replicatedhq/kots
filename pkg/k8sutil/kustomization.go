package k8sutil

import (
	"io/ioutil"

	"github.com/pkg/errors"
	kustomizetypes "sigs.k8s.io/kustomize/v3/pkg/types"
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

func WriteKustomizationToFile(kustomization *kustomizetypes.Kustomization, file string) error {
	b, err := yaml.Marshal(kustomization)
	if err != nil {
		return errors.Wrap(err, "failed to marshal kustomization")
	}

	if err := ioutil.WriteFile(file, b, 0644); err != nil {
		return errors.Wrap(err, "failed to write kustomization file")
	}

	return nil
}
