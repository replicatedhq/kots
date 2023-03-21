package helm

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"helm.sh/helm/v3/pkg/release"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

func MigrateExistingHelmReleaseSecrets(clientset kubernetes.Interface, releaseName string, releaseNS string) error {
	selectorLabels := map[string]string{
		"owner": "helm",
		"name":  releaseName,
	}
	listOpts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	}

	// list all helm releases secrets for given release name
	secretList, err := clientset.CoreV1().Secrets("").List(context.TODO(), listOpts)
	if err != nil {
		return errors.Wrapf(err, "failed to list release secrets for %s", releaseName)
	}

	logger.Debugf("found %d secrets for release %s", len(secretList.Items), releaseName)

	secretsToMove := []corev1.Secret{}
	for _, secret := range secretList.Items {
		if secret.Namespace != releaseNS {
			secretsToMove = append(secretsToMove, secret)
		}
	}

	logger.Debugf("found %d secrets to move for release %s", len(secretsToMove), releaseName)

	for _, secret := range secretsToMove {
		logger.Debugf("moving secret %s/%s to %s", secret.Namespace, secret.Name, releaseNS)
		release, err := HelmReleaseFromSecretData(secret.Data["release"])
		if err != nil {
			return errors.Wrapf(err, "failed to get release from secret data")
		}

		// set release namespace to the new namespace
		release.Namespace = releaseNS
		releaseStr, err := encodeRelease(release)
		if err != nil {
			return errors.Wrapf(err, "failed to encode release")
		}
		// create the new secret
		newReleaseSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secret.Name,
				Namespace: releaseNS,
				Labels:    secret.Labels,
			},
			StringData: map[string]string{
				"release": releaseStr,
			},
		}

		// create the new secret
		logger.Debugf("creating secret %s/%s", releaseNS, secret.Name)
		_, err = clientset.CoreV1().Secrets(releaseNS).Create(context.TODO(), &newReleaseSecret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to create secret %s/%s", releaseNS, secret.Name)
		}

		logger.Debugf("deleting secret %s/%s", secret.Namespace, secret.Name)
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
