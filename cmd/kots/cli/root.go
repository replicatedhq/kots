package cli

import (
	"os"
	"strings"

	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
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
		PersistentPostRun: func(cmd *cobra.Command, args []string) {
			// NOTE: If a PersistentPostRun is specified for a subcommand, it will override the root PersistentPostRun
			log := logger.NewCLILogger(cmd.ErrOrStderr())
			err := cliVersionCheck(log)
			if err != nil {
				// likely unable to set up port-forwarding to perform check
				// not logging since this would be expected for some commands
			}
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
	cmd.AddCommand(CompletionCmd())
	cmd.AddCommand(DockerRegistryCmd())
	cmd.AddCommand(EnableHACmd())
	cmd.AddCommand(UpgradeServiceCmd())
	cmd.AddCommand(AirgapUploadCmd())

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
