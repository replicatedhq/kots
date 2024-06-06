package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kotsadm",
		Short: "kotsadm is the Admin Console for KOTS",
		Long:  ``,
		Args:  cobra.MinimumNArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	cmd.PersistentFlags().String("log-level", "info", "set the log level")

	cmd.AddCommand(APICmd())
	cmd.AddCommand(MigrateCmd())
	cmd.AddCommand(CompletionCmd())

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	return cmd
}

func InitAndExecute() {
	if err := RootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetEnvPrefix("KOTSADM")
	viper.AutomaticEnv()
}
