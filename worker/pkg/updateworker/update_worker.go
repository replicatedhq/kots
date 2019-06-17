package updateworker

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/store"
	"github.com/replicatedhq/ship-cluster/worker/pkg/version"
	"k8s.io/client-go/kubernetes"
)

type Worker struct {
	Config *config.Config
	Logger log.Logger

	Store     store.Store
	K8sClient kubernetes.Interface
}

func (w *Worker) Run(ctx context.Context) error {
	logger := log.With(w.Logger, "method", "updateworker.Worker.Execute")

	level.Info(logger).Log("phase", "initialize",
		"version", version.Version(),
		"gitSHA", version.GitSHA(),
		"buildTime", version.BuildTime(),
		"buildTimeFallback", version.GetBuild().TimeFallback,
	)

	errCh := make(chan error, 3)

	go func() {
		level.Info(logger).Log("event", "db.poller.ready.start")
		err := w.startPollingDBForReadyUpdates(context.Background())
		level.Info(logger).Log("event", "db.poller.ready.fail", "err", err)
		errCh <- errors.Wrap(err, "ready poller ended")
	}()

	go func() {
		level.Info(logger).Log("event", "k8s.controller.start")
		err := w.runInformer(context.Background())
		level.Info(logger).Log("event", "k8s.controller.fail", "err", err)
		errCh <- errors.Wrap(err, "controller ended")
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

func (w *Worker) startPollingDBForReadyUpdates(ctx context.Context) error {
	logger := log.With(w.Logger, "method", "watchworker.Worker.startPollingDBForReadyUpdates")

	for {
		select {
		case <-time.After(w.Config.DBPollInterval):
			updateIDs, err := w.Store.ListReadyUpdateIDs(ctx)
			if err != nil {
				level.Error(logger).Log("event", "store.list.ready.updates.fail", "err", err)
				continue
			}

			if err := w.startUpdates(ctx, updateIDs); err != nil {
				level.Error(logger).Log("event", "ensure.update.running.fail", "err", err)
				continue
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (w *Worker) startUpdates(ctx context.Context, updateIDs []string) error {
	logger := log.With(w.Logger, "method", "updateWorker.Worker.startUpdates")

	for _, updateID := range updateIDs {
		if err := w.startUpdate(ctx, updateID); err != nil {
			level.Error(logger).Log("event", "startUpdate.fail", "err", err)
		}
	}

	return nil
}

func (w *Worker) startUpdate(ctx context.Context, updateID string) error {
	if err := w.Store.SetUpdateStarted(ctx, updateID); err != nil {
		return err
	}

	shipUpdate, err := w.Store.GetUpdate(context.TODO(), updateID)
	if err != nil {
		level.Error(w.Logger).Log("getUpdate", err)
		return err
	}

	if err := w.deployUpdate(shipUpdate); err != nil {
		level.Error(w.Logger).Log("deployUpdate", err)
		return err
	}

	return nil
}
