package cluster

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"
)

func runScheduler(ctx context.Context, wg *sync.WaitGroup, dataDir string) error {
	log := ctx.Value("log").(*logger.CLILogger)
	log.Info("starting kubernetes scheduler")

	schedulerConfigFile, err := schedulerConfigFilePath(dataDir)
	if err != nil {
		return errors.Wrap(err, "scheduler config file path")
	}

	args := []string{
		fmt.Sprintf("--config=%s", schedulerConfigFile),
		"--v=2",
	}

	command := app.NewSchedulerCommand()
	command.SetArgs(args)

	go func() {
		// TODO @divolgin this error needs to be nadled.
		logger.Infof("kubernetes scheduler exited %v", command.Execute())
	}()

	wg.Done()

	return nil
}
