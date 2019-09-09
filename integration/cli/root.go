package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/replicatedhq/kots/integration/replicated"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "kots",
		Short:         "",
		Long:          `.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			fmt.Println("\nRunning integration tests")

			for _, appType := range v.GetStringSlice("upstream") {
				if appType == "replicated" {
					if err := replicated.RunIntegration(); err != nil {
						return err
					}
				}
			}

			fmt.Println("\nAll integration tests completed")
			return nil
		},
	}

	cobra.OnInitialize(initConfig)

	cmd.AddCommand(NewFixture())

	cmd.Flags().StringSlice("upstream", []string{"replicated", "helm"}, "list of app types to test")

	viper.BindPFlags(cmd.Flags())

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
	viper.SetEnvPrefix("KOTS")
	viper.AutomaticEnv()
}
