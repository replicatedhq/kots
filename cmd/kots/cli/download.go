package cli

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/download"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func DownloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "download [app-slug]",
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

			downloadOptions := download.DownloadOptions{
				Namespace:  v.GetString("namespace"),
				Kubeconfig: v.GetString("kubeconfig"),
			}

			if err := download.Download(args[0], ExpandDir(v.GetString("dest")), downloadOptions); err != nil {
				return errors.Cause(err)
			}

			return nil
		},
	}

	cmd.Flags().String("kubeconfig", filepath.Join(homeDir(), ".kube", "config"), "the kubeconfig to use")
	cmd.Flags().String("namespace", "default", "the namespace to download from")
	cmd.Flags().String("dest", homeDir(), "the directory to store the application in")

	return cmd
}
