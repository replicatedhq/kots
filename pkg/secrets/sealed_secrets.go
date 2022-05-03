package secrets

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	sealedsecretsv1alpha1 "github.com/bitnami-labs/sealed-secrets/pkg/apis/sealed-secrets/v1alpha1"
	sealedsecretsscheme "github.com/bitnami-labs/sealed-secrets/pkg/client/clientset/versioned/scheme"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
	"io/ioutil"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	jsonserializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
	"os"
)

func replaceSecretsWithSealedSecrets(archivePath string, config map[string][]byte) error {
	secretPaths, err := getSecretsInPath(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to get secrets in path")
	}

	if len(secretPaths) == 0 {
		return nil
	}

	block, _ := pem.Decode(config["cert.pem"])
	if block == nil {
		return errors.New("unable to read public key from secret")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return errors.Wrap(err, "failed to parse certificate")
	}

	sealedsecretsscheme.AddToScheme(scheme.Scheme)

	decode := scheme.Codecs.UniversalDeserializer().Decode
	for _, secretPath := range secretPaths {
		contents, err := ioutil.ReadFile(secretPath)
		if err != nil {
			return errors.Wrap(err, "failed to read file")
		}

		multiDocYaml := bytes.Split(contents, []byte("---\n"))
		var secrets []byte
		var nonSecrets []byte
		for i := 0; i < len(multiDocYaml); i++ {
			object := multiDocYaml[i]
			if string(object) == "" {
				continue
			}

			decoded, _, err := decode(object, nil, nil)
			if err != nil {
				nonSecrets = append(nonSecrets, []byte("---\n")...)
				nonSecrets = append(nonSecrets, multiDocYaml[i]...)
				continue
			}

			secret, ok := decoded.(*v1.Secret)
			if !ok {
				nonSecrets = append(nonSecrets, []byte("---\n")...)
				nonSecrets = append(nonSecrets, multiDocYaml[i]...)
				continue
			}

			secretBytes, err := createSecret(cert, secret)
			if err != nil {
				return err
			}
			secrets = append(secrets, []byte("---\n")...)
			secrets = append(secrets, secretBytes...)
		}

		fileContents := append(secrets, nonSecrets...)
		if err := ioutil.WriteFile(secretPath, fileContents, 0644); err != nil {
			return errors.Wrap(err, "failed to write sealed secret")
		}
	}

	return nil
}

func createSecret(cert *x509.Certificate, secret *v1.Secret) ([]byte, error) {
	codecFactory := serializer.NewCodecFactory(scheme.Scheme)

	// sealed secrets require a namespace
	if secret.Namespace == "" {
		if os.Getenv("DEV_NAMESPACE") != "" {
			secret.Namespace = os.Getenv("DEV_NAMESPACE")
		} else {
			secret.Namespace = util.PodNamespace
		}
	}

	sealedSecret, err := sealedsecretsv1alpha1.NewSealedSecret(codecFactory, cert.PublicKey.(*rsa.PublicKey), secret)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create sealedsecret")
	}

	sealedSecret.APIVersion = "bitnami.com/v1alpha1"
	sealedSecret.Kind = "SealedSecret"

	s := jsonserializer.NewYAMLSerializer(jsonserializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(sealedSecret, &b); err != nil {
		return nil, errors.Wrap(err, "failed to serialized sealedsecret")
	}

	return b.Bytes(), nil
}
