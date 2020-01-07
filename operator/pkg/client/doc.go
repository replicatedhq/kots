package client

import (
	"strings"
)

func splitMutlidocYAMLIntoCRDsAndOthers(multidoc []byte) ([]byte, []byte, error) {
	crds := []string{}
	other := []string{}

	docs := strings.Split(string(multidoc), "\n---\n")
	for _, doc := range docs {
		if IsCRD([]byte(doc)) {
			crds = append(crds, doc)
		} else {
			other = append(other, doc)
		}
	}

	// if there were no crds, don't return the touched docs, keep the original
	if len(crds) == 0 {
		return nil, multidoc, nil
	}

	return []byte(strings.Join(crds, "\n---\n")), []byte(strings.Join(other, "\n---\n")), nil
}
