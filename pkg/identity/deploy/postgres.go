package deploy

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	kotsv1beta "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/segmentio/ksuid"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func EnsurePostgresSecret(ctx context.Context, clientset kubernetes.Interface, namespace, namePrefix string, config kotsv1beta.IdentityPostgresConfig) error {
	secret := postgresSecretResource(namePrefix, config)

	existingSecret, err := GetPostgresSecret(ctx, clientset, namespace, namePrefix)
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

	existingSecret = updatePostgresSecret(existingSecret, secret)

	_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update secret")
	}

	return nil
}

func RenderPostgresSecret(ctx context.Context, namePrefix string, config kotsv1beta.IdentityPostgresConfig) ([]byte, error) {
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	secret := postgresSecretResource(namePrefix, config)
	buf := bytes.NewBuffer(nil)
	if err := s.Encode(secret, buf); err != nil {
		return nil, errors.Wrap(err, "failed to encode secret")
	}

	return buf.Bytes(), nil
}

func GetPostgresSecret(ctx context.Context, clientset kubernetes.Interface, namespace, namePrefix string) (*corev1.Secret, error) {
	return clientset.CoreV1().Secrets(namespace).Get(ctx, prefixName(namePrefix, "dex-postgres"), metav1.GetOptions{})
}

func postgresSecretResource(namePrefix string, config kotsv1beta.IdentityPostgresConfig) *corev1.Secret {
	if config.Password == "" {
		config.Password = ksuid.New().String()
	}
	data := map[string][]byte{
		"PGHOST":     []byte(config.Host),
		"PGDATABASE": []byte(config.Database),
		"PGUSER":     []byte(config.User),
		"PGPASSWORD": []byte(config.Password),
	}
	if config.Port != "" {
		data["PGPORT"] = []byte(config.Port)
	}

	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   prefixName(namePrefix, "dex-postgres"),
			Labels: kotsadmtypes.GetKotsadmLabels(AdditionalLabels(namePrefix)),
		},
		Data: data,
	}
}

func updatePostgresSecret(existingSecret, desiredSecret *corev1.Secret) *corev1.Secret {
	if len(existingSecret.Data["PGHOST"]) == 0 {
		existingSecret.Data["PGHOST"] = desiredSecret.Data["PGHOST"]
		if len(existingSecret.Data["PGPORT"]) == 0 && len(desiredSecret.Data["PGPORT"]) > 0 {
			existingSecret.Data["PGPORT"] = desiredSecret.Data["PGPORT"]
		}
	}
	if len(existingSecret.Data["PGDATABASE"]) == 0 {
		existingSecret.Data["PGDATABASE"] = desiredSecret.Data["PGDATABASE"]
	}
	if len(existingSecret.Data["PGUSER"]) == 0 {
		existingSecret.Data["PGUSER"] = desiredSecret.Data["PGUSER"]
	}
	if len(existingSecret.Data["password"]) > 0 { // migrate to PGPASSWORD
		existingSecret.Data["PGPASSWORD"] = existingSecret.Data["password"]
		delete(existingSecret.Data, "password")
	} else if len(existingSecret.Data["PGPASSWORD"]) == 0 {
		existingSecret.Data["PGPASSWORD"] = desiredSecret.Data["PGPASSWORD"]
	}

	return existingSecret
}
