package kotsadm

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func JwtSecret(namespace string, jwt string) *corev1.Secret {
	if jwt == "" {
		jwt = uuid.New().String()
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-session",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Data: map[string][]byte{
			"key": []byte(jwt),
		},
	}

	return secret
}

func RqliteSecret(namespace string, password string) *corev1.Secret {
	if password == "" {
		password = uuid.New().String()
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-rqlite",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Data: map[string][]byte{
			"uri":             []byte(fmt.Sprintf("http://kotsadm:%s@kotsadm-rqlite.%s.svc.cluster.local:4001?timeout=60&disableClusterDiscovery=true", password, namespace)),
			"password":        []byte(password),
			"authconfig.json": []byte(fmt.Sprintf(`[{"username": "kotsadm", "password": "%s", "perms": ["all"]}, {"username": "*", "perms": ["status", "ready"]}]`, password)),
		},
	}

	return secret
}

func SharedPasswordSecret(namespace string, bcryptPassword string) *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-password",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Data: map[string][]byte{
			"passwordBcrypt": []byte(bcryptPassword),
		},
	}

	return secret
}

func S3Secret(namespace string, accessKey string, secretKey string) *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-minio",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Data: map[string][]byte{
			"accesskey": []byte(accessKey),
			"secretkey": []byte(secretKey),
		},
	}

	return secret
}

func ApiEncryptionKeySecret(namespace string, key string) *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-encryption",
			Namespace: namespace,
			Labels:    types.GetKotsadmLabels(),
		},
		Data: map[string][]byte{
			"encryptionKey": []byte(key),
		},
	}

	return secret
}

func ApiClusterTokenSecret(deployOptions types.DeployOptions) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: deployOptions.Namespace,
			Name:      types.ClusterTokenSecret,
			Labels:    types.GetKotsadmLabels(),
		},
		StringData: map[string]string{
			types.ClusterTokenSecret: deployOptions.AutoCreateClusterToken,
		},
	}
}

func PrivateKotsadmRegistrySecret(namespace string, registryConfig types.RegistryConfig) *corev1.Secret {
	return kotsadmversion.KotsadmPullSecret(namespace, registryConfig)
}
