package cli

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/cli-runtime/pkg/genericclioptions"
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

	kubernetesConfigFlags = genericclioptions.NewConfigFlags(false)
	kubernetesConfigFlags.AddFlags(cmd.PersistentFlags())

	cmd.AddCommand(PullCmd())
	cmd.AddCommand(InstallCmd())
	cmd.AddCommand(UploadCmd())
	cmd.AddCommand(DownloadCmd())
	cmd.AddCommand(UpstreamCmd())
	cmd.AddCommand(RemoveCmd())
	cmd.AddCommand(AdminConsoleCmd())
	cmd.AddCommand(ResetPasswordCmd())
	cmd.AddCommand(VersionCmd())
	cmd.AddCommand(VeleroCmd())
	cmd.AddCommand(BackupCmd())
	cmd.AddCommand(RestoreCmd())
	cmd.AddCommand(IngressCmd())
	cmd.AddCommand(IdentityServiceCmd())
	cmd.AddCommand(AppStatusCmd())
	cmd.AddCommand(GetCmd())
	cmd.AddCommand(SetCmd())

	viper.BindPFlags(cmd.Flags())

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}

func InitAndExecute() {
	if err := RootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetEnvPrefix("KOTS")
	viper.AutomaticEnv()
}
