package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func SetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                   "set",
		Short:                 "Configure kots resources",
		Long:                  ``,
		DisableFlagsInUseLine: true, // removes "kots set [flags]" from usage in help output

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

	cmd.AddCommand(SetConfigCmd())

	return cmd
}
