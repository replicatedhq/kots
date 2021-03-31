package cli

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/download"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func DownloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "download [appSlug]",
		Short:         "Download Kubernetes manifests from your cluster to the local filesystem",
		Long:          `Download the active Kubernetes manifests from a cluster to the local filesystem so that they can be edited and then reapplied to the cluster with 'kots upload'.`,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			appSlug := v.GetString("slug")
			if appSlug == "" {
				if len(args) == 1 {
					appSlug = args[0]
				} else {
					cmd.Help()
					os.Exit(1)
				}
			}

			downloadOptions := download.DownloadOptions{
				Namespace:             v.GetString("namespace"),
				Overwrite:             v.GetBool("overwrite"),
				DecryptPasswordValues: v.GetBool("decrypt-password-values"),
			}

			downloadPath := filepath.Join(ExpandDir(v.GetString("dest")), appSlug)
			if err := download.Download(appSlug, downloadPath, downloadOptions); err != nil {
				return errors.Cause(err)
			}

			log := logger.NewCLILogger()
			log.ActionWithoutSpinner("")
			log.Info("The application manifests have been downloaded and saved in %s\n\nAfter editing these files, you can upload a new version using", downloadPath)
			log.Info("  kubectl kots upload --namespace %s --slug %s %s", v.GetString("namespace"), appSlug, downloadPath)
			log.ActionWithoutSpinner("")

			return nil
		},
	}

	defaultDest := homeDir()
	cwd, err := os.Getwd()
	if err == nil {
		defaultDest = cwd
	}

	cmd.Flags().String("dest", defaultDest, "the directory to store the application in")
	cmd.Flags().Bool("overwrite", false, "overwrite any local files, if present")
	cmd.Flags().String("slug", "", "the application slug to download")
	cmd.Flags().Bool("decrypt-password-values", false, "decrypt password values to plaintext")

	return cmd
}
