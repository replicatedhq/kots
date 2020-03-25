package util

import (
	"github.com/pkg/errors"
	yaml "github.com/replicatedhq/yaml/v3"
)

// FixUpYAML is a general purpose function that will ensure that YAML is copmatible with KOTS
// This ensures that lines aren't wrapped at 80 chars which breaks template functions
func FixUpYAML(inputContent []byte) ([]byte, error) {
	yamlObj := map[string]interface{}{}

	err := yaml.Unmarshal(inputContent, &yamlObj)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal yaml")
	}

	inputContent, err = MarshalIndent(2, yamlObj)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal yaml")
	}

	return inputContent, nil
}
