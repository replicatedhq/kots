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
	Namespace      string
	Kubeconfig     string
	IncludeShip    bool
	IncludeGitHub  bool
	SharedPassword string
	ServiceType    string
	NodePort       int32
	Hostname       string
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

	if err := ensureRBAC(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure rbac exists")
	}

	if err := ensureMinio(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure minio")
	}

	if err := ensurePostgres(deployOptions.Namespace, clientset); err != nil {
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

	log.ChildActionWithSpinner("Waiting for kotsadm API to be ready")
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
