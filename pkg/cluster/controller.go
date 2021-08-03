package cluster

import (
	"context"
	"fmt"

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
		"--bind-address=0.0.0.0:11252",
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

	// <-ctx.Done()

	return nil
}
