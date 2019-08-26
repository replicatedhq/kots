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
		Use:   "kots",
		Short: "",
		Long:  `.`,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
	}

	cobra.OnInitialize(initConfig)

	cmd.AddCommand(PullCmd())
	cmd.AddCommand(InstallCmd())
	cmd.AddCommand(UploadCmd())
	cmd.AddCommand(DownloadCmd())
	cmd.AddCommand(AdminConsoleCmd())

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
