package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func HelmCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "helm [subcommand]",
		Short: "Commands for working with Helm charts",
		Long: `Examples:
kubectl kots helm post-renderer`,

		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				cmd.Help()
				os.Exit(1)
			}

			return nil
		},
	}

	cmd.AddCommand(PostRendererCmd())

	return cmd
}
