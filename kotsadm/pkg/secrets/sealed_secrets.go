package secrets

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"io/ioutil"
	"os"

	sealedsecretsv1alpha1 "github.com/bitnami-labs/sealed-secrets/pkg/apis/sealed-secrets/v1alpha1"
	sealedsecretsscheme "github.com/bitnami-labs/sealed-secrets/pkg/client/clientset/versioned/scheme"
	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	jsonserializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

func replaceSecretsWithSealedSecrets(archivePath string, config map[string][]byte) error {
	secretPaths, err := getSecretsInPath(archivePath)
	if err != nil {
		return errors.Wrap(err, "failed to get secrets in path")
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
	codecFactory := serializer.NewCodecFactory(scheme.Scheme)

	decode := scheme.Codecs.UniversalDeserializer().Decode
	for _, secretPath := range secretPaths {
		contents, err := ioutil.ReadFile(secretPath)
		if err != nil {
			return errors.Wrap(err, "failed to read file")
		}

		decoded, _, err := decode(contents, nil, nil)
		if err != nil {
			return nil
		}

		secret, ok := decoded.(*v1.Secret)
		if !ok {
			return nil
		}

		// sealed secrets require a namespace
		if secret.Namespace == "" {
			if os.Getenv("DEV_NAMESPACE") != "" {
				secret.Namespace = os.Getenv("DEV_NAMESPACE")
			}

			secret.Namespace = os.Getenv("POD_NAMESPACE")
		}

		sealedSecret, err := sealedsecretsv1alpha1.NewSealedSecret(codecFactory, cert.PublicKey.(*rsa.PublicKey), secret)
		if err != nil {
			return errors.Wrap(err, "failed to create sealedsecret")
		}

		s := jsonserializer.NewYAMLSerializer(jsonserializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

		var b bytes.Buffer
		if err := s.Encode(sealedSecret, &b); err != nil {
			return errors.Wrap(err, "failed to serialized sealedsecret")
		}

		if err := ioutil.WriteFile(secretPath, b.Bytes(), 0644); err != nil {
			return errors.Wrap(err, "failed to write sealed secret")
		}
	}

	return nil
}
