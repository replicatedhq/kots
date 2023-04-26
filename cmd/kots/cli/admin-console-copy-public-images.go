package cli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AdminCopyPublicImagesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "copy-public-images [registry host]",
		Short:         "Copy public admin console images",
		Long:          "Copy public admin console images to a private registry",
		Hidden:        true,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				cmd.Help()
				os.Exit(1)
			}

			endpoint := args[0]

			log := logger.NewCLILogger(cmd.OutOrStdout())
			v := viper.GetViper()

			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			options, err := genAndCheckPushOptions(endpoint, namespace, log, v)
			if err != nil {
				return err
			}

			err = kotsadm.CopyImages(*options, namespace)
			if err != nil {
				return errors.Wrap(err, "failed to copy images")
			}

			return nil
		},
	}

	cmd.Flags().String("registry-username", "", "user name to use to authenticate with the registry")
	cmd.Flags().String("registry-password", "", "password to use to authenticate with the registry")
	cmd.Flags().Bool("skip-registry-check", false, "skip the connectivity test and validation of the provided registry information")

	cmd.Flags().String("kotsadm-tag", "", "set to override the tag of kotsadm. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().MarkHidden("kotsadm-tag")

	return cmd
}
