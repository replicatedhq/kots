package kotsadm

import (
	"fmt"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func jwtSecret(namespace string, jwt string) *corev1.Secret {
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
		},
		Data: map[string][]byte{
			"key": []byte(jwt),
		},
	}

	return secret
}

func pgSecret(namespace string, password string) *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-postgres",
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"uri": []byte(fmt.Sprintf("postgresql://kotsadm:%s@kotsadm-postgres/kotsadm?connect_timeout=10&sslmode=disable", password)),
		},
	}

	return secret
}

func sharedPasswordSecret(namespace string, bcryptPassword string) *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-password",
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"passwordBcrypt": []byte(bcryptPassword),
		},
	}

	return secret
}

func s3Secret(namespace string, accessKey string, secretKey string) *corev1.Secret {
	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kotsadm-minio",
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"accesskey": []byte(accessKey),
			"secretkey": []byte(secretKey),
		},
	}

	return secret
}
