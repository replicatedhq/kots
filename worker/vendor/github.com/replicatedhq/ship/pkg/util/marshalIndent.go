package util

import (
	"bytes"

	"github.com/pkg/errors"
	yaml "gopkg.in/laverya/yaml.v3"
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
