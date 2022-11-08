package cli

import (
	"time"

	"github.com/pkg/errors"

	"github.com/replicatedhq/kots/pkg/ha"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func EnableHACmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "enable-ha",
		Short:         "(Deprecated) Enables HA mode for the admin console",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			timeout, err := time.ParseDuration(v.GetString("wait-duration"))
			if err != nil {
				return errors.Wrap(err, "failed to parse timeout value")
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			canRunHA, reason, err := ha.CanRunHA(cmd.Context(), clientset)
			if err != nil {
				return errors.Wrap(err, "failed to check if can run in HA mode")
			}

			if !canRunHA {
				return errors.Errorf("Cannot enable HA mode because: %s", reason)
			}

			log := logger.NewCLILogger(cmd.OutOrStdout())
			log.ActionWithSpinner("Enabling HA mode")

			if err := ha.EnableHA(cmd.Context(), clientset, namespace, timeout); err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to enable HA mode")
			}

			log.FinishSpinner()
			log.ActionWithoutSpinner("HA mode enabled successfully")

			return nil
		},
	}

	cmd.Flags().String("wait-duration", "5m", "timeout out to be used while waiting for individual components to be ready. must be in Go duration format (eg: 10s, 2m)")

	return cmd
}
