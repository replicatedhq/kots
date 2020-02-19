package cli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/download"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func DownloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "download [app-slug]",
		Short:         "Download Kubernetes manifests from your cluster to the local filesystem",
		Long:          `Download the active Kubernetes manifests from a cluster to the local filesystem so that they can be edited and then reapplied to the cluster with 'kots upload'.`,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if len(args) != 1 {
				cmd.Help()
				os.Exit(1)
			}

			appSlug := args[0]

			downloadOptions := download.DownloadOptions{
				Namespace:             v.GetString("namespace"),
				KubernetesConfigFlags: kubernetesConfigFlags,
				Overwrite:             v.GetBool("overwrite"),
			}

			if err := download.Download(appSlug, ExpandDir(v.GetString("dest")), downloadOptions); err != nil {
				return errors.Cause(err)
			}

			log := logger.NewLogger()
			log.ActionWithoutSpinner("")
			log.Info("The application manifests have been downloaded and saved in %s\n\nAfter editing these files, you can upload a new version using", ExpandDir(v.GetString("dest")))
			log.Info("  kubectl kots upload --namespace %s --slug %s %s", v.GetString("namespace"), appSlug, ExpandDir(v.GetString("dest")))
			log.ActionWithoutSpinner("")

			return nil
		},
	}

	cmd.Flags().String("dest", homeDir(), "the directory to store the application in")
	cmd.Flags().Bool("overwrite", false, "overwrite any local files, if present")

	return cmd
}
