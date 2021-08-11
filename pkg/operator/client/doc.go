package client

import (
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func splitMutlidocYAMLIntoFirstApplyAndOthers(multidoc []byte) ([]byte, []byte, error) {
	firstApply := []string{}
	other := []string{}

	docs := strings.Split(string(multidoc), "\n---\n")
	for _, doc := range docs {
		if IsCRD([]byte(doc)) {
			firstApply = append(firstApply, doc)
		} else if IsNamespace([]byte(doc)) {
			firstApply = append(firstApply, doc)
		} else {
			other = append(other, doc)
		}
	}

	// if there were no crds, don't return the touched docs, keep the original
	if len(firstApply) == 0 {
		return nil, multidoc, nil
	}

	return []byte(strings.Join(firstApply, "\n---\n")), []byte(strings.Join(other, "\n---\n")), nil
}

func docsByNamespace(multidoc []byte, defaultNamespace string) (map[string][]byte, error) {
	byNamespace := map[string][]string{}

	docs := strings.Split(string(multidoc), "\n---\n")
	for _, doc := range docs {
		o := OverlySimpleGVKWithName{}

		if err := yaml.Unmarshal([]byte(doc), &o); err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal doc to look for namespace")
		}

		namespace := o.Metadata.Namespace
		if namespace == "" {
			namespace = defaultNamespace
		}

		_, ok := byNamespace[namespace]
		if ok {
			byNamespace[namespace] = append(byNamespace[namespace], doc)
		} else {
			byNamespace[namespace] = []string{doc}
		}
	}

	result := map[string][]byte{}
	for k, v := range byNamespace {
		result[k] = []byte(strings.Join(v, "\n---\n"))
	}

	return result, nil
}
