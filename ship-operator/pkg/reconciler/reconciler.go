package reconciler

import (
	"github.com/go-kit/kit/log"
	shipv1beta1 "github.com/replicatedhq/ship-operator/pkg/apis/ship/v1beta1"
	"github.com/replicatedhq/ship-operator/pkg/generator"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Reconciler struct {
	logger log.Logger
	client.Client
	instanceName string
	namespace    string
	generator    *generator.Generator
	instance     *shipv1beta1.ShipWatch
	watchJob     *batchv1.Job
	updateJob    *batchv1.Job
	secret       *corev1.Secret
}

func New(
	client client.Client,
	logger log.Logger,
	instanceName string,
	namespace string,
	instance *shipv1beta1.ShipWatch,
	watchJob *batchv1.Job,
	updateJob *batchv1.Job,
	secret *corev1.Secret,
) *Reconciler {
	return &Reconciler{
		generator:    generator.NewGenerator(instance),
		logger:       logger,
		Client:       client,
		instanceName: instanceName,
		namespace:    namespace,
		instance:     instance,
		watchJob:     watchJob,
		updateJob:    updateJob,
		secret:       secret,
	}
}

func (r *Reconciler) Reconcilers() []func() error {
	var reconcilers []func() error

	actions := r.getActions()

	if actions.addSecretMeta {
		reconcilers = append(reconcilers, r.addSecretMeta)
	}
	if actions.removeSecretMeta {
		reconcilers = append(reconcilers, r.removeSecretMeta)
	}
	if actions.deleteWatchJob {
		reconcilers = append(reconcilers, r.deleteWatchJob)
	}
	if actions.deleteUpdateJob {
		reconcilers = append(reconcilers, r.deleteUpdateJob)
	}
	if actions.createWatchJob {
		reconcilers = append(reconcilers, r.createWatchJob)
	}
	if actions.createUpdateJob {
		reconcilers = append(reconcilers, r.createUpdateJob)
	}

	reconcilers = append(reconcilers, r.pruneCompletedPods)

	return reconcilers
}
