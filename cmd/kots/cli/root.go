package cli

import (
	"os"
	"strings"

	"github.com/replicatedhq/kots/pkg/k8sutil"
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

	k8sutil.AddFlags(cmd.PersistentFlags())

	cmd.AddCommand(PullCmd())
	cmd.AddCommand(InstallCmd())
	cmd.AddCommand(UploadCmd())
	cmd.AddCommand(DownloadCmd())
	cmd.AddCommand(UpstreamCmd())
	cmd.AddCommand(RemoveCmd())
	cmd.AddCommand(AdminConsoleCmd())
	cmd.AddCommand(ResetPasswordCmd())
	cmd.AddCommand(ResetTLSCmd())
	cmd.AddCommand(VersionCmd())
	cmd.AddCommand(VeleroCmd())
	cmd.AddCommand(DockerCmd())
	cmd.AddCommand(BackupCmd())
	cmd.AddCommand(RestoreCmd())
	cmd.AddCommand(IngressCmd())
	cmd.AddCommand(IdentityServiceCmd())
	cmd.AddCommand(AppStatusCmd())
	cmd.AddCommand(GetCmd())
	cmd.AddCommand(SetCmd())
	cmd.AddCommand(RunCmd())

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
