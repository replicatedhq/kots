package kotsutil

import (
	"context"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func HasEmbeddedRegistry(clientset kubernetes.Interface) bool {
	var secret *corev1.Secret
	var err error

	// kURL registry secret is always in the 'default' namespace
	// Embedded cluster registry secret is always in the 'kotsadm' namespace

	for _, ns := range []string{"default", "kotsadm"} {
		secret, err = clientset.CoreV1().Secrets(ns).Get(context.TODO(), "registry-creds", metav1.GetOptions{})
		if err == nil {
			break
		}
	}

	if secret != nil {
		if secret.Type == corev1.SecretTypeDockerConfigJson {
			return true
		}
	}

	return false
}

func GetEmbeddedRegistryCreds(clientset kubernetes.Interface) (hostname string, username string, password string) {
	var secret *corev1.Secret
	var err error

	// kURL registry secret is always in the 'default' namespace
	// Embedded cluster registry secret is always in the 'kotsadm' namespace

	for _, ns := range []string{"default", "kotsadm"} {
		secret, err = clientset.CoreV1().Secrets(ns).Get(context.TODO(), "registry-creds", metav1.GetOptions{})
		if err == nil {
			break
		}
	}
	if secret == nil {
		return
	}

	dockerJson, ok := secret.Data[".dockerconfigjson"]
	if !ok {
		return
	}

	type dockerRegistryAuth struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Auth     string `json:"auth"`
	}
	dockerConfig := struct {
		Auths map[string]dockerRegistryAuth `json:"auths"`
	}{}

	err = json.Unmarshal(dockerJson, &dockerConfig)
	if err != nil {
		return
	}

	for host, auth := range dockerConfig.Auths {
		if auth.Username == "kurl" || auth.Username == "embedded-cluster" {
			hostname = host
			username = auth.Username
			password = auth.Password
			return
		}
	}

	return
}

func GetKurlS3Secret() (*corev1.Secret, error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get k8s clientset")
	}

	secret, err := clientset.CoreV1().Secrets("default").Get(context.TODO(), "kotsadm-s3", metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get secret")
	}

	return secret, nil
}
