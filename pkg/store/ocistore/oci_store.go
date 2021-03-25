package ocistore

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/pkg/errors"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

/* OCIStore stores most data in an OCI compatible image repository,
   but does not make guarantees that every thing is stored there.
   Some data is stored locally in Kuberntes ConfigMaps and Secrets
   to speed up retrieval

   A note about "transactions": in the pg store, there were a few
   places that relied on transactions to ensure integrity
   Here, this is stored in configmaps and secrets, and this inegrity
   is provided by the Kubernetes API's enforcement of puts.
   If a caller GETs a configmap, updates it and then tries to PUT that
   configmap, but another process has modified it, the PUT will
   be rejected. This level of consistency is all that's needed for KOTS
*/
var (
	ErrNotFound       = errors.New("not found")
	ErrNotImplemented = errors.New("not implemented in ocistore")
)

type cachedTaskStatus struct {
	expirationTime time.Time
	taskStatus     taskStatus
}

type OCIStore struct {
	BaseURI   string
	PlainHTTP bool

	sessionSecret     *corev1.Secret
	sessionExpiration time.Time

	cachedTaskStatus map[string]*cachedTaskStatus
}

func (s *OCIStore) Init() error {
	return nil
}

func (s *OCIStore) WaitForReady(ctx context.Context) error {
	return nil
}

func (s *OCIStore) IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if aerr, ok := err.(awserr.Error); ok && aerr.Code() == "NotFound" {
		return true
	}
	if kuberneteserrors.IsNotFound(err) {
		return true
	}
	return errors.Cause(err) == ErrNotFound
}

func canIgnoreEtcdError(err error) bool {
	if err == nil {
		return true
	}

	if strings.Contains(err.Error(), "connection refused") {
		return true
	}

	if strings.Contains(err.Error(), "request timed out") {
		return true
	}

	return false
}

func StoreFromEnv() *OCIStore {
	return &OCIStore{
		BaseURI:   os.Getenv("STORAGE_BASEURI"),
		PlainHTTP: os.Getenv("STORAGE_BASEURI_PLAINHTTP") == "true",
	}
}

func (c OCIStore) GetClientset() (*kubernetes.Clientset, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kubernetes clientset")
	}

	return clientset, nil
}

func (s *OCIStore) getSecret(name string) (*corev1.Secret, error) {
	clientset, err := s.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	existingSecret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed to get secret")
	} else if kuberneteserrors.IsNotFound(err) {
		secret := corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: os.Getenv("POD_NAMESPACE"),
				Labels: map[string]string{
					"owner": "kotsadm",
				},
			},
			Data: map[string][]byte{},
		}

		createdSecret, err := clientset.CoreV1().Secrets(os.Getenv("POD_NAMESPACE")).Create(context.TODO(), &secret, metav1.CreateOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create secret")
		}

		return createdSecret, nil
	}

	return existingSecret, nil
}

func (s *OCIStore) getConfigmap(name string) (*corev1.ConfigMap, error) {
	clientset, err := s.GetClientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}

	existingConfigmap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed to get configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		configmap := corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: os.Getenv("POD_NAMESPACE"),
				Labels: map[string]string{
					"owner": "kotsadm",
				},
			},
			Data: map[string]string{},
		}

		createdConfigmap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Create(context.TODO(), &configmap, metav1.CreateOptions{})
		if err != nil {
			return nil, errors.Wrap(err, "failed to create configmap")
		}

		return createdConfigmap, nil
	}

	return existingConfigmap, nil
}

func (s *OCIStore) updateConfigmap(configmap *corev1.ConfigMap) error {
	clientset, err := s.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Update(context.Background(), configmap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}

func (s *OCIStore) ensureApplicationMetadata(applicationMetadata string, namespace string) error {
	clientset, err := s.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	existingConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), "kotsadm-application-metadata", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing metadata config map")
		}

		metadata := []byte(applicationMetadata)
		_, err := clientset.CoreV1().ConfigMaps(namespace).Create(context.TODO(), kotsadmobjects.ApplicationMetadataConfig(metadata, namespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create metadata config map")
		}

		return nil
	}

	if existingConfigMap.Data == nil {
		existingConfigMap.Data = map[string]string{}
	}

	existingConfigMap.Data["application.yaml"] = applicationMetadata

	_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Update(context.Background(), existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}

	return nil
}
