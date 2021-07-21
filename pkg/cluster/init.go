package cluster

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"go.uber.org/zap"
)

// clusterInit will ensure that all certs and keys are in well-known locations
func clusterInit(ctx context.Context, dataDir string, slug string, version string) error {
	log := ctx.Value("log").(*logger.CLILogger)

	log.Debug("initing kubernetes whatnot for slug %s, version %s", slug, version)

	// this generates the following files:
	//
	//   <datadir>/kubernetes/service-account-key.pem
	//   <datadir>/kubernetes/service-account-signing-key.pem

	// https://github.com/kelseyhightower/kubernetes-the-hard-way/blob/master/docs/08-bootstrapping-kubernetes-controllers.md

	// ensure that the kubernetes directory exists
	kubernetesDir := filepath.Join(dataDir, "kubernetes")
	if _, err := os.Stat(kubernetesDir); os.IsNotExist(err) {
		log.Info("creating kubernetes cert dir")
		if err := os.Mkdir(kubernetesDir, 0755); err != nil {
			return errors.Wrap(err, "create kubernetes directory")
		}
	}

	if err := ensureCA(dataDir); err != nil {
		return errors.Wrap(err, "ensure ca")
	}
	log.Info("verified ca file")

	if err := ensureApiKeyAndCert(dataDir); err != nil {
		return errors.Wrap(err, "ensure api key and cert")
	}
	log.Info("verified tls key and cert for api")

	serviceAccountSigningKeyFile, err := ensureServiceAccountSigningKeyFile(dataDir)
	if err != nil {
		return errors.Wrap(err, "ensure service account sigining key file")
	}
	logger.Info("verified service account signing key file",
		zap.String("serviceAccountSigningKeyFile", serviceAccountSigningKeyFile))

	serviceAccountKeyFile, err := ensureServiceAccountKeyFile(dataDir)
	if err != nil {
		return errors.Wrap(err, "ensuire service account key file")
	}
	logger.Info("verified service account key file",
		zap.String("serviceAccountKeyFile", serviceAccountKeyFile))

	kubeconfigFile, err := ensureKubeconfigFile(slug, dataDir)
	if err != nil {
		return errors.Wrap(err, "ensure kubeconfig")
	}
	logger.Info("verified kubeconfig file",
		zap.String("kubeconfigFile", kubeconfigFile))

	schedulerConfigFile, err := ensureSchedulerConfigFile(dataDir)
	if err != nil {
		return errors.Wrap(err, "ensure scheduler config file")
	}
	logger.Info("verified scheduler config file",
		zap.String("schedulerConfigFile", schedulerConfigFile))

	authenticationConfigFile, err := ensureAuthenticationConfigFile(dataDir)
	if err != nil {
		return errors.Wrap(err, "ensure authentication config file")
	}
	logger.Info("verified authentication config file",
		zap.String("authenticationConfigFile", authenticationConfigFile))

	authorizationConfigFile, err := ensureAuthorizationConfigFile(dataDir)
	if err != nil {
		return errors.Wrap(err, "ensure authorization config file")
	}
	logger.Info("verified authorization config file",
		zap.String("authorizationConfigFile", authorizationConfigFile))

	return nil
}

func schedulerConfigFilePath(dataDir string) (string, error) {
	kubeconfigFile, err := filepath.Abs(filepath.Join(dataDir, "kubernetes", "kube-scheduler.yaml"))
	if err != nil {
		return "", errors.Wrap(err, "generate filename")
	}

	return kubeconfigFile, nil
}

func ensureSchedulerConfigFile(dataDir string) (string, error) {
	schedulerConfigFile, err := schedulerConfigFilePath(dataDir)
	if err != nil {
		return "", errors.Wrap(err, "get scheduler config file path")
	}

	if _, err := os.Stat(schedulerConfigFile); os.IsNotExist(err) {
		logger.Info("creating new scheduler config file")

		kubeconfigFile, err := kubeconfigFilePath(dataDir)
		if err != nil {
			return "", errors.Wrap(err, "get kubeconfig path")
		}

		b := fmt.Sprintf(`apiVersion: kubescheduler.config.k8s.io/v1beta1
kind: KubeSchedulerConfiguration
healthzBindAddress: "0.0.0.0:11251"
metricsBindAddress: "0.0.0.0:11251"
clientConnection:
  kubeconfig: %q
leaderElection:
  leaderElect: true
`, kubeconfigFile)

		if err := ioutil.WriteFile(schedulerConfigFile, []byte(b), 0600); err != nil {
			return "", errors.Wrap(err, "write file")
		}
	}

	return schedulerConfigFile, nil
}

