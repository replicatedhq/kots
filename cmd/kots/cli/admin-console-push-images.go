package cli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AdminPushImagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "push-images [airgap filename] [registry host]",
		Short:         "Push admin console images",
		Long:          "Push admin console images from airgap bundle to a private registry",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if len(args) != 2 {
				cmd.Help()
				os.Exit(1)
			}

			airgapArchive := args[0]
			options := kotsadmtypes.PushImagesOptions{
				Registry: registry.RegistryOptions{
					Endpoint: args[1],
					Username: v.GetString("registry-username"),
					Password: v.GetString("registry-password"),
				},
				ProgressWriter: os.Stdout,
			}

			err := kotsadm.PushImages(airgapArchive, options)
			if err != nil {
				return errors.Wrap(err, "failed to push images")
			}

			return nil
		},
	}

	cmd.Flags().String("registry-username", "", "user name to use to authenticate with the registry")
	cmd.Flags().String("registry-password", "", "password to use to authenticate with the registry")

	return cmd
}
