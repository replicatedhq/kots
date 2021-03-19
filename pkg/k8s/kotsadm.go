package k8s

import (
	"context"
	"os"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	KotsadmIDConfigMapName = "kotsadm-id"
)

func FindKotsadmImage(namespace string) (string, error) {
	client, err := Clientset()
	if err != nil {
		return "", errors.Wrap(err, "failed to get k8s client set")
	}

	if os.Getenv("KOTSADM_ENV") == "dev" {
		namespace = os.Getenv("POD_NAMESPACE")
	}

	kotsadmDeployment, err := client.AppsV1().Deployments(namespace).Get(context.TODO(), "kotsadm", metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get kotsadm deployment")
	}

	apiContainerIndex := -1
	for i, container := range kotsadmDeployment.Spec.Template.Spec.Containers {
		if container.Name == "kotsadm" {
			apiContainerIndex = i
			break
		}
	}

	if apiContainerIndex == -1 {
		return "", errors.New("kotsadm container not found")
	}

	kotsadmImage := kotsadmDeployment.Spec.Template.Spec.Containers[apiContainerIndex].Image

	return kotsadmImage, nil
}

// IsKotsadmClusterScoped will check if kotsadm has cluster scope access or not
func IsKotsadmClusterScoped(ctx context.Context, clientset kubernetes.Interface, namespace string) bool {
	rb, err := clientset.RbacV1().ClusterRoleBindings().Get(ctx, "kotsadm-rolebinding", metav1.GetOptions{})
	if err != nil {
		return false
	}
	for _, s := range rb.Subjects {
		if s.Kind != "ServiceAccount" {
			continue
		}
		if s.Name != "kotsadm" {
			continue
		}
		if s.Namespace != "" && s.Namespace == namespace {
			return true
		}
		if s.Namespace == "" && namespace == metav1.NamespaceDefault {
			return true
		}
	}
	return false
}

func GetKotsadmIDConfigMap() (*corev1.ConfigMap, error) {
	clientset, err := Clientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}
	namespace := os.Getenv("POD_NAMESPACE")
	existingConfigmap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), KotsadmIDConfigMapName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed to get configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		return nil, nil
	}
	return existingConfigmap, nil
}

func CreateKotsadmIDConfigMap(kotsadmID string) error {
	var err error = nil
	clientset, err := Clientset()
	if err != nil {
		return err
	}
	configmap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      KotsadmIDConfigMapName,
			Namespace: os.Getenv("POD_NAMESPACE"),
			Labels: map[string]string{
				kotsadmtypes.KotsadmKey: kotsadmtypes.KotsadmLabelValue,
				kotsadmtypes.ExcludeKey: kotsadmtypes.ExcludeValue,
			},
		},
		Data: map[string]string{"id": kotsadmID},
	}
	_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Create(context.TODO(), &configmap, metav1.CreateOptions{})
	return err
}

func IsKotsadmIDConfigMapPresent() (bool, error) {
	clientset, err := Clientset()
	if err != nil {
		return false, errors.Wrap(err, "failed to get clientset")
	}
	namespace := os.Getenv("POD_NAMESPACE")
	_, err = clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), KotsadmIDConfigMapName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return false, errors.Wrap(err, "failed to get configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		return false, nil
	}
	return true, nil
}

func UpdateKotsadmIDConfigMap(kotsadmID string) error {
	clientset, err := Clientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}
	namespace := os.Getenv("POD_NAMESPACE")
	existingConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), KotsadmIDConfigMapName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		return nil
	}
	if existingConfigMap.Data == nil {
		existingConfigMap.Data = map[string]string{}
	}
	existingConfigMap.Data["id"] = kotsadmID

	_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Update(context.Background(), existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}
	return nil
}
