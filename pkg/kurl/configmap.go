package kurl

import (
	"context"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8s"
	"github.com/replicatedhq/kots/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const configMapName = "kurl-config"
const configMapNamespace = "kube-system"

const bootstrapTokenKey = "bootstrap_token"
const bootstrapTokenExpirationKey = "bootstrap_token_expiration"

const certKey = "cert_key"
const certsExpirationKey = "upload_certs_expiration"

func IsKurl() bool {
	clientset, err := k8s.Clientset()
	if err != nil {
		logger.Error(err)
		return false
	}

	_, err = ReadConfigMap(clientset)
	if err != nil {
		return false
	}

	return true
}

// ReadConfigMap will read the Kurl config from a configmap
func ReadConfigMap(client kubernetes.Interface) (*corev1.ConfigMap, error) {
	return client.CoreV1().ConfigMaps(configMapNamespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
}

// UpdateConfigMap will save the Kurl config in a configmap
func UpdateConfigMap(client kubernetes.Interface, generateBootstrapToken, uploadCerts bool) (*corev1.ConfigMap, error) {
	cm, err := client.CoreV1().ConfigMaps(configMapNamespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "get configmap")
	}

	// To be backwards compatible with kotsadm 1.1.0 and 1.2.0, if neither the bootstrap token nor
	// the upload certs flags are set then generate a token for a worker node
	if !uploadCerts {
		generateBootstrapToken = true
	}

	if generateBootstrapToken {
		bootstrapTokenDuration := time.Hour * 24
		bootstrapTokenExpiration := time.Now().Add(bootstrapTokenDuration)
		bootstrapToken, err := k8s.GenerateBootstrapToken(client, bootstrapTokenDuration)
		if err != nil {
			return nil, errors.Wrap(err, "generate bootstrap token")
		}

		cm.Data[bootstrapTokenKey] = bootstrapToken
		cm.Data[bootstrapTokenExpirationKey] = bootstrapTokenExpiration.Format(time.RFC3339)
	}

	if uploadCerts {
		certsDuration := time.Hour * 2
		certsExpiration := time.Now().Add(certsDuration)

		ctx, cancel := context.WithTimeout(context.TODO(), 60*time.Second)
		defer cancel()

		key, err := createCertAndKey(ctx, client, os.Getenv("POD_NAMESPACE"))
		if err != nil {
			return nil, errors.Wrap(err, "upload certs with new key")
		}

		cm.Data[certKey] = key
		cm.Data[certsExpirationKey] = certsExpiration.Format(time.RFC3339)
	}

	cm, err = client.CoreV1().ConfigMaps(configMapNamespace).Update(context.TODO(), cm, metav1.UpdateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "update configmap")
	}

	return cm, nil
}
