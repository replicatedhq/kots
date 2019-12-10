package kotsadm

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
)

type DeployOptions struct {
	Namespace              string
	Kubeconfig             string
	Context                string
	IncludeShip            bool
	IncludeGitHub          bool
	SharedPassword         string
	SharedPasswordBcrypt   string
	S3AccessKey            string
	S3SecretKey            string
	JWT                    string
	PostgresPassword       string
	APIEncryptionKey       string
	AutoCreateClusterToken string
	ServiceType            string
	NodePort               int32
	Hostname               string
	ApplicationMetadata    []byte
}

type UpgradeOptions struct {
	Namespace  string
	Kubeconfig string
}

// YAML will return a map containing the YAML needed to run the admin console
func YAML(deployOptions DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}

	if deployOptions.ApplicationMetadata != nil {
		metadataDocs, err := getApplicationMetadataYAML(deployOptions.ApplicationMetadata, deployOptions.Namespace)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get application metadata yaml")
		}
		for n, v := range metadataDocs {
			docs[n] = v
		}
	}

	minioDocs, err := getMinioYAML(deployOptions.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get minio yaml")
	}
	for n, v := range minioDocs {
		docs[n] = v
	}

	postgresDocs, err := getPostgresYAML(deployOptions.Namespace, deployOptions.PostgresPassword)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get postgres yaml")
	}
	for n, v := range postgresDocs {
		docs[n] = v
	}

	migrationDocs, err := getMigrationsYAML(deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get migrations yaml")
	}
	for n, v := range migrationDocs {
		docs[n] = v
	}

	// secrets
	secretsDocs, err := getSecretsYAML(&deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get secrets yaml")
	}
	for n, v := range secretsDocs {
		docs[n] = v
	}

	// api
	apiDocs, err := getApiYAML(deployOptions.Namespace, deployOptions.AutoCreateClusterToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get api yaml")
	}
	for n, v := range apiDocs {
		docs[n] = v
	}

	// web
	webDocs, err := getWebYAML(deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get web yaml")
	}
	for n, v := range webDocs {
		docs[n] = v
	}

	// operator
	operatorDocs, err := getOperatorYAML(deployOptions.Namespace, deployOptions.AutoCreateClusterToken)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get operator yaml")
	}
	for n, v := range operatorDocs {
		docs[n] = v
	}

	return docs, nil
}

func Upgrade(upgradeOptions UpgradeOptions) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes clientset")
	}

	log := logger.NewLogger()

	_, err = clientset.CoreV1().Namespaces().Get(upgradeOptions.Namespace, metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err := errors.New("The namespace cannot be found or accessed")
		log.Error(err)
		return err
	}

	deployOptions, err := readDeployOptionsFromCluster(upgradeOptions.Namespace, upgradeOptions.Kubeconfig, clientset)
	if err != nil {
		return errors.Wrap(err, "failed to read deploy options")
	}

	if err := ensureKotsadm(*deployOptions, clientset, log); err != nil {
		return errors.Wrap(err, "failed to uppgrade admin console")
	}

	return nil
}

func Deploy(deployOptions DeployOptions) error {
	cfg, err := config.GetConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "failed to create kubernetes clientset")
	}

	log := logger.NewLogger()

	namespace := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: deployOptions.Namespace,
		},
	}

	log.ChildActionWithSpinner("Creating namespace")
	_, err = clientset.CoreV1().Namespaces().Create(namespace)
	if err != nil && !kuberneteserrors.IsAlreadyExists(err) {
		// Can't create namespace, but this might be a role restriction and namespace might already exist.
		_, err := clientset.CoreV1().Pods(deployOptions.Namespace).List(metav1.ListOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to verify access to namespace")
		}
	}
	log.FinishChildSpinner()

	if deployOptions.AutoCreateClusterToken == "" {
		deployOptions.AutoCreateClusterToken = uuid.New().String()
	}

	if err := ensureKotsadm(deployOptions, clientset, log); err != nil {
		return errors.Wrap(err, "failed to deploy admin console")
	}

	return nil
}

func ensureKotsadm(deployOptions DeployOptions, clientset *kubernetes.Clientset, log *logger.Logger) error {
	if err := ensureMinio(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure minio")
	}

	if err := ensurePostgres(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure postgres")
	}

	if err := runSchemaHeroMigrations(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to run database migrations")
	}

	if err := ensureSecrets(&deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure secrets exist")
	}

	if err := ensureAPI(&deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api exists")
	}

	log.ChildActionWithSpinner("Waiting for Admin Console to be ready")
	if err := waitForAPI(&deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to wait for API")
	}
	log.FinishSpinner()

	if err := ensureWeb(&deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure web exists")
	}

	if err := ensureOperator(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator")
	}

	return nil
}

func readDeployOptionsFromCluster(namespace string, kubeconfig string, clientset *kubernetes.Clientset) (*DeployOptions, error) {
	deployOptions := DeployOptions{
		Namespace:     namespace,
		Kubeconfig:    kubeconfig,
		IncludeShip:   false,
		IncludeGitHub: false,
		ServiceType:   "ClusterIP",
		Hostname:      "localhost:8800",
	}

	// Shared password, we can't read the original, but we can check if there's a bcrypted value
	// the caller should not recreate if there is a password bcrypt on the return value
	sharedPasswordSecret, err := getSharedPasswordSecret(namespace, clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get shared password secret")
	}
	if sharedPasswordSecret != nil {
		data, ok := sharedPasswordSecret.Data["passwordBcrypt"]
		if ok {
			deployOptions.SharedPasswordBcrypt = string(data)
		}
	}
	if deployOptions.SharedPasswordBcrypt == "" {
		sharedPassword, err := promptForSharedPassword()
		if err != nil {
			return nil, errors.Wrap(err, "failed to prompt for shared password")
		}

		deployOptions.SharedPassword = sharedPassword
	}

	// s3 secret, get from cluster or create new random values
	s3Secret, err := getS3Secret(namespace, clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get s3 secret")
	}
	if s3Secret != nil {
		accessKey, ok := s3Secret.Data["accesskey"]
		if ok {
			deployOptions.S3AccessKey = string(accessKey)
		}

		secretyKey, ok := s3Secret.Data["secretkey"]
		if ok {
			deployOptions.S3SecretKey = string(secretyKey)
		}
	}
	if deployOptions.S3AccessKey == "" {
		deployOptions.S3AccessKey = uuid.New().String()
	}
	if deployOptions.S3SecretKey == "" {
		deployOptions.S3SecretKey = uuid.New().String()
	}

	// jwt key, get or create new value
	jwtSecret, err := getJWTSessionSecret(namespace, clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get jwt secret")
	}
	if jwtSecret != nil {
		sessionKey, ok := jwtSecret.Data["key"]
		if ok {
			deployOptions.JWT = string(sessionKey)
		}
	}

	// postgres password, read from the secret or create new password
	pgSecret, err := getPostgresSecret(namespace, clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get postgres secret")
	}
	if pgSecret != nil {
		password, ok := pgSecret.Data["password"]
		if ok {
			deployOptions.PostgresPassword = string(password)
		}
	}

	// API encryption key, read from the secret or create new password
	encyptionSecret, err := getAPIEncryptionSecret(namespace, clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get postgres secret")
	}
	if encyptionSecret != nil {
		key, ok := encyptionSecret.Data["encryptionKey"]
		if ok {
			deployOptions.APIEncryptionKey = string(key)
		}
	}

	return &deployOptions, nil
}
