package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AdminConsoleUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "upgrade",
		Short:         "Upgrade the admin console to the latest version",
		Long:          "Upgrade the admin console to the latest version",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if v.GetBool("force-upgrade-kurl") {
				prompt := promptui.Prompt{
					Label:     fmt.Sprintf("Upgrading a kotsadm instance created by kURL can result in data loss. Do you want to continue"),
					IsConfirm: true,
				}

				for {
					resp, err := prompt.Run()
					if err == promptui.ErrInterrupt {
						os.Exit(-1)
					}
					if strings.ToLower(resp) == "n" {
						os.Exit(-1)
					}
					if strings.ToLower(resp) == "y" {
						break
					}
				}
			}

			upgradeOptions := kotsadmtypes.UpgradeOptions{
				Namespace:             v.GetString("namespace"),
				KubernetesConfigFlags: kubernetesConfigFlags,
				ForceUpgradeKurl:      v.GetBool("force-upgrade-kurl"),
				EnsureRBAC:            v.GetBool("ensure-rbac"),

				KotsadmOptions: kotsadmtypes.KotsadmOptions{
					OverrideVersion:   v.GetString("kotsadm-tag"),
					OverrideRegistry:  v.GetString("kotsadm-registry"),
					OverrideNamespace: v.GetString("kotsadm-namespace"),
					Username:          v.GetString("registry-username"),
					Password:          v.GetString("registry-password"),
				},
			}

			timeout, err := time.ParseDuration(v.GetString("wait-duration"))
			if err != nil {
				return errors.Wrap(err, "failed to parse timeout value")
			}

			upgradeOptions.Timeout = timeout

			log := logger.NewCLILogger()

			if upgradeOptions.Namespace != "default" {
				log.ActionWithoutSpinner("Upgrading Admin Console")
			} else {
				log.ActionWithoutSpinner("Upgrading Admin Console in the default namespace")
			}
			if err := kotsadm.Upgrade(upgradeOptions); err != nil {
				return errors.Wrap(err, "failed to upgrade")
			}

			log.ActionWithoutSpinner("")
			log.ActionWithoutSpinner("The Admin Console is running the latest version")
			log.ActionWithoutSpinner("To access the Admin Console, run kubectl kots admin-console --namespace %s", v.GetString("namespace"))
			log.ActionWithoutSpinner("")

			return nil
		},
	}

	cmd.Flags().Bool("force-upgrade-kurl", false, "set to force upgrade even if this is a kurl cluster")
	cmd.Flags().String("kotsadm-tag", "", "set to override the tag of kotsadm. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().String("kotsadm-registry", "", "set to override the registry of kotsadm images. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().String("registry-username", "", "user name to use to authenticate with the registry")
	cmd.Flags().String("registry-password", "", "password to use to authenticate with the registry")
	cmd.Flags().String("kotsadm-namespace", "", "set to override the namespace of kotsadm images. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().String("wait-duration", "2m", "timeout out to be used while waiting for individual components to be ready.  must be in Go duration format (eg: 10s, 2m)")
	cmd.Flags().Bool("ensure-rbac", true, "when set, kots will create the roles and rolebindings necessary to manage applications")
	cmd.Flags().MarkHidden("force-upgrade-kurl")
	cmd.Flags().MarkHidden("kotsadm-tag")
	cmd.Flags().MarkHidden("kotsadm-namespace")
	cmd.Flags().MarkHidden("ensure-rbac")

	return cmd
}
