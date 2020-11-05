package kotsutil

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

func GetKurlRegistryCreds() (hostname string, username string, password string, finalErr error) {
	cfg, err := config.GetConfig()
	if err != nil {
		finalErr = errors.Wrap(err, "failed to get cluster config")
		return
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		finalErr = errors.Wrap(err, "failed to create kubernetes clientset")
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

func SyncKurlKotsadmS3Secret(k8sConfigFlags *genericclioptions.ConfigFlags) error {
	clientset, err := k8sutil.GetClientset(k8sConfigFlags)
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	s3Secret, err := clientset.CoreV1().Secrets("default").Get(context.TODO(), "kotsadm-s3", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get s3 secret")
	}

	s3Endpoint := string(s3Secret.Data["endpoint"])

	if strings.Contains(s3Endpoint, "rook-ceph") {
		rookSecret, err := clientset.CoreV1().Secrets("rook-ceph").Get(context.TODO(), "rook-ceph-object-user-rook-ceph-store-kurl", metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to get rook-ceph-object-user-rook-ceph-store-kurl secret")
		}
		s3Secret.Data["access-key-id"] = rookSecret.Data["AccessKey"]
		s3Secret.Data["secret-access-key"] = rookSecret.Data["SecretKey"]

		rookService, err := clientset.CoreV1().Services("rook-ceph").Get(context.TODO(), "rook-ceph-rgw-rook-ceph-store", metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to get rook-ceph-rgw-rook-ceph-store service")
		}
		s3Secret.Data["object-store-cluster-ip"] = []byte(rookService.Spec.ClusterIP)

		_, err = clientset.CoreV1().Secrets("default").Update(context.TODO(), s3Secret, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to update s3 secret from rook secret")
		}

		return nil
	}

	if strings.Contains(s3Endpoint, "minio") {
		minioSecret, err := clientset.CoreV1().Secrets("minio").Get(context.TODO(), "minio-credentials", metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to get minio-credentials secret")
		}
		s3Secret.Data["access-key-id"] = minioSecret.Data["MINIO_ACCESS_KEY"]
		s3Secret.Data["secret-access-key"] = minioSecret.Data["MINIO_SECRET_KEY"]

		minioService, err := clientset.CoreV1().Services("minio").Get(context.TODO(), "minio", metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to get minio service")
		}
		s3Secret.Data["object-store-cluster-ip"] = []byte(minioService.Spec.ClusterIP)

		_, err = clientset.CoreV1().Secrets("default").Update(context.TODO(), s3Secret, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to update s3 secret from minio secret")
		}

		return nil
	}

	return errors.Errorf("unsupported object store %s", s3Endpoint)
}
