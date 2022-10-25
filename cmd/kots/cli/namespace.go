package cli

import (
	"strings"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/tools/clientcmd"
)

func getNamespaceOrDefault(namespace string) (string, error) {
	if namespace == "" {
		clientCfg, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
		if err != nil {
			return "", errors.Wrap(err, "failed to load kubeconfig")
		}
		cctx := clientCfg.Contexts[clientCfg.CurrentContext]
		if cctx != nil {
			namespace = cctx.Namespace
		}
	}

	if namespace == "" {
		namespace = metav1.NamespaceDefault
	}

	err := validateNamespace(namespace)
	if err != nil {
		return "", errors.Wrapf(err, "invalid namespace %s", namespace)
	}

	return namespace, nil
}

func validateNamespace(namespace string) error {
	if namespace == "" {
		return errors.New("namespace is required")
	}
	if strings.Contains(namespace, "_") {
		return errors.New("a namespace should not contain the _ character")
	}

	errs := validation.IsValidLabelValue(namespace)
	if len(errs) > 0 {
		return errors.New(errs[0])
	}

	return nil
}
