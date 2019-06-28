package analyzeworker

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/store"
	"github.com/replicatedhq/ship-cluster/worker/pkg/version"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

type Worker struct {
	Config *config.Config
	Logger *zap.SugaredLogger

	Store     store.Store
	K8sClient kubernetes.Interface
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (w *Worker) Run(ctx context.Context) error {
	w.Logger.Infow("starting analyzeworker",
		zap.String("version", version.Version()),
		zap.String("gitSHA", version.GitSHA()),
		zap.Time("buildTime", version.BuildTime()),
	)

	go func() {
		Serve(ctx, w.Config.AnalyzeServerAddress)
	}()

	errCh := make(chan error, 2)

	go func() {
		err := w.startPollingDBForReadyAnalysis(context.Background())
		w.Logger.Errorw("analyzeworker dbpoller failed", zap.Error(err))
		errCh <- errors.Wrap(err, "ready poller ended")
	}()

	go func() {
		err := w.runInformer(context.Background())
		w.Logger.Errorw("analyzeworker informer failed", zap.Error(err))
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

func (w *Worker) startPollingDBForReadyAnalysis(ctx context.Context) error {
	for {
		select {
		case <-time.After(w.Config.DBPollInterval):
			supportBundleIDs, err := w.Store.ListReadyAnalysisIDs(ctx)
			if err != nil {
				w.Logger.Errorw("analyzeworker polling failed", zap.Error(err))
				continue
			}

			if err := w.startAnalyses(ctx, supportBundleIDs); err != nil {
				w.Logger.Errorw("analyzeworker start failed", zap.Error(err))
				continue
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (w *Worker) startAnalyses(ctx context.Context, supportBundleIDs []string) error {
	for _, supportBundleID := range supportBundleIDs {
		if err := w.startAnalysis(ctx, supportBundleID); err != nil {
			w.Logger.Errorw("analyzeworker run watch failed", zap.Error(err))
		}
	}

	return nil
}

func (w *Worker) startAnalysis(ctx context.Context, supportBundleID string) error {
	if err := w.Store.SetAnalysisStarted(ctx, supportBundleID); err != nil {
		return err
	}

	supportBundle, err := w.Store.GetSupportBundle(context.TODO(), supportBundleID)
	if err != nil {
		w.Logger.Errorw("analysisworker get supportbundle failed", zap.String("supportBundleID", supportBundleID), zap.Error(err))
		return err
	}
	if err := w.deployAnalyzer(supportBundle); err != nil {
		w.Logger.Errorw("analysis deploy update failed", zap.String("supportBundleID", supportBundleID), zap.Error(err))
		return err
	}

	return nil
}
