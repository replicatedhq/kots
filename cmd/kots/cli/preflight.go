package cli

import (
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func PreflightCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "preflight",
		Short:         "Run preflights",
		Long:          `Run preflights without a running KOTS installation`,
		SilenceUsage:  false,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			log := logger.NewCLILogger(cmd.OutOrStdout())
			v := viper.GetViper()
			v.BindPFlags(cmd.Flags())

			_, err := cmd.Flags().GetString("preflight-spec")
			if err != nil {
				log.Errorf("failed to get preflight flag: %v\n", err)
				return err
			}

			_, err = cmd.Flags().GetString("config-spec")
			if err != nil {
				log.Errorf("failed to get config flag: %v\n", err)
				return err
			}

			return nil
		},
	}

	cmd.Flags().String("preflight-spec", "", "the filename or url of the Preflight spec")
	cmd.Flags().String("config-spec", "", "the filename of the Config spec")

	return cmd
}
