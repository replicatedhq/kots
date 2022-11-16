package cli

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/password"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func ResetPasswordCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "reset-password [namespace]",
		Short:         "Change the password on the admin console",
		Long:          `Change the password on the Admin Console`,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			log := logger.NewCLILogger(cmd.OutOrStdout())

			// use namespace-as-arg if provided, else use namespace from -n/--namespace
			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}
			if len(args) == 1 {
				namespace = args[0]
			} else if len(args) > 1 {
				fmt.Printf("more than one argument supplied: %+v\n", args)
				os.Exit(1)
			}

			if namespace == "" {
				fmt.Printf("a namespace must be provided as an argument or via -n/--namespace\n")
				os.Exit(1)
			}

			log.ActionWithoutSpinner("Reset the admin console password for %s", namespace)
			newPassword, err := util.PromptForNewPassword()
			if err != nil {
				os.Exit(1)
			}

			if err := setKotsadmPassword(newPassword, namespace); err != nil {
				return errors.Wrap(err, "failed to set new password")
			}

			log.ActionWithoutSpinner("The admin console password has been reset")
			return nil
		},
	}

	return cmd
}

func setKotsadmPassword(newPassword string, namespace string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to create k8s client")
	}

	if err := password.ChangePassword(clientset, namespace, newPassword); err != nil {
		return errors.Wrap(err, "failed to set new password")
	}
	return nil
}
