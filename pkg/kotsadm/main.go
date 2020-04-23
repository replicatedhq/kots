package kotsadm

import (
	"github.com/google/uuid"
	"github.com/pkg/errors"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
}

// YAML will return a map containing the YAML needed to run the admin console
func YAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
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

	minioDocs, err := getMinioYAML(deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get minio yaml")
	}
	for n, v := range minioDocs {
		docs[n] = v
	}

	postgresDocs, err := getPostgresYAML(deployOptions)
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
	apiDocs, err := getApiYAML(deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get api yaml")
	}
	for n, v := range apiDocs {
		docs[n] = v
	}

	// kotsadm
	kotsadmDocs, err := getKotsadmYAML(deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kotsadm yaml")
	}
	for n, v := range kotsadmDocs {
		docs[n] = v
	}

	// operator
	operatorDocs, err := getOperatorYAML(deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get operator yaml")
	}
	for n, v := range operatorDocs {
		docs[n] = v
	}

	return docs, nil
}

func Upgrade(upgradeOptions types.UpgradeOptions) error {
	clientset, err := k8sutil.GetClientset(upgradeOptions.KubernetesConfigFlags)
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	log := logger.NewLogger()

	if err := canUpgrade(upgradeOptions, clientset, log); err != nil {
		log.Error(err)
		return err
	}

	deployOptions, err := readDeployOptionsFromCluster(upgradeOptions.Namespace, upgradeOptions.KubernetesConfigFlags, clientset)
	if err != nil {
		return errors.Wrap(err, "failed to read deploy options")
	}

	if err := ensureKotsadm(*deployOptions, clientset, log); err != nil {
		return errors.Wrap(err, "failed to upgrade admin console")
	}

	if err := removeUnusedKotsadmComponents(*deployOptions, clientset, log); err != nil {
		return errors.Wrap(err, "failed to removed unused admin console components")
	}

	return nil
}

func Deploy(deployOptions types.DeployOptions) error {
	clientset, err := k8sutil.GetClientset(deployOptions.KubernetesConfigFlags)
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
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

	limitRange, err := maybeGetNamespaceLimitRanges(clientset, deployOptions.Namespace)
	if err != nil {
		return errors.Wrap(err, "failed to get limit ranges for namespace")
	}
	deployOptions.LimitRange = limitRange

	deployOptions.IsOpenShift = isOpenshift(clientset)

	if err := ensureKotsadm(deployOptions, clientset, log); err != nil {
		return errors.Wrap(err, "failed to deploy admin console")
	}

	return nil
}

func canUpgrade(upgradeOptions types.UpgradeOptions, clientset *kubernetes.Clientset, log *logger.Logger) error {
	_, err := clientset.CoreV1().Namespaces().Get(upgradeOptions.Namespace, metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		err := errors.New("The namespace cannot be found or accessed")
		return err
	}

	if upgradeOptions.ForceUpgradeKurl {
		return nil
	}

	// don't upgrade kurl clusters.  kurl only installs into default namespace.
	if upgradeOptions.Namespace != "" && upgradeOptions.Namespace != "default" {
		return nil
	}

	nodes, err := clientset.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to list nodes")
	}

	for _, node := range nodes.Items {
		for k, v := range node.Labels {
			if k == "kurl.sh/cluster" && v == "true" {
				return errors.New("upgrading kURL clusters is not supported")
			}
		}
	}

	return nil
}

func removeUnusedKotsadmComponents(deployOptions types.DeployOptions, clientset *kubernetes.Clientset, log *logger.Logger) error {
	// if there's a kotsadm web deployment, rmove (pre 1.11.0)
	_, err := clientset.AppsV1().Deployments(deployOptions.Namespace).Get("kotsadm-web", metav1.GetOptions{})
	if err == nil {
		if err := clientset.AppsV1().Deployments(deployOptions.Namespace).Delete("kotsadm-web", &metav1.DeleteOptions{}); err != nil {
			return errors.Wrap(err, "failed to delete kotsadm-web deployment")
		}
	}

	// if there's a service named "kotsadm-api", remove (pre 1.11.0)
	_, err = clientset.CoreV1().Services(deployOptions.Namespace).Get("kotsadm-api", metav1.GetOptions{})
	if err == nil {
		if err := clientset.CoreV1().Services(deployOptions.Namespace).Delete("kotsadm-api", &metav1.DeleteOptions{}); err != nil {
			return errors.Wrap(err, "failed to delete kotsadm-api service")
		}
	}

	return nil
}

func ensureKotsadm(deployOptions types.DeployOptions, clientset *kubernetes.Clientset, log *logger.Logger) error {
	// check additional namespaces early in case there are rbac issues we don't
	// leave the cluster in a partially deployed state
	if deployOptions.ApplicationMetadata != nil {
		// If the metadata parses, and if the metadata contains additional namespaces
		// attempt to create
		if err := ensureAdditionalNamespaces(&deployOptions, clientset, log); err != nil {
			return errors.Wrap(err, "failed to ensure additional namespaces")
		}
	}

	if deployOptions.License != nil {
		// if there's a license, we write it as a secret and kotsadm will
		// find it on startup and handle installation
		if err := ensureLicenseSecret(&deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure license s")
		}
	}

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

	if err := ensureKotsadmComponent(&deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure kotsadm exists")
	}

	if err := ensureAPI(&deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure api exists")
	}

	log.ChildActionWithSpinner("Waiting for Admin Console to be ready")
	if err := waitForKotsadm(&deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to wait for API")
	}
	if err := waitForAPI(&deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to wait for API")
	}
	log.FinishSpinner()

	if err := ensureOperator(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure operator")
	}

	return nil
}

func readDeployOptionsFromCluster(namespace string, kubernetesConfigFlags *genericclioptions.ConfigFlags, clientset *kubernetes.Clientset) (*types.DeployOptions, error) {
	deployOptions := types.DeployOptions{
		Namespace:             namespace,
		KubernetesConfigFlags: kubernetesConfigFlags,
		IncludeShip:           false,
		IncludeGitHub:         false,
		ServiceType:           "ClusterIP",
		Hostname:              "localhost:8800",
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

	// AutoCreateClusterToken
	autocreateClusterToken, err := getAPIAutoCreateClusterToken(namespace, clientset)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get auto create cluster token")
	}
	if autocreateClusterToken != "" {
		deployOptions.AutoCreateClusterToken = autocreateClusterToken
	}

	return &deployOptions, nil
}
