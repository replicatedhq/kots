package updateworker

import (
	"context"
	"os"
	"os/signal"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/store"
	"github.com/replicatedhq/ship-cluster/worker/pkg/version"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

type Worker struct {
	Config *config.Config
	Logger *zap.SugaredLogger

	Store     store.Store
	K8sClient kubernetes.Interface
}

func (w *Worker) Run(ctx context.Context) error {
	w.Logger.Infow("starting updateworker",
		zap.String("version", version.Version()),
		zap.String("gitSHA", version.GitSHA()),
		zap.Time("buildTime", version.BuildTime()),
	)

	errCh := make(chan error, 3)

	go func() {
		updateServer := UpdateServer{
			Logger: w.Logger,
			Viper:  viper.New(),
			Worker: w,
			Store:  w.Store,
		}

		updateServer.Serve(ctx, w.Config.UpdateServerAddress)
	}()

	go func() {
		err := w.startPollingDBForReadyUpdates(context.Background())
		w.Logger.Errorw("udpateworker dbpoller failed", zap.Error(err))
		errCh <- errors.Wrap(err, "ready poller ended")
	}()

	go func() {
		err := w.runInformer(context.Background())
		w.Logger.Errorw("updateworker informer failed", zap.Error(err))
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
	for {
		select {
		case <-time.After(w.Config.DBPollInterval):
			updateIDs, err := w.Store.ListReadyUpdateIDs(ctx)
			if err != nil {
				w.Logger.Errorw("updateworker polling failed", zap.Error(err))
				continue
			}

			if err := w.startUpdates(ctx, updateIDs); err != nil {
				w.Logger.Errorw("updateworker start updates failed", zap.Error(err))
				continue
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (w *Worker) startUpdates(ctx context.Context, updateIDs []string) error {
	for _, updateID := range updateIDs {
		if err := w.startUpdate(ctx, updateID); err != nil {
			w.Logger.Errorw("updateworker start update failed", zap.String("updateID", updateID), zap.Error(err))
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
		w.Logger.Errorw("updateworker get update failed", zap.String("updateID", updateID), zap.Error(err))
		return err
	}

	if err := w.deployUpdate(shipUpdate); err != nil {
		w.Logger.Errorw("updateworker deploy update failed", zap.String("updateID", updateID), zap.Error(err))
		return err
	}

	return nil
}
