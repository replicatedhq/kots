package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func GetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get [resource]",
		Short: "Display kots resources",
		Long: `Examples:
kubectl kots get apps`,

		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
	}

	cmd.AddCommand(GetAppsCmd())
	cmd.AddCommand(GetBackupsCmd())
	cmd.AddCommand(GetVersionsCmd())
	cmd.AddCommand(GetConfigCmd())
	cmd.AddCommand(GetRestoresCmd())
	cmd.AddCommand(GetJoinCmd())

	return cmd
}
