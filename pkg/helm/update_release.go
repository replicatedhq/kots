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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// MigrateExistingHelmReleaseSecrets will move all helm release secrets from the kotsadm namespace to the release namespace
func MigrateExistingHelmReleaseSecrets(clientset kubernetes.Interface, releaseName string, releaseNamespace string, kotsadmNamespace string) error {
	selectorLabels := map[string]string{
		"owner": "helm",
		"name":  releaseName,
	}
	listOpts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	}

	// list all helm releases secrets for given release name
	secretList, err := clientset.CoreV1().Secrets(kotsadmNamespace).List(context.TODO(), listOpts)
	if err != nil {
		return errors.Wrapf(err, "failed to list release secrets for %s", releaseName)
	}

	secretsToMove := []corev1.Secret{}
	for _, secret := range secretList.Items {
		if secret.Namespace != releaseNamespace {
			secretsToMove = append(secretsToMove, secret)
		}
	}

	for _, secret := range secretsToMove {
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
		
		newReleaseSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secret.Name,
				Namespace: releaseNamespace,
				Labels:    secret.Labels,
			},
			StringData: map[string]string{
				"release": releaseStr,
			},
		}

		_, err = clientset.CoreV1().Secrets(releaseNamespace).Create(context.TODO(), &newReleaseSecret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to create secret %s/%s", releaseNamespace, secret.Name)
		}

		err = clientset.CoreV1().Secrets(secret.Namespace).Delete(context.TODO(), secret.Name, metav1.DeleteOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to delete secret %s/%s", secret.Namespace, secret.Name)
		}
	}

	return nil
}

func encodeRelease(rls *release.Release) (string, error) {
	b, err := json.Marshal(rls)
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
