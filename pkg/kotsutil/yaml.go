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
	docs := bytes.Split(inputContent, []byte("\n---\n"))

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

// RemoveEmptyMappingFields removes empty mapping fields from a yaml document for the specific
// properties that kots applies kustomizations to. This is necessary because kustomize will
// fail to apply a kustomization if these fields contain null values: https://github.com/kubernetes-sigs/kustomize/issues/5050
//
// The properties are:
//
//   - metadata.labels
//   - metadata.annotations
//   - spec.containers
//   - spec.initContainers
//   - spec.template.spec.containers
//   - spec.template.spec.initContainers
//   - spec.imagePullSecrets
//   - spec.template.spec.imagePullSecrets
func RemoveEmptyMappingFields(input []byte) ([]byte, error) {
	var data map[string]interface{}
	err := k8syaml.Unmarshal([]byte(input), &data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal yaml")
	}

	removedItems := removeNilValuesFromMap(data)
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

func removeNilValuesFromMap(input map[string]interface{}) bool {
	removedItems := false
	if metadata, ok := input["metadata"].(map[string]interface{}); ok {
		if removeEmptyMappingsFromMetadata(metadata) {
			removedItems = true
		}
	}
	if spec, ok := input["spec"].(map[string]interface{}); ok {
		if removeEmptyMappingsFromSpec(spec) {
			removedItems = true
		}
		if template, ok := spec["template"].(map[string]interface{}); ok {
			if templateSpec, ok := template["spec"].(map[string]interface{}); ok {
				if removeEmptyMappingsFromSpec(templateSpec) {
					removedItems = true
				}
			}
			if templateMetadata, ok := template["metadata"].(map[string]interface{}); ok {
				if removeEmptyMappingsFromMetadata(templateMetadata) {
					removedItems = true
				}
			}
		}
	}

	return removedItems
}

func removeEmptyMappingsFromMetadata(metadata map[string]interface{}) bool {
	removedItems := false
	if labels, ok := metadata["labels"]; ok && labels == nil {
		delete(metadata, "labels")
		removedItems = true
	}
	if annotations, ok := metadata["annotations"]; ok && annotations == nil {
		delete(metadata, "annotations")
		removedItems = true
	}
	return removedItems
}

func removeEmptyMappingsFromSpec(spec map[string]interface{}) bool {
	removedItems := false
	if containers, ok := spec["containers"]; ok && containers == nil {
		delete(spec, "containers")
		removedItems = true
	}
	if initContainers, ok := spec["initContainers"]; ok && initContainers == nil {
		delete(spec, "initContainers")
		removedItems = true
	}
	if imagePullSecrets, ok := spec["imagePullSecrets"]; ok && imagePullSecrets == nil {
		delete(spec, "imagePullSecrets")
		removedItems = true
	}
	return removedItems
}