func kubeconfigFilePath(dataDir string) (string, error) {
	kubeconfigFile, err := filepath.Abs(filepath.Join(dataDir, "kubernetes", "kubeconfig"))
	if err != nil {
		return "", errors.Wrap(err, "generate filename")
	}

	return kubeconfigFile, nil
}

func ensureKubeconfigFile(slug string, dataDir string) (string, error) {
	kubeconfigFile, err := kubeconfigFilePath(dataDir)
	if err != nil {
		return "", errors.Wrap(err, "get kubeconfig file path")
	}

	if _, err := os.Stat(kubeconfigFile); os.IsNotExist(err) {
		logger.Info("creating new kubeconfig file")

		certFile, err := caCertFilePath(dataDir)
		if err != nil {
			return "", errors.Wrap(err, "get cert file path")
		}
		data, err := ioutil.ReadFile(certFile)
		if err != nil {
			return "", errors.Wrap(err, "read cert file")
		}
		encoded := base64.StdEncoding.EncodeToString(data)
		kubeconfigBytes, err := getLocalKubeconfig([]byte(encoded), slug)
		if err != nil {
			return "", errors.Wrap(err, "get local kubeconfig contents")
		}

		if err := ioutil.WriteFile(kubeconfigFile, kubeconfigBytes, 0600); err != nil {
			return "", errors.Wrap(err, "write file")
		}

	}
	return kubeconfigFile, nil
}

func getLocalKubeconfig(cert []byte, localToken string) ([]byte, error) {
	b := fmt.Sprintf(`apiVersion: v1
clusters:
- name: kubernetes
  cluster:
    certificate-authority-data: %s
    server: "https://localhost:8443"

contexts:
- name: local@kubernetes
  context:
    cluster: kubernetes
    user: local

current-context: local@kubernetes
kind: Config
preferences: {}
users:
- name: local
  user:
    token: %s`, cert, localToken)

	return []byte(b), nil
}

func serviceAccountSigningKeyFilePath(dataDir string) (string, error) {
	serviceAccountSigningKeyFile, err := filepath.Abs(filepath.Join(dataDir, "kubernetes", "service-account-signing-key.pem"))
	if err != nil {
		return "", errors.Wrap(err, "generate filename")
	}

	return serviceAccountSigningKeyFile, nil
}

func ensureServiceAccountSigningKeyFile(dataDir string) (string, error) {
	// TOOD should this be in the /storage directory (i.e. a PVC)
	serviceAccountSigningKeyFile, err := serviceAccountSigningKeyFilePath(dataDir)
	if err != nil {
		return "", errors.Wrap(err, "get service account signing key file path")
	}

	if _, err := os.Stat(serviceAccountSigningKeyFile); os.IsNotExist(err) {
		logger.Info("creating new service account signing key")
		pk, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return "", errors.Wrap(err, "generate key")
		}

		var privateKeyBytes []byte = x509.MarshalPKCS1PrivateKey(pk)
		privateKeyBlock := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privateKeyBytes,
		}

		privateBytes := pem.EncodeToMemory(privateKeyBlock)
		if err := ioutil.WriteFile(serviceAccountSigningKeyFile, privateBytes, 0600); err != nil {
			return "", errors.Wrap(err, "write file")
		}
	}

	return serviceAccountSigningKeyFile, nil
}

func serviceAccountKeyFilePath(dataDir string) (string, error) {
	serviceAccountKeyFile, err := filepath.Abs(filepath.Join(dataDir, "kubernetes", "service-account-key.pem"))
	if err != nil {
		return "", errors.Wrap(err, "generate filename")
	}

	return serviceAccountKeyFile, nil
}

func ensureServiceAccountKeyFile(dataDir string) (string, error) {
	serviceAccountKeyFile, err := serviceAccountKeyFilePath(dataDir)
	if err != nil {
		return "", errors.Wrap(err, "generate service account key file path")
	}

	if _, err := os.Stat(serviceAccountKeyFile); os.IsNotExist(err) {
		logger.Info("creating new service account signing key")
		pk, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return "", errors.Wrap(err, "generate key")
		}

		var privateKeyBytes []byte = x509.MarshalPKCS1PrivateKey(pk)
		privateKeyBlock := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: privateKeyBytes,
		}

		privateBytes := pem.EncodeToMemory(privateKeyBlock)
		if err := ioutil.WriteFile(serviceAccountKeyFile, privateBytes, 0600); err != nil {
			return "", errors.Wrap(err, "write file")
		}
	}

	return serviceAccountKeyFile, nil
}

