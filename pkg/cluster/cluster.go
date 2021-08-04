package cluster

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
)

// Start will start the embedded cluster.
// This function blocks until the cluster control plane has started
func Start(ctx context.Context, slug string, dataDir string) error {
	log := ctx.Value("log").(*logger.CLILogger)
	log.ActionWithSpinner("Starting cluster")
	defer log.FinishSpinner()

	// init tls and misc
	// this function is synchronous and blocks until ready
	if err := clusterInit(ctx, dataDir, slug, "1.21.3"); err != nil {
		return errors.Wrap(err, "init cluster")
	}

	// start the api server
	if err := runAPIServer(ctx, dataDir, slug); err != nil {
		return errors.Wrap(err, "start api server")
	}

	// start the scheduler on port 11251 (this is a non-standard port)
	// wg.Add(1)
	// if err := runScheduler(ctx, &wg, dataDir); err != nil {
	// 	return errors.Wrap(err, "start scheduler")
	// }

	// // start the controller manager on port 11252 (non standard)
	// wg.Add(1)
	// if err := runController(ctx, &wg, dataDir); err != nil {
	// 	return errors.Wrap(err, "start controller")
	// }

	return nil
}
