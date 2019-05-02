package shipwatch

import (
	"context"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	shipv1beta1 "github.com/replicatedhq/ship-operator/pkg/apis/ship/v1beta1"
	"github.com/replicatedhq/ship-operator/pkg/logger"
	"github.com/replicatedhq/ship-operator/pkg/reconciler"
	"github.com/replicatedhq/ship-operator/pkg/ship"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// Add creates a new ShipWatch Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
// USER ACTION REQUIRED: update cmd/manager/main.go to call this ship.Add(mgr) to install this Controller
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileShipWatch{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),

		logger: logger.FromEnv(),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("shipwatch-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to ShipWatch
	err = c.Watch(&source.Kind{Type: &shipv1beta1.ShipWatch{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to created Jobs
	err = c.Watch(&source.Kind{Type: &batchv1.Job{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &shipv1beta1.ShipWatch{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to Secrets
	mapFn := handler.ToRequestsFunc(
		func(a handler.MapObject) []reconcile.Request {
			names := ship.GetShipWatchInstanceNamesFromMeta(a.Meta)
			var reqs []reconcile.Request
			for _, name := range names {
				reqs = append(reqs, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      name,
						Namespace: a.Meta.GetNamespace(),
					},
				})
			}
			return reqs
		})
	// only interested in Update events for Secrets labeled with a Shipwatch
	// instance name whose resource version has changed
	isUpdate := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return false
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldShipwatchNames := ship.GetShipWatchInstanceNamesFromMeta(e.MetaOld)
			newShipwatchNames := ship.GetShipWatchInstanceNamesFromMeta(e.MetaNew)

			if len(oldShipwatchNames) == 0 && len(newShipwatchNames) == 0 {
				return false
			}
			if e.MetaOld.GetResourceVersion() == e.MetaNew.GetResourceVersion() {
				return false
			}

			return true
		},
		GenericFunc: func(e event.GenericEvent) bool {
			names := ship.GetShipWatchInstanceNamesFromMeta(e.Meta)
			return len(names) > 0
		},
	}
	err = c.Watch(
		&source.Kind{Type: &corev1.Secret{}},
		&handler.EnqueueRequestsFromMapFunc{
			ToRequests: mapFn,
		},
		isUpdate)
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileShipWatch{}

// ReconcileShipWatch reconciles a ShipWatch object
type ReconcileShipWatch struct {
	client.Client
	scheme *runtime.Scheme

	logger log.Logger
}

// Reconcile reads that state of the cluster for a ShipWatch object and makes changes based on the state read
// and what is in the ShipWatch.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Jobs and Secrets
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=secrets;pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=ship.replicated.com,resources=shipwatches,verbs=get;list;watch;create;update;patch;delete
func (r *ReconcileShipWatch) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	debug := level.Debug(log.With(r.logger, "method", "Reconcile"))
	errLogger := level.Error(log.With(r.logger, "method", "Reconcile"))

	debug.Log("event", "reconcileShipWatch", "name", request.Name)

	ctx := context.TODO()

	// Fetch the instance, watch job, update job, and secret, ignoring NotFound errors
	instance := &shipv1beta1.ShipWatch{}
	err := r.Get(ctx, request.NamespacedName, instance)
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			instance = nil
		} else {
			errLogger.Log("step", "get.instance", "error", err)
			return reconcile.Result{Requeue: true}, err
		}
	}
	watchJob := &batchv1.Job{}
	watchJobNN := types.NamespacedName{
		Name:      request.NamespacedName.Name + "-watch",
		Namespace: request.NamespacedName.Namespace,
	}
	err = r.Get(ctx, watchJobNN, watchJob)
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			watchJob = nil
		} else {
			errLogger.Log("step", "get.watchJob", "error", err)
			return reconcile.Result{Requeue: true}, err
		}
	}
	updateJob := &batchv1.Job{}
	updateJobNN := types.NamespacedName{
		Name:      request.NamespacedName.Name + "-update",
		Namespace: request.NamespacedName.Namespace,
	}
	err = r.Get(ctx, updateJobNN, updateJob)
	if err != nil {
		if kuberneteserrors.IsNotFound(err) {
			updateJob = nil
		} else {
			errLogger.Log("step", "get.updateJob", "error", err)
			return reconcile.Result{Requeue: true}, err
		}
	}
	secret, err := r.getShipWatchSecret(ctx, request.NamespacedName.Name, request.NamespacedName.Namespace)
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		errLogger.Log("step", "get.secret", "error", err)
		return reconcile.Result{Requeue: true}, err
	}

	rec := reconciler.New(
		r.Client,
		r.logger,
		request.NamespacedName.Name,
		request.NamespacedName.Namespace,
		instance,
		watchJob,
		updateJob,
		secret,
	)

	reconcilers := rec.Reconcilers()
	for _, fn := range reconcilers {
		if err := fn(); err != nil {
			errLogger.Log("error", err)
			return reconcile.Result{Requeue: true}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileShipWatch) getShipWatchSecret(ctx context.Context, shipwatchName string, ns string) (*corev1.Secret, error) {
	opts := &client.ListOptions{
		Namespace: ns,
	}
	secrets := &corev1.SecretList{}
	err := r.List(ctx, opts, secrets)
	if err != nil {
		level.Error(log.With(r.logger)).Log("method", "getShipWatchSecret", "error", err)
		return nil, err
	}
	for _, secret := range secrets.Items {
		if ship.HasSecretMeta(&secret, shipwatchName) {
			return &secret, nil
		}
	}

	return nil, nil
}
