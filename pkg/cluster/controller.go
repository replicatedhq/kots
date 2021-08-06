package cluster

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"k8s.io/kubernetes/cmd/kube-controller-manager/app"
)

func runController(ctx context.Context, dataDir string) error {
	log := ctx.Value("log").(*logger.CLILogger)
	log.Info("starting kubernetes controller manager")

	serviceAccountKeyFile, err := serviceAccountKeyFilePath(dataDir)
	if err != nil {
		return errors.Wrap(err, "service account key file")
	}
	kubeconfigFile, err := kubeconfigFilePath(dataDir)
	if err != nil {
		return errors.Wrap(err, "kubeconfig file")
	}

	caCertFile, err := caCertFilePath(dataDir)
	if err != nil {
		return errors.Wrap(err, "ca cert file")
	}
	caKeyFile, err := caKeyFilePath(dataDir)
	if err != nil {
		return errors.Wrap(err, "ca key file")
	}

	args := []string{
		"--bind-address=0.0.0.0",
		"--secure-port=11252",
		"--port=0", // Don't serve insecure
		"--cluster-cidr=10.200.0.0/16",
		"--cluster-name=kubernetes",
		fmt.Sprintf("--cluster-signing-cert-file=%s", caCertFile),
		fmt.Sprintf("--cluster-signing-key-file=%s", caKeyFile),
		fmt.Sprintf("--kubeconfig=%s", kubeconfigFile),
		"--leader-elect=true",
		fmt.Sprintf("--root-ca-file=%s", caCertFile),
		fmt.Sprintf("--service-account-private-key-file=%s", serviceAccountKeyFile),
		"--service-cluster-ip-range=10.32.0.0/24",
		"--use-service-account-credentials=true",
		"--v=2",
	}

	command := app.NewControllerManagerCommand()
	command.SetArgs(args)

	go func() {
		logger.Infof("kubernetes contoller manager exited %v", command.Execute())
	}()

	// watch the readyz endpoint to know when the api server has started
	stopWaitingAfter := time.Now().Add(time.Minute)
	for {
		url := "http://localhost:11252/healthz"

		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client := http.Client{Transport: tr}

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
			return errors.New("controller manager did not start")
		}

		time.Sleep(time.Second)
	}
}
