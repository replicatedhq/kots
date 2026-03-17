package helm

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	relv1 "helm.sh/helm/v4/pkg/release/v1"
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
		err := moveHelmReleaseSecret(clientset, secret, releaseNamespace, kotsadmNamespace)
		if err != nil {
			return errors.Wrapf(err, "failed to move helm release secret %s to %s", secret.Name, releaseNamespace)
		}
	}
	return nil
}

// moveHelmReleaseSecret will create a new secret in the releaseNamespace and delete the old one from the kotsadmNamespace
func moveHelmReleaseSecret(clientset kubernetes.Interface, secret corev1.Secret, releaseNamespace string, kotsadmNamespace string) error {
	release, err := helmReleaseFromSecretData(secret.Data["release"])
	if err != nil {
		return errors.Wrapf(err, "failed to get release from secret data")
	}

	// set release namespace to the new namespace
	release.Namespace = releaseNamespace
	releaseStr, err := encodeRelease(release)
	if err != nil {
		return errors.Wrapf(err, "failed to encode release")
	}

	secret.ResourceVersion = ""
	secret.Namespace = releaseNamespace
	secret.Data["release"] = []byte(releaseStr)

	_, err = clientset.CoreV1().Secrets(releaseNamespace).Create(context.TODO(), &secret, metav1.CreateOptions{})
	// if the secret already exists in releaseNamespace, we can ignore the error
	if err != nil && !kuberneteserrors.IsAlreadyExists(err) {
		return errors.Wrapf(err, "failed to create secret %s/%s", releaseNamespace, secret.Name)
	}

	err = clientset.CoreV1().Secrets(kotsadmNamespace).Delete(context.TODO(), secret.Name, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to delete secret %s/%s", kotsadmNamespace, secret.Name)
	}

	return nil
}

func encodeRelease(helmRelease *relv1.Release) (string, error) {
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

func helmReleaseFromSecretData(data []byte) (*relv1.Release, error) {
	base64Reader := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(data))
	gzreader, err := gzip.NewReader(base64Reader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gzip reader")
	}
	defer gzreader.Close()

	releaseData, err := io.ReadAll(gzreader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read from gzip reader")
	}

	release := &relv1.Release{}
	err = json.Unmarshal(releaseData, &release)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal release data")
	}

	return release, nil
}
