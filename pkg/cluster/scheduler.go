package cluster

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"k8s.io/kubernetes/cmd/kube-scheduler/app"
)

func runScheduler(ctx context.Context, dataDir string) error {
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

	// watch the readyz endpoint to know when the api server has started
	stopWaitingAfter := time.Now().Add(time.Minute)
	for {
		url := "http://localhost:11251/healthz"

		client := http.Client{}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return errors.Wrap(err, "failed to create http request")
		}

		resp, err := client.Do(req)
		if err != nil {
			time.Sleep(time.Second)
			continue // keep trying
		}
		if resp.StatusCode == http.StatusOK {
			return nil
		}

		if stopWaitingAfter.Before(time.Now()) {
			return errors.New("scheduler did not start")
		}

		time.Sleep(time.Second)
	}
}
