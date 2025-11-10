package kotsadm

import (
	"bytes"
	"context"

	"github.com/pkg/errors"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getLicenseSecretYAML(deployOptions *types.DeployOptions) (map[string][]byte, error) {
	if deployOptions.License == nil {
		return nil, errors.New("deploy options license is nil")
	}

	if !deployOptions.License.IsV1() && !deployOptions.License.IsV2() {
		return nil, errors.New("no license to encode")
	}

	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	// Encode the actual license object (V1 or V2), not the wrapper
	if deployOptions.License.IsV1() {
		if err := s.Encode(deployOptions.License.V1, &b); err != nil {
			return nil, errors.Wrap(err, "failed to encode v1beta1 license")
		}
	} else if deployOptions.License.IsV2() {
		if err := s.Encode(deployOptions.License.V2, &b); err != nil {
			return nil, errors.Wrap(err, "failed to encode v1beta2 license")
		}
	}

	var license bytes.Buffer
	if err := s.Encode(kotsadmobjects.LicenseSecret(deployOptions.Namespace, deployOptions.License.GetAppSlug(), deployOptions.Airgap, b.String()), &license); err != nil {
		return nil, errors.Wrap(err, "failed to marshal license secret")
	}
	docs["secret-license.yaml"] = license.Bytes()

	return docs, nil
}

func ensureLicenseSecret(deployOptions *types.DeployOptions, clientset *kubernetes.Clientset) (bool, error) {
	existingSecret, err := getLicenseSecret(deployOptions.Namespace, clientset)
	if err != nil {
		return false, errors.Wrap(err, "failed to check for existing license secret")
	}

	if existingSecret != nil {
		return false, nil
	}

	_, err = clientset.CoreV1().Secrets(deployOptions.Namespace).Create(context.TODO(), kotsadmobjects.LicenseSecret(deployOptions.Namespace, deployOptions.License.GetAppSlug(), deployOptions.Airgap, deployOptions.LicenseData), metav1.CreateOptions{})
	if err != nil {
		return false, errors.Wrap(err, "failed to create license secret")
	}

	return true, nil
}

func getLicenseSecret(namespace string, clientset *kubernetes.Clientset) (*corev1.Secret, error) {
	licenseSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), "kotsadm-default-license", metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to get license secret from cluster")
	}

	return licenseSecret, nil
}
