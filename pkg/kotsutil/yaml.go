package kotsutil

import (
	"bytes"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
	yaml "github.com/replicatedhq/yaml/v3"
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
