package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/replicatedhq/ship-operator-tools/pkg/version"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

func RootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ship-operator-tools",
		Short:         "make pull requests",
		Long:          `Make pull requests with GitHub app credentials`,
		SilenceUsage:  true,
		SilenceErrors: true,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(1)
		},
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			version.Init()
			return nil
		},
	}
	cobra.OnInitialize(initConfig)

	cmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is /etc/replicated/ship-operator-tools.yaml)")
	cmd.PersistentFlags().String("log-level", "off", "Log level")

	cmd.AddCommand(Webhook())
	viper.BindPFlags(cmd.Flags())
	viper.BindPFlags(cmd.PersistentFlags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}

func Execute() {
	if err := RootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath("/etc/replicated")
		viper.SetConfigName("ship-operator-tools")
	}

	viper.AutomaticEnv() // read in environment variables that match
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
