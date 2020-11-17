package cli

import (
	"strings"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation"
)

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
