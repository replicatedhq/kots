package kotsadm

import (
	"bytes"
	"context"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/crypto"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getSecretsYAML(deployOptions *types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var jwt bytes.Buffer
	if err := s.Encode(kotsadmobjects.JwtSecret(deployOptions.Namespace, deployOptions.JWT), &jwt); err != nil {
		return nil, errors.Wrap(err, "failed to marshal jwt secret")
	}
	docs["secret-jwt.yaml"] = jwt.Bytes()

	var rqliteSecret bytes.Buffer
	if err := s.Encode(kotsadmobjects.RqliteSecret(deployOptions.Namespace, deployOptions.RqlitePassword), &rqliteSecret); err != nil {
		return nil, errors.Wrap(err, "failed to marshal rqlite secret")
	}
	docs["secret-rqlite.yaml"] = rqliteSecret.Bytes()

	if deployOptions.SharedPasswordBcrypt == "" {
		bcryptPassword, err := bcrypt.GenerateFromPassword([]byte(deployOptions.SharedPassword), 10)
		if err != nil {
			return nil, errors.Wrap(err, "failed to bcrypt shared password")
		}
		deployOptions.SharedPasswordBcrypt = string(bcryptPassword)
	}
	var sharedPassword bytes.Buffer
	if err := s.Encode(kotsadmobjects.SharedPasswordSecret(deployOptions.Namespace, deployOptions.SharedPasswordBcrypt), &sharedPassword); err != nil {
		return nil, errors.Wrap(err, "failed to marshal shared password secret")
	}
	docs["secret-shared-password.yaml"] = sharedPassword.Bytes()

	if deployOptions.APIEncryptionKey == "" {
		err := crypto.NewAESCipher()
		if err != nil {
			return nil, errors.Wrap(err, "failed to create new API encryption key")
		}
		deployOptions.APIEncryptionKey = crypto.ToString()
	}
	var apiEncryptionBuffer bytes.Buffer
	if err := s.Encode(kotsadmobjects.ApiEncryptionKeySecret(deployOptions.Namespace, deployOptions.APIEncryptionKey), &apiEncryptionBuffer); err != nil {
		return nil, errors.Wrap(err, "failed to marshal shared password secret")
	}
	docs["secret-api-encryption.yaml"] = apiEncryptionBuffer.Bytes()

	if deployOptions.IncludeMinio {
		var s3 bytes.Buffer
		if deployOptions.S3SecretKey == "" {
			deployOptions.S3SecretKey = uuid.New().String()
		}
		if deployOptions.S3AccessKey == "" {
			deployOptions.S3AccessKey = uuid.New().String()
		}
		if err := s.Encode(kotsadmobjects.S3Secret(deployOptions.Namespace, deployOptions.S3AccessKey, deployOptions.S3SecretKey), &s3); err != nil {
			return nil, errors.Wrap(err, "failed to marshal s3 secret")
		}
		docs["secret-s3.yaml"] = s3.Bytes()
	}

	var tokenSecret bytes.Buffer
	if err := s.Encode(kotsadmobjects.ApiClusterTokenSecret(*deployOptions), &tokenSecret); err != nil {
		return nil, errors.Wrap(err, "failed to marshal api cluster token secret")
	}
	docs["secret-api-cluster-token.yaml"] = tokenSecret.Bytes()

	// this secret is optional
	if secret := kotsadmobjects.PrivateKotsadmRegistrySecret(deployOptions.Namespace, deployOptions.RegistryConfig); secret != nil {
		var registrySecret bytes.Buffer
		if err := s.Encode(secret, &registrySecret); err != nil {
			return nil, errors.Wrap(err, "failed to marshal private kotsadm registry secret")
		}
		docs["private-kotsadm-registry-secret.yaml"] = registrySecret.Bytes()
	}

	return docs, nil
}

func ensureSecrets(deployOptions *types.DeployOptions, clientset kubernetes.Interface) error {
	if err := ensureJWTSessionSecret(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure jwt session secret")
	}

	if err := ensureRqliteSecret(*deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure rqlite secret")
	}

	if deployOptions.SharedPasswordBcrypt == "" {
		if err := ensureSharedPasswordSecret(deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure shared password secret")
		}
	}

	if deployOptions.IncludeMinio {
		if err := ensureS3Secret(deployOptions.Namespace, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure s3 secret")
		}
	}

	if err := ensureAPIEncryptionSecret(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure s3 secret")
	}

	if err := ensureAPIClusterTokenSecret(*deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api cluster token secret")
	}

	return nil
}

func getS3Secret(namespace string, clientset kubernetes.Interface) (*corev1.Secret, error) {
	s3Secret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), "kotsadm-minio", metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to get s3 secret from cluster")
	}

	return s3Secret, nil
}

func ensureS3Secret(namespace string, clientset kubernetes.Interface) error {
	existingS3Secret, err := getS3Secret(namespace, clientset)
	if err != nil {
		return errors.Wrap(err, "failed to check for existing s3 secret")
	}

	if existingS3Secret == nil {
		_, err := clientset.CoreV1().Secrets(namespace).Create(context.TODO(), kotsadmobjects.S3Secret(namespace, uuid.New().String(), uuid.New().String()), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create s3 secret")
		}
	}

	return nil
}

