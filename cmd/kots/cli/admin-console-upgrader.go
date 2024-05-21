package cli

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/upgrader"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AdminConsoleUpgraderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrader",
		Short: "Starts the KOTS Admin Console upgrader service",
		Long:  `Starts the KOTS Admin Console upgrader service`,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			params := upgrader.ServerParams{
				Port: fmt.Sprintf("%d", viper.GetInt("port")),
			}
			if err := upgrader.Serve(params); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().IntP("port", "p", 30000, "local port to listen on")

	return cmd
}
