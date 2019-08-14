package k8sutil

import (
	"io/ioutil"

	"github.com/pkg/errors"
	kustomizetypes "sigs.k8s.io/kustomize/v3/pkg/types"
	"sigs.k8s.io/yaml"
)

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
