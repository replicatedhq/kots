package watchworker

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/replicatedhq/ship-cluster/worker/pkg/email"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	multierror "github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/store"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
	"github.com/replicatedhq/ship-cluster/worker/pkg/version"
	shipwatchclientset "github.com/replicatedhq/ship-operator/pkg/client/shipwatchclientset"
	"github.com/spf13/viper"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Worker struct {
	Config *config.Config
	Logger log.Logger

	Store         store.Store
	K8sClient     kubernetes.Interface
	ShipK8sClient shipwatchclientset.Interface
	Mailer        *email.Mailer

	GithubToken string
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

// NewWorker gets an instance using viper to pull config
func NewWorker(
	logger log.Logger,
	config *config.Config,

	store store.Store,
	k8sClient kubernetes.Interface,
	shipK8sClient shipwatchclientset.Interface,
	mailer *email.Mailer,
) (*Worker, error) {
	return &Worker{
		Config:        config,
		Logger:        logger,
		Store:         store,
		K8sClient:     k8sClient,
		ShipK8sClient: shipK8sClient,
		Mailer:        mailer,
	}, nil
}

func (w *Worker) Run(ctx context.Context) error {
	logger := log.With(w.Logger, "method", "watchworker.Worker.Execute")

	level.Info(logger).Log("phase", "initialize",
		"version", version.Version(),
		"gitSHA", version.GitSHA(),
		"buildTime", version.BuildTime(),
		"buildTimeFallback", version.GetBuild().TimeFallback,
	)

	errCh := make(chan error, 2)

	go func() {
		watchServer := WatchServer{
			Logger: logger,
			Viper:  viper.New(),
			Worker: w,
			Store:  w.Store,
			Mailer: w.Mailer,
		}

		watchServer.Serve(ctx, w.Config.WatchServerAddress)
	}()

	go func() {
		level.Info(logger).Log("event", "db.poller.ready.start")
		err := w.startPollingDBForReadyOperators(context.Background())
		level.Info(logger).Log("event", "db.poller.ready.fail", "err", err)
		errCh <- errors.Wrap(err, "ready poller ended")
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	select {
	case <-c:
		// TODO: possibly cleanup
		return nil
	case err := <-errCh:
		return err
	}
}

func (w *Worker) startPollingDBForReadyOperators(ctx context.Context) error {
	logger := log.With(w.Logger, "method", "watchworker.Worker.startPollingDBForReadyOperators")

	for {
		select {
		case <-time.After(w.Config.DBPollInterval):
			watchIDs, err := w.Store.ListReadyWatchIDs(ctx)
			if err != nil {
				level.Error(logger).Log("event", "store.list.ready.watch.operators.fail", "err", err)
				continue
			}

			if err := w.ensureOperatorsRunning(ctx, watchIDs); err != nil {
				level.Error(logger).Log("event", "deploy.operators.fail", "err", err)
				continue
			}

			// Get all watch IDs that are running from kubernetes
			runningWatchIDs, err := w.listRunningWatchIDs(ctx)
			watchIDsToDelete := diffSlices(runningWatchIDs, watchIDs)
			if err := w.ensureOperatorsNotRunning(ctx, watchIDsToDelete); err != nil {
				level.Error(logger).Log("event", "undeploy.operators.fail", "err", err)
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (w *Worker) listRunningWatchIDs(ctx context.Context) ([]string, error) {
	shipwatches, err := w.ShipK8sClient.ShipV1beta1().ShipWatches("").List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "list shipwatches")
	}

	watchIDs := make([]string, 0, 0)
	for _, sw := range shipwatches.Items {
		watchIDs = append(watchIDs, sw.Name)
	}

	return watchIDs, nil
}

func (w *Worker) ensureOperatorsNotRunning(ctx context.Context, watchIDs []string) (result error) {
	for _, watchID := range watchIDs {
		if err := w.ensureOperatorNotRunning(ctx, watchID); err != nil {
			result = multierror.Append(result, errors.Wrapf(err, "watch state %s", watchID))
			continue
		}
	}
	return
}

func (w *Worker) ensureOperatorNotRunning(ctx context.Context, watchID string) error {
	debug := level.Debug(log.With(w.Logger, "method", "watchWorker.Worker.ensureOperatorNotRunning"))

	namespaceName := types.Watch{ID: watchID}.Namespace()

	// Try to select the namespace first, because kubernetes api gets pretty vocal when not found, it thinks
	// there's a rbac problem and log spams us
	if _, err := w.K8sClient.CoreV1().Namespaces().Get(namespaceName, metav1.GetOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			// Debug log, not info...  Again, log spam
			debug.Log("event", "get ns error", "err", err)
		}
		return nil
	}

	// Ensure the namespace is gone, use k8s garbage collection to delete the shipwatch and secret
	if err := w.K8sClient.CoreV1().Namespaces().Delete(namespaceName, &metav1.DeleteOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			return errors.Wrap(err, "delete namespace")
		}
	}

	return nil
}

func (w *Worker) ensureOperatorsRunning(ctx context.Context, watchIDs []string) (result error) {
	for _, watchID := range watchIDs {
		if err := w.ensureOperatorRunning(ctx, watchID); err != nil {
			result = multierror.Append(result, errors.Wrapf(err, "watch state %s", watchID))
			continue
		}
	}
	return
}

func (w *Worker) ensureOperatorRunning(ctx context.Context, watchID string) error {
	// get the watch from the database
	watch, err := w.Store.GetWatch(ctx, watchID)
	if err != nil {
		return errors.Wrap(err, "get watch")
	}

	// build a namespace for the watch and ensure that it is running
	namespace := namespaceForWatch(watch)
	var namespaceCreated bool
	if namespaceCreated, err = w.ensureNamespace(namespace); err != nil {
		return errors.Wrap(err, "ensureNamespace")
	}

	// apply a networkPolicy to the new namespace
	networkPolicy := networkPolicySpec(watch)
	if err := w.ensureNetworkPolicy(networkPolicy); err != nil {
		return errors.Wrap(err, "ensureNetworkPolicy")
	}

	secret, shipwatch := watchToCustomResource(w.Config.ShipImage, w.Config.ShipTag, w.Config.ShipPullPolicy, watch, namespace.Name, w.Config.GithubToken)

	// ensure that the secret exists
	if err := w.ensureSecret(secret); err != nil {
		return errors.Wrap(err, "ensureSecret")
	}

	// ensure that the role and roleBindings exist
	role := roleSpec(watch)
	if err := w.ensureRole(role); err != nil {
		return errors.Wrap(err, "ensureRole")
	}

	roleBinding := roleBindingSpec(watch)
	if err := w.ensureRoleBinding(roleBinding); err != nil {
		return errors.Wrap(err, "ensureRoleBinding")
	}

	// ensure that the shipWatch exists and is up to date
	if err := w.ensureShipwatch(shipwatch); err != nil {
		return errors.Wrap(err, "ensureShipwatch")
	}

	// slow down the thundering herd by ratelimiting creation of new ship watches
	if namespaceCreated {
		time.Sleep(w.Config.WatchCreationInterval)
	}

	return nil
}

func diffSlices(a, b []string) (diff []string) {
	m := make(map[string]struct{})

	for _, item := range b {
		m[item] = struct{}{}
	}

	for _, item := range a {
		if _, ok := m[item]; !ok {
			diff = append(diff, item)
		}
	}

	return
}
