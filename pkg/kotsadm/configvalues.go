package kotsadm

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func ensureConfigValuesSecret(deployOptions *types.DeployOptions, clientset kubernetes.Interface) (bool, error) {
	existingSecret, err := getConfigValuesSecret(deployOptions.Namespace, clientset)
	if err != nil {
		return false, errors.Wrap(err, "failed to check for existing config values secret")
	}

	if existingSecret != nil {
		return false, nil
	}

	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)
	var b bytes.Buffer
	if err := s.Encode(deployOptions.ConfigValues, &b); err != nil {
		return false, errors.Wrap(err, "failed to encode config values")
	}

	_, err = clientset.CoreV1().Secrets(deployOptions.Namespace).Create(context.TODO(), kotsadmobjects.ConfigValuesSecret(deployOptions.Namespace, b.String()), metav1.CreateOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to create config values secret")
	}

	return true, nil
}

func getConfigValuesSecret(namespace string, clientset kubernetes.Interface) (*corev1.Secret, error) {
	configValuesSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), "kotsadm-default-configvalues", metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to get config values secret from cluster")
	}

	return configValuesSecret, nil
}
