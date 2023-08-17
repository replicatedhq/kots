package kotsutil

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
	yaml "github.com/replicatedhq/yaml/v3"
	k8syaml "sigs.k8s.io/yaml"
)

// FixUpYAML is a general purpose function that will ensure that YAML is compatible with KOTS
// This ensures that lines aren't wrapped at 80 chars which breaks template functions
func FixUpYAML(inputContent []byte) ([]byte, error) {
	docs := util.ConvertToSingleDocs(inputContent)

	fixedUpDocs := make([][]byte, 0)
	for _, doc := range docs {
		yamlObj := map[string]interface{}{}

		err := yaml.Unmarshal(doc, &yamlObj)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal yaml")
		}

		fixedUpDoc, err := util.MarshalIndent(2, yamlObj)
		if err != nil {
			return nil, errors.Wrap(err, "failed to marshal yaml")
		}

		fixedUpDocs = append(fixedUpDocs, fixedUpDoc)
	}

	// MarshalIndent add a line break at the end of each file
	return bytes.Join(fixedUpDocs, []byte("---\n")), nil
}

// RemoveNilFieldsFromYAML removes nil fields from a yaml document.
// This is necessary because kustomize will fail to apply a kustomization if these fields contain nil values: https://github.com/kubernetes-sigs/kustomize/issues/5050
func RemoveNilFieldsFromYAML(input []byte) ([]byte, error) {
	var data map[string]interface{}
	err := k8syaml.Unmarshal([]byte(input), &data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal yaml")
	}

	removedItems := removeNilFieldsFromMap(data)
	if !removedItems {
		// no changes were made, return the original input
		return input, nil
	}

	output, err := k8syaml.Marshal(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal yaml")
	}

	return output, nil
}

func removeNilFieldsFromMap(input map[string]interface{}) bool {
	removedItems := false

	for key, value := range input {
		if value == nil {
			delete(input, key)
			removedItems = true
			continue
		}

		if valueMap, ok := value.(map[string]interface{}); ok {
			removedItems = removeNilFieldsFromMap(valueMap) || removedItems
			continue
		}

		if valueSlice, ok := value.([]interface{}); ok {
			for idx := range valueSlice {
				if itemMap, ok := valueSlice[idx].(map[string]interface{}); ok {
					removedItems = removeNilFieldsFromMap(itemMap) || removedItems
				}
			}
			continue
		}
	}

	return removedItems
}
