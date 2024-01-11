package secrets

import (
	"context"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"

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

func findPathsWithSecrets(archiveDir string) ([]string, error) {
	var paths []string
	decode := scheme.Codecs.UniversalDeserializer().Decode

	err := filepath.Walk(archiveDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		multiDocYaml := util.ConvertToSingleDocs(contents)
		for _, doc := range multiDocYaml {
			_, gvk, err := decode(doc, nil, nil)
			if err != nil {
				// not a yaml file
				continue
			}
			if gvk.Group != "" || gvk.Version != "v1" || gvk.Kind != "Secret" {
				continue
			}
			paths = append(paths, path)
			break
		}

		return nil
	})

	if err != nil {
		return nil, errors.Wrap(err, "could not walk through the archive directory")
	}

	return paths, nil
}
