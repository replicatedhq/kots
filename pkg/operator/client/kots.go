package client

import (
	"strings"

	"github.com/pkg/errors"
	troubleshootclientsetscheme "github.com/replicatedhq/troubleshoot/pkg/client/troubleshootclientset/scheme"
	"k8s.io/client-go/kubernetes/scheme"
)

func removeInternalGVK(manifests []byte) ([]byte, error) {
	cleaned := []string{}

	splitRenderedContents := strings.Split(string(manifests), "\n---\n")
	troubleshootclientsetscheme.AddToScheme(scheme.Scheme)
	for _, splitRenderedContent := range splitRenderedContents {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		_, gvk, err := decode([]byte(splitRenderedContent), nil, nil)
		if err != nil {
			return nil, errors.Wrap(err, "unable to decode yaml")
		}

		if gvk.Group == "troubleshoot.replicated.com" {
			if gvk.Version == "v1beta1" {
				if gvk.Kind == "Collector" {
					continue
				}

				if gvk.Kind == "Preflight" {
					continue
				}
			}
		}
		if gvk.Group == "velero.io" && gvk.Version == "v1" && gvk.Kind == "Backup" {
			continue
		}

		cleaned = append(cleaned, splitRenderedContent)
	}

	return []byte(strings.Join(cleaned, "\n---\n")), nil
}