func caCertFilePath(dataDir string) (string, error) {
	filename, err := filepath.Abs(filepath.Join(dataDir, "kubernetes", "ca.crt"))
	if err != nil {
		return "", errors.Wrap(err, "generate filename")
	}

	return filename, nil
}

func caKeyFilePath(dataDir string) (string, error) {
	filename, err := filepath.Abs(filepath.Join(dataDir, "kubernetes", "ca.key"))
	if err != nil {
		return "", errors.Wrap(err, "generate filename")
	}

	return filename, nil
}

func ensureCA(dataDir string) error {
	caKeyFile, err := caKeyFilePath(dataDir)
	if err != nil {
		return errors.Wrap(err, "generate key filename")
	}
	caCertFile, err := caCertFilePath(dataDir)
	if err != nil {
		return errors.Wrap(err, "generate cert filename")
	}

	_, keyFileError := os.Stat(caKeyFile)
	_, certFileError := os.Stat(caKeyFile)

	if os.IsNotExist(keyFileError) || os.IsNotExist(certFileError) {
		logger.Info("creating new ca")
		// ensure both are deleted
		if err := os.RemoveAll(caKeyFile); err != nil {
			return errors.Wrap(err, "delete key")
		}
		if err := os.RemoveAll(caCertFile); err != nil {
			return errors.Wrap(err, "delete cert")
		}

		ca := &x509.Certificate{
			SerialNumber: big.NewInt(2021),
			Subject: pkix.Name{
				Organization:  []string{"Replicated"},
				Country:       []string{"US"},
				Province:      []string{""},
				Locality:      []string{"Los Angeles"},
				StreetAddress: []string{""},
				PostalCode:    []string{""},
			},
			NotBefore:             time.Now(),
			NotAfter:              time.Now().AddDate(10, 0, 0),
			IsCA:                  true,
			ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
			KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
			BasicConstraintsValid: true,
		}

		caPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return errors.Wrap(err, "generate key")
		}

		caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
		if err != nil {
			return errors.Wrap(err, "generate cert")
		}

		caPEM := new(bytes.Buffer)
		pem.Encode(caPEM, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: caBytes,
		})
		if err := ioutil.WriteFile(caCertFile, caPEM.Bytes(), 0600); err != nil {
			return errors.Wrap(err, "write cert")
		}

		caPrivKeyPEM := new(bytes.Buffer)
		pem.Encode(caPrivKeyPEM, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
		})
		if err := ioutil.WriteFile(caKeyFile, caPrivKeyPEM.Bytes(), 0600); err != nil {
			return errors.Wrap(err, "write key")
		}
	}

	return nil
}

func apiCertFilePath(dataDir string) (string, error) {
	filename, err := filepath.Abs(filepath.Join(dataDir, "kubernetes", "apiserver.crt"))
	if err != nil {
		return "", errors.Wrap(err, "generate filename")
	}

	return filename, nil
}

func apiKeyFilePath(dataDir string) (string, error) {
	filename, err := filepath.Abs(filepath.Join(dataDir, "kubernetes", "apiserver.key"))
	if err != nil {
		return "", errors.Wrap(err, "generate filename")
	}

	return filename, nil
}

