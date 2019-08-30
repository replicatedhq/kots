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

const (
	kotsadmURL = "https://gist.githubusercontent.com/marccampbell/fb4b367d66beeddb5b4258a82704f75f/raw/e44b48579f34375e8608fd0d0bee350dfb76e7af/kotsadm.yaml"
)

var (
	postgresPassword       = uuid.New().String()
	minioAccessKey         = uuid.New().String()
	minioSecret            = uuid.New().String()
	autoCreateClusterToken = uuid.New().String()
)

type DeployOptions struct {
	Namespace            string
	Kubeconfig           string
	IncludeShip          bool
	IncludeGitHub        bool
	SharedPassword       string
	SharedPasswordBcrypt string
	S3AccessKey          string
	S3SecretKey          string
	JWT                  string
	PostgresPassword     string
	ServiceType          string
	NodePort             int32
	Hostname             string
	ApplicationMetadata  []byte
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

	migrationDocs, err := getMigrationsYAML(deployOptions.Namespace)
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
	apiDocs, err := getApiYAML(deployOptions.Namespace)
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
	operatorDocs, err := getOperatorYAML(deployOptions.Namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get operator yaml")
	}
	for n, v := range operatorDocs {
		docs[n] = v
	}

	return docs, nil
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

	namespace, err := clientset.CoreV1().Namespaces().Get(deployOptions.Namespace, metav1.GetOptions{})
	if kuberneteserrors.IsNotFound(err) {
		namespace = &corev1.Namespace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Namespace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: deployOptions.Namespace,
			},
		}

		log.ChildActionWithSpinner("Creating namespace")
		_, err := clientset.CoreV1().Namespaces().Create(namespace)
		if err != nil {
			return errors.Wrap(err, "failed to create namespace")
		}
		log.FinishChildSpinner()

	} else if err != nil {
		return errors.Wrap(err, "failed to get namespace")
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
