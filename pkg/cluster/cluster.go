package cluster

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
)

// Start will start the embedded cluster.
// This function blocks until the cluster control plane has started
func Start(ctx context.Context, slug string, dataDir string) error {
	log := ctx.Value("log").(*logger.CLILogger)
	log.ActionWithoutSpinner("Starting cluster")

	// init tls and misc
	// this function is synchronous and blocks until ready
	if err := clusterInit(ctx, dataDir, slug, "1.21.3"); err != nil {
		return errors.Wrap(err, "init cluster")
	}

	// start the api server
	// if err := runAPIServer(ctx, dataDir, slug); err != nil {
	// 	return errors.Wrap(err, "start api server")
	// }

	// // start the scheduler on port 11251 (this is a non-standard port)
	// if err := runScheduler(ctx, dataDir); err != nil {
	// 	return errors.Wrap(err, "start scheduler")
	// }

	// start the controller manager on port 11252 (non standard)
	// TODO the controller should start
	// if err := runController(ctx, dataDir); err != nil {
	// 	return errors.Wrap(err, "start controller")
	// }

	// because these are all synchoronous, the api is ready and we
	// can install our addons
	// kubeconfigPath, err := kubeconfigFilePath(dataDir)
	// if err != nil {
	// 	return errors.Wrap(err, "get kubeconfig path")
	// }

	// if err := installCNI(kubeconfigPath); err != nil {
	// 	return errors.Wrap(err, "install antrea")
	// }

	// fmt.Println("starting cri")
	// if err := startCRI(dataDir); err != nil {
	// 	return errors.Wrap(err, "install cri")
	// }

	fmt.Println("starting kubelet")
	if err := startKubelet(ctx, dataDir); err != nil {
		return errors.Wrap(err, "install kubelet")
	}

	return nil
}