func getJWTSessionSecret(namespace string, clientset kubernetes.Interface) (*corev1.Secret, error) {
	jwtSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), "kotsadm-session", metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to get jwt session secret from cluster")
	}

	return jwtSecret, nil
}

func ensureJWTSessionSecret(namespace string, clientset kubernetes.Interface) error {
	existingJWTSessionSecret, err := getJWTSessionSecret(namespace, clientset)
	if err != nil {
		return errors.Wrap(err, "failed to check for existing jwt sesssion secret")
	}

	if existingJWTSessionSecret == nil {
		_, err := clientset.CoreV1().Secrets(namespace).Create(context.TODO(), kotsadmobjects.JwtSecret(namespace, uuid.New().String()), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create jwt session secret")
		}
	}

	return nil
}

func getRqliteSecret(namespace string, clientset kubernetes.Interface) (*corev1.Secret, error) {
	rqliteSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), "kotsadm-rqlite", metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get rqlite secret from cluster")
	}

	return rqliteSecret, nil
}

func ensureRqliteSecret(deployOptions types.DeployOptions, clientset kubernetes.Interface) error {
	existingRqliteSecret, err := getRqliteSecret(deployOptions.Namespace, clientset)
	if err != nil {
		return errors.Wrap(err, "failed to check for existing rqlite secret")
	}

	if existingRqliteSecret == nil {
		_, err := clientset.CoreV1().Secrets(deployOptions.Namespace).Create(context.TODO(), kotsadmobjects.RqliteSecret(deployOptions.Namespace, deployOptions.RqlitePassword), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create rqlite secret")
		}
	}

	return nil
}

func getSharedPasswordSecret(namespace string, clientset kubernetes.Interface) (*corev1.Secret, error) {
	sharedPasswordSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), "kotsadm-password", metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to get shared password secret from cluster")
	}

	return sharedPasswordSecret, nil
}

func ensureSharedPasswordSecret(deployOptions *types.DeployOptions, clientset kubernetes.Interface) error {
	if deployOptions.SharedPassword == "" {
		sharedPassword, err := util.PromptForNewPassword()
		if err != nil {
			return errors.Wrap(err, "failed to prompt for shared password")
		}

		deployOptions.SharedPassword = sharedPassword
	}

	bcryptPassword, err := bcrypt.GenerateFromPassword([]byte(deployOptions.SharedPassword), 10)
	if err != nil {
		return errors.Wrap(err, "failed to bcrypt shared password")
	}

	existingSharedPasswordSecret, err := getSharedPasswordSecret(deployOptions.Namespace, clientset)
	if err != nil {
		return errors.Wrap(err, "failed to check for existing password secret")
	}
	if existingSharedPasswordSecret == nil {
		_, err := clientset.CoreV1().Secrets(deployOptions.Namespace).Create(context.TODO(), kotsadmobjects.SharedPasswordSecret(deployOptions.Namespace, string(bcryptPassword)), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create password secret")
		}
	}

	// TODO handle update

	return nil
}

func ensureAPIEncryptionSecret(deployOptions *types.DeployOptions, clientset kubernetes.Interface) error {
	secret, err := getAPIEncryptionSecret(deployOptions.Namespace, clientset)
	if err != nil {
		return errors.Wrap(err, "failed to check for existing api encryption secret")
	}

	if secret != nil {
		if key, _ := secret.Data["encryptionKey"]; len(key) > 0 {
			return nil
		}
	}

	if deployOptions.APIEncryptionKey == "" {
		err = crypto.NewAESCipher()
		if err != nil {
			return errors.Wrap(err, "failed to create new AES cipher")
		}
		deployOptions.APIEncryptionKey = crypto.ToString()
	}

	_, err = clientset.CoreV1().Secrets(deployOptions.Namespace).Create(context.TODO(), kotsadmobjects.ApiEncryptionKeySecret(deployOptions.Namespace, deployOptions.APIEncryptionKey), metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create API encryption secret")
	}

	return nil
}

func getAPIEncryptionSecret(namespace string, clientset kubernetes.Interface) (*corev1.Secret, error) {
	apiSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), "kotsadm-encryption", metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "failed to get api encryption secret from cluster")
	}

	return apiSecret, nil
}

func ensureAPIClusterTokenSecret(deployOptions types.DeployOptions, clientset kubernetes.Interface) error {
	_, err := clientset.CoreV1().Secrets(deployOptions.Namespace).Get(context.TODO(), types.ClusterTokenSecret, metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing cluster token secret")
		}

		_, err := clientset.CoreV1().Secrets(deployOptions.Namespace).Create(context.TODO(), kotsadmobjects.ApiClusterTokenSecret(deployOptions), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create cluster token secret")
		}
	}

	// We have never changed the api cluster token secret. We created it in 1.12.x

	return nil
}

func getAPIClusterToken(namespace string, cli kubernetes.Interface) (string, error) {
	apiSecret, err := cli.CoreV1().Secrets(namespace).Get(context.TODO(), types.ClusterTokenSecret, metav1.GetOptions{})
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			return "", nil
		}

		return "", errors.Wrap(err, "failed to get api cluster token secret from cluster")
	}

	tokenBytes, ok := apiSecret.Data[types.ClusterTokenSecret]
	if !ok {
		return "", nil
	}

	return string(tokenBytes), nil
}
