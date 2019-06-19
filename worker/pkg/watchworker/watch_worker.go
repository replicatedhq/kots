package watchworker

import (
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/store"
	"github.com/replicatedhq/ship-cluster/worker/pkg/version"
	shipspecs "github.com/replicatedhq/ship/pkg/specs"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/spf13/viper"
)

type Worker struct {
	Config *config.Config
	Logger log.Logger

	Store store.Store
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (w *Worker) Run(ctx context.Context) error {
	logger := log.With(w.Logger, "method", "watchworker.Worker.Execute")

	level.Info(logger).Log("phase", "initialize",
		"version", version.Version(),
		"gitSHA", version.GitSHA(),
		"buildTime", version.BuildTime(),
		"buildTimeFallback", version.GetBuild().TimeFallback,
	)

	errCh := make(chan error, 1)

	go func() {
		level.Info(logger).Log("event", "db.poller.ready.start")
		err := w.startPollingDBForReadyWatches(context.Background())
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

func (w *Worker) startPollingDBForReadyWatches(ctx context.Context) error {
	logger := log.With(w.Logger, "method", "watchworker.Worker.startPollingDBForReadyWatches")

	for {
		select {
		case <-time.After(w.Config.DBPollInterval):
			watchIDs, err := w.Store.ListReadyWatchIDs(ctx)
			if err != nil {
				level.Error(logger).Log("event", "store.list.ready.watches.fail", "err", err)
				continue
			}

			if err := w.runWatches(ctx, watchIDs); err != nil {
				level.Error(logger).Log("event", "ensure.watch.running.fail", "err", err)
				continue
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (w *Worker) runWatches(ctx context.Context, watchIDs []string) error {
	logger := log.With(w.Logger, "method", "watchworker.Worker.runWatches")

	for _, watchID := range watchIDs {
		if err := w.runWatch(ctx, watchID); err != nil {
			level.Error(logger).Log("event", "runWatch.fail", "err", err)
		}
	}

	return nil
}

func (w *Worker) runWatch(ctx context.Context, watchID string) error {
	logger := log.With(w.Logger, "method", "watchworker.Worker.runWatch")

	watch, err := w.Store.GetWatch(ctx, watchID)
	if err != nil {
		level.Error(logger).Log("event", "getWatch", "err", err)
		return err
	}

	isSuccess := false
	defer func() error {
		if !isSuccess {
			if err := w.Store.SetWatchDeferred(ctx, watchID); err != nil {
				level.Error(logger).Log("event", "set watch deferred", "err", err)
				return err
			}
		}

		return nil
	}()

	existingState := state.State{}
	if err := json.Unmarshal([]byte(watch.StateJSON), &existingState); err != nil {
		level.Error(logger).Log("event", "unmarshalState", "err", err)
		return err
	}

	existingSHA := existingState.Versioned().V1.ContentSHA

	shipViper := viper.New()
	shipViper.Set("customer-endpoint", "https://pg.replicated.com/graphql")

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
			level.Error(logger).Log("event", "cancel incomplete watch updates", "err", err)
			return err
		}
		if err := w.Store.CreateWatchUpdate(ctx, watchID); err != nil {
			level.Error(logger).Log("event", "set watch update needed", "err", err)
			return err
		}
	}

	isSuccess = true
	if err := w.Store.SetWatchChecked(ctx, watchID); err != nil {
		level.Error(logger).Log("event", "set watch checked", "err", err)
		return err
	}
	return nil
}
