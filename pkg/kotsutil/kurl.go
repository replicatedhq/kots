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

func GetKurlRegistryCreds() (hostname string, username string, password string, finalErr error) {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		finalErr = errors.Wrap(err, "failed to get k8s clientset")
		return
	}

	// kURL registry secret is always in default namespace
	secret, err := clientset.CoreV1().Secrets("default").Get(context.TODO(), "registry-creds", metav1.GetOptions{})
	if err != nil {
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
		if auth.Username == "kurl" {
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

func IsKurl(clientset kubernetes.Interface) bool {
	_, err := clientset.CoreV1().ConfigMaps("kube-system").Get(context.TODO(), "kurl-config", metav1.GetOptions{})
	if err != nil {
		return false
	}

	return true
}