func ensureApiKeyAndCert(dataDir string) error {
	apiKeyFile, err := apiKeyFilePath(dataDir)
	if err != nil {
		return errors.Wrap(err, "generate key filename")
	}
	apiCertFile, err := apiCertFilePath(dataDir)
	if err != nil {
		return errors.Wrap(err, "generate cert filename")
	}

	_, keyFileError := os.Stat(apiKeyFile)
	_, certFileError := os.Stat(apiCertFile)

	if os.IsNotExist(keyFileError) || os.IsNotExist(certFileError) {
		logger.Info("creating new api cert and key")
		// ensure both are deleted
		if err := os.RemoveAll(apiKeyFile); err != nil {
			return errors.Wrap(err, "delete key")
		}
		if err := os.RemoveAll(apiCertFile); err != nil {
			return errors.Wrap(err, "delete cert")
		}

		// read the ca cert and key
		caKeyFile, err := caKeyFilePath(dataDir)
		if err != nil {
			return errors.Wrap(err, "read ca key path")
		}
		data, err := ioutil.ReadFile(caKeyFile)
		if err != nil {
			return errors.Wrap(err, "read key file")
		}
		privPem, _ := pem.Decode(data)
		if privPem == nil {
			return errors.New("decode ca key")
		}
		var parsedKey interface{}
		if parsedKey, err = x509.ParsePKCS1PrivateKey(privPem.Bytes); err != nil {
			parsedKey, err = x509.ParsePKCS8PrivateKey(privPem.Bytes)
			if err != nil {
				return errors.Wrap(err, "parse ca key")
			}
		}
		caKey, ok := parsedKey.(*rsa.PrivateKey)
		if !ok {
			return errors.New("ca key was not castable")
		}

		caCertFile, err := caCertFilePath(dataDir)
		if err != nil {
			return errors.Wrap(err, "read ca cert path")
		}
		data, err = ioutil.ReadFile(caCertFile)
		if err != nil {
			return errors.Wrap(err, "read ca cert file")
		}
		certPem, _ := pem.Decode(data)
		if certPem == nil {
			return errors.New("decode ca cert")
		}
		caCert, err := x509.ParseCertificate(certPem.Bytes)
		if err != nil {
			return errors.Wrap(err, "parse ca cert")
		}

		cert := &x509.Certificate{
			SerialNumber: big.NewInt(1658),
			Subject: pkix.Name{
				Organization:  []string{"Replicated"},
				Country:       []string{"US"},
				Province:      []string{""},
				Locality:      []string{"Los Angeles"},
				StreetAddress: []string{""},
				PostalCode:    []string{""},
			},
			IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
			NotBefore:    time.Now(),
			NotAfter:     time.Now().AddDate(10, 0, 0),
			SubjectKeyId: []byte{1, 2, 3, 4, 6},
			ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
			KeyUsage:     x509.KeyUsageDigitalSignature,
			DNSNames:     []string{"localhost"},
		}

		certPrivKey, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return errors.Wrap(err, "generate key")
		}

		certBytes, err := x509.CreateCertificate(rand.Reader, cert, caCert, &certPrivKey.PublicKey, caKey)
		if err != nil {
			return err
		}

		certPEM := new(bytes.Buffer)
		pem.Encode(certPEM, &pem.Block{
			Type:  "CERTIFICATE",
			Bytes: certBytes,
		})
		if err := ioutil.WriteFile(apiCertFile, certPEM.Bytes(), 0600); err != nil {
			return errors.Wrap(err, "write cert")
		}

		caPrivKeyPEM := new(bytes.Buffer)
		pem.Encode(caPrivKeyPEM, &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
		})
		if err := ioutil.WriteFile(apiKeyFile, caPrivKeyPEM.Bytes(), 0600); err != nil {
			return errors.Wrap(err, "write key")
		}
	}

	return nil
}

func authenticationConfigFilePath(dataDir string) (string, error) {
	filename, err := filepath.Abs(filepath.Join(dataDir, "kubernetes", "authentication-config.yaml"))
	if err != nil {
		return "", errors.Wrap(err, "generate filename")
	}

	return filename, nil
}

func ensureAuthenticationConfigFile(dataDir string) (string, error) {
	authenticationConfigFile, err := authenticationConfigFilePath(dataDir)
	if err != nil {
		return "", errors.Wrap(err, "generate authentication config filename")
	}

	if _, err := os.Stat(authenticationConfigFile); os.IsNotExist(err) {
		logger.Info("creating new authentication config file")

		b := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
  - name: authn
    cluster:
      server: http://localhost:8880/api/v1/cluster-authn?id=%s
      insecure-skip-tls-verify: true
users:
  - name: kube-apiserver
contexts:
  - context:
      cluster: authn
      user: kube-apiserver
    name: authn
current-context: authn`, "test")

		if err := ioutil.WriteFile(authenticationConfigFile, []byte(b), 0600); err != nil {
			return "", errors.Wrap(err, "write authentication config file")
		}
	}

	return authenticationConfigFile, nil
}

func authorizationConfigFilePath(dataDir string) (string, error) {
	filename, err := filepath.Abs(filepath.Join(dataDir, "kubernetes", "authorizationC-config.yaml"))
	if err != nil {
		return "", errors.Wrap(err, "generate filename")
	}

	return filename, nil
}

func ensureAuthorizationConfigFile(dataDir string) (string, error) {
	authorizationConfigFile, err := authorizationConfigFilePath(dataDir)
	if err != nil {
		return "", errors.Wrap(err, "generate authentication config filename")
	}

	if _, err := os.Stat(authorizationConfigFile); os.IsNotExist(err) {
		logger.Info("creating new authorization config file")

		b := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
  - name: authz
    cluster:
      server: http://localhost:8880/api/v1/cluster-authz?id=%s
      insecure-skip-tls-verify: true
users:
  - name: kube-apiserver
contexts:
  - context:
      cluster: authz
      user: kube-apiserver
    name: authz
current-context: authz`, "test")

		if err := ioutil.WriteFile(authorizationConfigFile, []byte(b), 0600); err != nil {
			return "", errors.Wrap(err, "write authorization config file")
		}
	}

	return authorizationConfigFile, nil
}
