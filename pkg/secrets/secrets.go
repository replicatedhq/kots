package secrets

import (
	"context"
	"io/ioutil"
	"k8s.io/client-go/kubernetes"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
)

func ReplaceSecretsInPath(archiveDir string, clientset kubernetes.Interface) error {
	logger.Debug("checking for secrets replacers")

	secrets, err := clientset.CoreV1().Secrets(util.PodNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: "kots.io/buildphase=secret",
	})
	if err != nil {
		return errors.Wrap(err, "failed to list secrets")
	}

	if len(secrets.Items) == 0 {
		return nil
	}

	if len(secrets.Items) > 1 {
		return errors.New("multiple secret buildphases are not supported")
	}

	secret := secrets.Items[0]
	secretType := secret.Labels["kots.io/secrettype"]

	switch secretType {
	case "sealedsecrets":
		return replaceSecretsWithSealedSecrets(archiveDir, secret.Data)
	default:
		return errors.Errorf("unknown secret type %q", secretType)
	}
}

func getSecretsInPath(archiveDir string) ([]string, error) {
	secretPaths := []string{}
	decode := scheme.Codecs.UniversalDeserializer().Decode

	err := filepath.Walk(archiveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrap(err, "could not walk through the archive directory")
		}

		if info.IsDir() {
			return nil
		}

		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		_, gvk, err := decode(contents, nil, nil)
		if err != nil {
			return nil
		}

		if gvk.Group != "" || gvk.Version != "v1" || gvk.Kind != "Secret" {
			return nil
		}

		secretPaths = append(secretPaths, path)
		return nil
	})

	if err != nil {
		return nil, err
	}

	return secretPaths, nil
}
