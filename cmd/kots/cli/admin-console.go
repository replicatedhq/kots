package cli

import (
	"os"
	"os/signal"
	"path/filepath"

	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AdminConsoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "admin-console [namespace]",
		Short:         "",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			log := logger.NewLogger()
			log.Info("")

			if len(args) == 0 {
				cmd.Help()
				os.Exit(1)
			}

			podName, err := findKotsweb(args[0])
			if err != nil {
				return err
			}

			stopCh, err := k8sutil.PortForward(v.GetString("kubeconfig"), 8800, 3000, args[0], podName)
			if err != nil {
				return err
			}
			defer close(stopCh)

			log.Info("Press Ctrl+C to exit")
			log.Info("Go to http://localhost:8800 to access the Admin Console")

			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, os.Interrupt)

			<-signalChan

			log.Info("Cleaning up")

			return nil
		},
	}

	cmd.Flags().String("kubeconfig", filepath.Join(homeDir(), ".kube", "config"), "the kubeconfig to use")

	return cmd
}
