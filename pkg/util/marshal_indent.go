package util

import (
	"bytes"

	"github.com/pkg/errors"
	yaml "github.com/replicatedhq/yaml/v3" // using replicatedhq/yaml/v3 because that allows setting line length
)

func MarshalIndent(indent int, in interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(indent)
	enc.SetLineLength(-1)
	err := enc.Encode(in)
	if err != nil {
		return nil, errors.Wrapf(err, "marshal with indent %d", indent)
	}

	return buf.Bytes(), nil
}
