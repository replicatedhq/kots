package cli

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/upload"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

/*
This code needs to be worked into an existing CLI method
*/

func InstallLicenseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "install-license [path-to-license]",
		Short:         "",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if len(args) == 0 {
				cmd.Help()
				os.Exit(1)
			}

			uploadLicenseOptions := upload.UploadLicenseOptions{
				Namespace:  v.GetString("namespace"),
				Kubeconfig: v.GetString("kubeconfig"),
				NewAppName: v.GetString("name"),
			}

			if err := upload.UploadLicense(ExpandDir(args[0]), uploadLicenseOptions); err != nil {
				return errors.Cause(err)
			}

			return nil
		},
	}

	cmd.Flags().String("kubeconfig", filepath.Join(homeDir(), ".kube", "config"), "the kubeconfig to use")
	cmd.Flags().String("namespace", "default", "the namespace to upload to")
	cmd.Flags().String("name", "", "the name of the kotsadm application to create")

	return cmd
}
