package deploy

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/segmentio/ksuid"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func EnsureClientSecret(ctx context.Context, clientset kubernetes.Interface, namespace, namePrefix string) error {
	secret := ClientSecretResource(namePrefix, "")

	_, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secret.Name, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing secret")
		}

		_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create secret")
		}

		return nil
	}

	// no patch needed

	return nil
}

func RenderClientSecret(ctx context.Context, clientset kubernetes.Interface, namespace, namePrefix, existingClientSecret string) ([]byte, error) {
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	secret := ClientSecretResource(namePrefix, existingClientSecret)
	buf := bytes.NewBuffer(nil)
	if err := s.Encode(secret, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode secret")
	}

	return buf.Bytes(), nil
}

func GetClientSecret(ctx context.Context, clientset kubernetes.Interface, namespace, namePrefix string) (string, error) {
	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, prefixName(namePrefix, "dex-client"), metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return string(secret.Data["DEX_CLIENT_SECRET"]), nil
}

func ClientSecretResource(namePrefix, existingClientSecret string) *corev1.Secret {
	clientSecret := existingClientSecret
	if clientSecret == "" {
		clientSecret = ksuid.New().String()
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   prefixName(namePrefix, "dex-client"),
			Labels: kotsadmtypes.GetKotsadmLabels(AdditionalLabels(namePrefix)),
		},
		Data: map[string][]byte{
			"DEX_CLIENT_SECRET": []byte(clientSecret),
		},
	}
}
