package imageworker

import (
	"io"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	pkgdebug "github.com/replicatedhq/ship-cluster/worker/pkg/debug"
	"github.com/replicatedhq/ship-cluster/worker/pkg/logger"
	"github.com/replicatedhq/ship-cluster/worker/pkg/store"
	"go.uber.org/dig"
)

func buildInjector(c *config.Config, out io.Writer) (*dig.Container, error) {
	providers := []interface{}{
		func() *config.Config {
			return c
		},
		func() io.Writer {
			return out
		},
		logger.New,

		store.NewSQLStore,
		pkgdebug.NewServer,

		NewWorker,
	}

	container := dig.New()

	for _, provider := range providers {
		err := container.Provide(provider)
		if err != nil {
			return nil, errors.Wrap(err, "register providers")
		}
	}

	return container, nil
}

func Get(c *config.Config, out io.Writer) (*Worker, error) {
	debug := log.With(level.Debug(logger.New(c, out)), "component", "injector", "phase", "instance.get")

	debug.Log("event", "injector.build")
	injector, err := buildInjector(c, out)
	if err != nil {
		debug.Log("event", "injector.build.fail", "error", err)
		return nil, errors.Wrap(err, "build injector")
	}

	var worker *Worker

	// we return nil below , so the error will only ever be a construction error
	debug.Log("event", "injector.invoke")
	if err := injector.Invoke(func(w *Worker) {
		debug.Log("event", "injector.invoke.resolve")
		worker = w
	}); err != nil {
		debug.Log("event", "injector.invoke.fail", "err", err)
		return nil, errors.Wrap(err, "resolve dependencies")
	}

	return worker, nil
}
