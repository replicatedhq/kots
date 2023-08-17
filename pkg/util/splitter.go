package util

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	yaml "github.com/replicatedhq/yaml/v3"
)

type OverlySimpleGVK struct {
	APIVersion string               `yaml:"apiVersion"`
	Kind       string               `yaml:"kind"`
	Metadata   OverlySimpleMetadata `yaml:"metadata"`
}

type OverlySimpleMetadata struct {
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

func SplitYAML(input []byte) (map[string][]byte, error) {
	outputFiles := map[string][]byte{}
	docs := YAMLBytesToSingleDocs(input)

	for _, doc := range docs {
		if bytes.HasPrefix(doc, []byte("---\n")) {
			doc = doc[4:]
		}

		if len(doc) == 0 {
			continue
		}

		filename, err := generateName(outputFiles, doc, 0)
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate name")
		}

		outputFiles[filename] = doc
	}

	return outputFiles, nil
}

func generateName(outputFiles map[string][]byte, content []byte, suffix int) (string, error) {
	o := OverlySimpleGVK{}

	if err := yaml.Unmarshal(content, &o); err != nil {
		return "", errors.Wrap(err, "failed to unmarshal yaml")
	}

	filename := fmt.Sprintf("%s-%s", o.Metadata.Name, strings.ToLower(o.Kind))
	if suffix > 0 {
		filename = fmt.Sprintf("%s-%d", filename, suffix)
	}
	filename = fmt.Sprintf("%s.yaml", filename)

	if _, exists := outputFiles[filename]; exists {
		return generateName(outputFiles, content, suffix+1)
	}

	return filename, nil
}
