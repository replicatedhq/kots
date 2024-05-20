package cli

import (
	"os"
	"os/signal"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	k8sutiltypes "github.com/replicatedhq/kots/pkg/k8sutil/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AdminConsoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "admin-console",
		Short:         "Make the admin console accessible",
		Long:          "Establish port forwarding for localhost access to the kotsadm admin console.",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			log := logger.NewCLILogger(cmd.OutOrStdout())

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			timeout, err := time.ParseDuration(v.GetString("wait-duration"))
			if err != nil {
				return errors.Wrap(err, "failed to parse timeout value")
			}

			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			getPodName := func() (string, error) {
				podName, err := k8sutil.WaitForKotsadm(clientset, namespace, timeout)
				if err != nil {
					if _, ok := errors.Cause(err).(*k8sutiltypes.ErrorTimeout); ok {
						return podName, errors.Errorf("kotsadm failed to start: %s. Use the --wait-duration flag to increase timeout.", err)
					}
					return podName, errors.Wrap(err, "failed to wait for web")
				}
				return podName, nil
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			localPort := viper.GetInt("port")
			pollForAdditionalPorts := true
			if localPort != 8800 {
				pollForAdditionalPorts = false
			}

			adminConsolePort, errChan, err := k8sutil.PortForward(localPort, 3000, namespace, getPodName, pollForAdditionalPorts, stopCh, log)
			if err != nil {
				return errors.Wrap(err, "failed to port forward")
			}

			go func() {
				select {
				case err := <-errChan:
					if err != nil {
						log.Error(err)
						os.Exit(-1)
					}
				case <-stopCh:
				}
			}()

			if adminConsolePort != localPort {
				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner("Port %d is not available. The Admin Console is running on port %d", localPort, adminConsolePort)
				log.ActionWithoutSpinner("")
			}

			log.ActionWithoutSpinner("Press Ctrl+C to exit")
			log.ActionWithoutSpinner("Go to http://localhost:%d to access the Admin Console", adminConsolePort)

			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, os.Interrupt)

			<-signalChan

			log.ActionWithoutSpinner("Cleaning up")

			return nil
		},
	}

	cmd.Flags().Int("port", 8800, "local port to listen on")
	cmd.Flags().String("wait-duration", "10s", "timeout to be used while waiting for kotsadm pod to become ready. must be in Go duration format (eg: 10s, 2m)")

	cmd.AddCommand(AdminConsoleUpgradeCmd())
	cmd.AddCommand(AdminPushImagesCmd())
	cmd.AddCommand(AdminCopyPublicImagesCmd())
	cmd.AddCommand(GarbageCollectImagesCmd())
	cmd.AddCommand(AdminGenerateManifestsCmd())
	cmd.AddCommand(AdminConsoleUpgraderCmd())

	return cmd
}
