package helm

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

const (
	HelmReleaseSecretType = "helm.sh/release.v1"
)

// MigrateExistingHelmReleaseSecrets will move all helm release secrets from the kotsadm namespace to the release namespace
func MigrateExistingHelmReleaseSecrets(clientset kubernetes.Interface, releaseName string, releaseNamespace string, kotsadmNamespace string) error {
	selectorLabels := labels.Set{
		"owner": "helm",
		"name":  releaseName,
	}
	fieldSelectorMap := fields.Set{
		"type": HelmReleaseSecretType,
	}
	listOpts := metav1.ListOptions{
		LabelSelector: selectorLabels.AsSelector().String(),
		FieldSelector: fieldSelectorMap.AsSelector().String(),
	}

	secretList, err := clientset.CoreV1().Secrets(kotsadmNamespace).List(context.TODO(), listOpts)
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil
		}
		return errors.Wrapf(err, "failed to list release secrets for %s", releaseName)
	}

	if len(secretList.Items) == 0 {
		return nil
	}

	for _, secret := range secretList.Items {
		err := moveHelmReleaseSecret(clientset, secret, releaseNamespace)
		if err != nil {
			return errors.Wrapf(err, "failed to move helm release secret %s to %s", secret.Name, releaseNamespace)
		}
	}
	return nil
}

// moveHelmReleaseSecret will create a new secret in the releaseNamespace and delete the old one from the kotsadmNamespace
func moveHelmReleaseSecret(clientset kubernetes.Interface, secret corev1.Secret, releaseNamespace string) error {
	release, err := HelmReleaseFromSecretData(secret.Data["release"])
	if err != nil {
		return errors.Wrapf(err, "failed to get release from secret data")
	}

	// set release namespace to the new namespace
	release.Namespace = releaseNamespace
	releaseStr, err := encodeRelease(release)
	if err != nil {
		return errors.Wrapf(err, "failed to encode release")
	}

	// update the secret 
	newReleaseSecret := secret.DeepCopy()
	newReleaseSecret.Namespace = releaseNamespace
	newReleaseSecret.Data["release"] = []byte(releaseStr)


	// newReleaseSecret := corev1.Secret{
	// 	Type: secret.Type,
	// 	TypeMeta: metav1.TypeMeta{
	// 		Kind:       secret.Kind,
	// 		APIVersion: secret.APIVersion,
	// 	},
	// 	ObjectMeta: metav1.ObjectMeta{
	// 		Name:      secret.Name,
	// 		Labels:    secret.Labels,
	// 		Namespace: releaseNamespace,
	// 	},
	// 	StringData: map[string]string{
	// 		"release": releaseStr,
	// 	},
	// }

	_, err = clientset.CoreV1().Secrets(releaseNamespace).Create(context.TODO(), newReleaseSecret, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to create secret %s/%s", releaseNamespace, secret.Name)
	}

	// err = clientset.CoreV1().Secrets(secret.Namespace).Delete(context.TODO(), secret.Name, metav1.DeleteOptions{})
	// if err != nil {
	// 	return errors.Wrapf(err, "failed to delete secret %s/%s", secret.Namespace, secret.Name)
	// }

	return nil
}

func encodeRelease(helmRelease *release.Release) (string, error) {
	b, err := json.Marshal(helmRelease)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	w, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return "", err
	}
	if _, err = w.Write(b); err != nil {
		return "", err
	}
	w.Close()

	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}
