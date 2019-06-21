package watchworker

import (
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/store"
	"github.com/replicatedhq/ship-cluster/worker/pkg/version"
	shipspecs "github.com/replicatedhq/ship/pkg/specs"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Worker struct {
	Config *config.Config
	Logger *zap.SugaredLogger

	Store store.Store
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (w *Worker) Run(ctx context.Context) error {
	w.Logger.Infow("starting watchworker",
		zap.String("version", version.Version()),
		zap.String("gitSHA", version.GitSHA()),
		zap.Time("buildTime", version.BuildTime()),
	)

	go func() {
		Serve(ctx, w.Config.InitServerAddress)
	}()

	errCh := make(chan error, 1)

	go func() {
		os.Setenv("GITHUB_TOKEN", w.Config.GithubToken)

		err := w.startPollingDBForReadyWatches(context.Background())
		w.Logger.Errorw("watchworker dbpoller failed", zap.Error(err))
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

func (w *Worker) startPollingDBForReadyWatches(ctx context.Context) error {
	for {
		select {
		case <-time.After(w.Config.DBPollInterval):
			watchIDs, err := w.Store.ListReadyWatchIDs(ctx)
			if err != nil {
				w.Logger.Errorw("watchworker polling failed", zap.Error(err))
				continue
			}

			if err := w.runWatches(ctx, watchIDs); err != nil {
				w.Logger.Errorw("watchworker checkimage failed", zap.Error(err))
				continue
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (w *Worker) runWatches(ctx context.Context, watchIDs []string) error {
	for _, watchID := range watchIDs {
		if err := w.runWatch(ctx, watchID); err != nil {
			w.Logger.Errorw("watchworker run watch failed", zap.Error(err))
		}
	}

	return nil
}

func (w *Worker) runWatch(ctx context.Context, watchID string) error {
	watch, err := w.Store.GetWatch(ctx, watchID)
	if err != nil {
		w.Logger.Errorw("watchworker get watch failed", zap.Error(err))
		return err
	}

	isSuccess := false
	defer func() error {
		if !isSuccess {
			if err := w.Store.SetWatchDeferred(ctx, watchID); err != nil {
				w.Logger.Errorw("watchworker set watch defered failed", zap.Error(err))
				return err
			}
		}

		return nil
	}()

	existingState := state.State{}
	if err := json.Unmarshal([]byte(watch.StateJSON), &existingState); err != nil {
		w.Logger.Errorw("watchworker unmarshal state failed", zap.Error(err))
		return err
	}

	existingSHA := existingState.Versioned().V1.ContentSHA

	shipViper := viper.New()
	shipViper.Set("customer-endpoint", "https://pg.replicated.com/graphql")
	shipViper.Set("prefer-git", true)
	shipViper.Set("retries", 3)

	contentProcessor, err := shipspecs.NewContentProcessor(shipViper)
	if err != nil {
		return err
	}
	defer contentProcessor.RemoveAll()

	resolvedUpstream, err := contentProcessor.MaybeResolveVersionedUpstream(ctx, existingState.V1.Upstream, existingState)
	if err != nil {
		return err
	}
	latestSHA, err := contentProcessor.ReadContentSHAForWatch(ctx, resolvedUpstream)
	if err != nil {
		return err
	}

	if existingSHA != latestSHA {
		if err := w.Store.CancelIncompleteWatchUpdates(ctx, watchID); err != nil {
			w.Logger.Errorw("watchworker cancel uncomplete watch updates failed", zap.Error(err))
			return err
		}
		if err := w.Store.CreateWatchUpdate(ctx, watchID); err != nil {
			w.Logger.Errorw("watchworker create watch update failed", zap.Error(err))
			return err
		}
	}

	isSuccess = true
	if err := w.Store.SetWatchChecked(ctx, watchID); err != nil {
		w.Logger.Errorw("watchworker set watch checked failed", zap.Error(err))
		return err
	}
	return nil
}
