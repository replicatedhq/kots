package redact

import (
	"os"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

// GetRedactSpec returns the redaction yaml spec, a pretty error string, and the underlying error
func GetRedactSpec() (string, string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "", "failed to get cluster config", errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "", "failed to create kubernetes clientset", errors.Wrap(err, "failed to create kubernetes clientset")
	}

	configMap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Get("kotsadm-redact", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			// not a not found error, so a real error
			return "", "failed to get kotsadm-redact configMap", errors.Wrap(err, "failed to get kotsadm-redact configMap")
		} else {
			// not found, so return empty string
			return "", "", nil
		}
	}

	encodedData, ok := configMap.Data["kotsadm-redact"]
	if !ok {
		return "", "failed to read kotadm-redact key in configmap", errors.New("failed to read kotadm-redact key in configmap")
	}

	return encodedData, "", nil
}

// SetRedactSpec sets the global redact spec to the specified string, and returns a pretty error string + the underlying error
func SetRedactSpec(spec string) (string, error) {
	cfg, err := config.GetConfig()
	if err != nil {
		return "failed to get cluster config", errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return "failed to create kubernetes clientset", errors.Wrap(err, "failed to create kubernetes clientset")
	}

	configMap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Get("kotsadm-redact", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			// not a not found error, so a real error
			return "failed to get kotsadm-redact configMap", errors.Wrap(err, "failed to get kotsadm-redact configMap")
		} else {
			// not found, so create it fresh
			newMap := v1.ConfigMap{
				TypeMeta: metav1.TypeMeta{
					Kind:       "ConfigMap",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "kotsadm-redact",
					Namespace: os.Getenv("POD_NAMESPACE"),
					Labels: map[string]string{
						"kots.io/kotsadm": "true",
					},
				},
				Data: map[string]string{
					"kotsadm-redact": spec,
				},
			}
			_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Create(&newMap)
			if err != nil {
				return "failed to create kotsadm-redact configMap", errors.Wrap(err, "failed to create kotsadm-redact configMap")
			}
			return "", nil
		}
	}

	configMap.Data["kotsadm-redact"] = spec
	_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Update(configMap)
	if err != nil {
		return "failed to update kotsadm-redact configMap", errors.Wrap(err, "failed to update kotsadm-redact configMap")
	}
	return "", nil
}
