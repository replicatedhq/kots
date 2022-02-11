package kotsadm

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	kotsadmobjects "github.com/replicatedhq/kots/pkg/kotsadm/objects"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

func getPostgresYAML(deployOptions types.DeployOptions) (map[string][]byte, error) {
	docs := map[string][]byte{}
	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	if deployOptions.PostgresPassword == "" {
		deployOptions.PostgresPassword = uuid.New().String()
	}

	if !deployOptions.IsOpenShift {
		var configmap bytes.Buffer
		if err := s.Encode(kotsadmobjects.PostgresConfigMap(deployOptions), &configmap); err != nil {
			return nil, errors.Wrap(err, "failed to marshal postgres configmap")
		}
		docs["postgres-configmap.yaml"] = configmap.Bytes()
	}

	size, err := getSize(deployOptions, "postgres", resource.MustParse("1Gi"))
	if err != nil {
		return nil, errors.Wrap(err, "failed to get size")
	}
	postgresSts, err := kotsadmobjects.PostgresStatefulset(deployOptions, size)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get postgres statefulset definition")
	}
	var statefulset bytes.Buffer
	if err := s.Encode(postgresSts, &statefulset); err != nil {
		return nil, errors.Wrap(err, "failed to marshal postgres statefulset")
	}
	docs["postgres-statefulset.yaml"] = statefulset.Bytes()

	var service bytes.Buffer
	if err := s.Encode(kotsadmobjects.PostgresService(deployOptions.Namespace), &service); err != nil {
		return nil, errors.Wrap(err, "failed to marshal postgres service")
	}
	docs["postgres-service.yaml"] = service.Bytes()

	return docs, nil
}

func ensurePostgres(deployOptions types.DeployOptions, clientset *kubernetes.Clientset) error {
	if err := ensurePostgresSecret(deployOptions, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure postgres secret")
	}

	if !deployOptions.IsOpenShift {
		if err := ensurePostgresConfigMap(deployOptions, clientset); err != nil {
			return errors.Wrap(err, "failed to ensure postgres configmap")
		}
	}

	size, err := getSize(deployOptions, "postgres", resource.MustParse("1Gi"))
	if err != nil {
		return errors.Wrap(err, "failed to get size")
	}

	if err := ensurePostgresStatefulset(deployOptions, clientset, size); err != nil {
		return errors.Wrap(err, "failed to ensure postgres statefulset")
	}

	if err := ensurePostgresService(deployOptions.Namespace, clientset); err != nil {
		return errors.Wrap(err, "failed to ensure postgres service")
	}

	return nil
}

func ensurePostgresStatefulset(deployOptions types.DeployOptions, clientset *kubernetes.Clientset, size resource.Quantity) error {
	desiredPostgres, err := kotsadmobjects.PostgresStatefulset(deployOptions, size)
	if err != nil {
		return errors.Wrap(err, "failed to get desired postgres statefulset definition")
	}

	ctx := context.TODO()
	existingPostgres, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Get(ctx, "kotsadm-postgres", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing statefulset")
		}

		_, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Create(ctx, desiredPostgres, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create postgres statefulset")
		}

		return nil
	}

	if len(existingPostgres.Spec.Template.Spec.Containers) != 1 || len(desiredPostgres.Spec.Template.Spec.Containers) != 1 {
		return errors.New("postgres stateful set cannot be upgraded")
	}

	existingPostgres.Spec.Template.Spec.Volumes = desiredPostgres.Spec.Template.Spec.DeepCopy().Volumes
	existingPostgres.Spec.Template.Spec.Containers[0].Image = desiredPostgres.Spec.Template.Spec.Containers[0].Image
	existingPostgres.Spec.Template.Spec.Containers[0].VolumeMounts = desiredPostgres.Spec.Template.Spec.Containers[0].DeepCopy().VolumeMounts

	_, err = clientset.AppsV1().StatefulSets(deployOptions.Namespace).Update(ctx, existingPostgres, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update postgres statefulset")
	}

	return nil
}

func ensurePostgresService(namespace string, clientset *kubernetes.Clientset) error {
	_, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), "kotsadm-postgres", metav1.GetOptions{})
	if err != nil {
		if !kuberneteserrors.IsNotFound(err) {
			return errors.Wrap(err, "failed to get existing service")
		}

		_, err := clientset.CoreV1().Services(namespace).Create(context.TODO(), kotsadmobjects.PostgresService(namespace), metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "Failed to create service")
		}
	}

	return nil
}

func waitForHealthyStatefulSet(name string, deployOptions types.DeployOptions, clientset *kubernetes.Clientset, log *logger.CLILogger) error {
	log.ChildActionWithSpinner("Waiting for datastore to be ready")
	defer log.FinishChildSpinner()

	start := time.Now()
	for {
		s, err := clientset.AppsV1().StatefulSets(deployOptions.Namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to list pods")
		}

		if s.Status.ReadyReplicas == *s.Spec.Replicas && s.Status.UpdateRevision == s.Status.CurrentRevision {
			return nil
		}

		time.Sleep(time.Second)

		if time.Now().Sub(start) > time.Duration(deployOptions.Timeout) {
			return &types.ErrorTimeout{Message: fmt.Sprintf("timeout waiting for %s pod", name)}
		}
	}
}
