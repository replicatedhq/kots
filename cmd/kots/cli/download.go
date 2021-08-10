package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/download"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type DownloadOutput struct {
	Success          bool   `json:"success"`
	DownloadLocation string `json:"downloadLocation,omitempty"`
	UploadCommand    string `json:"uploadCommand,omitempty"`
	Error            string `json:"error,omitempty"`
}

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

			output := v.GetString("output")
			if output != "json" && output != "" {
				return errors.Errorf("output format %s not supported (allowed formats are: json)", output)
			}

			downloadOptions := download.DownloadOptions{
				Namespace:             v.GetString("namespace"),
				Overwrite:             v.GetBool("overwrite"),
				Silent:                output != "",
				DecryptPasswordValues: v.GetBool("decrypt-password-values"),
			}

			var downloadOutput DownloadOutput
			downloadPath := filepath.Join(ExpandDir(v.GetString("dest")), appSlug)
			err := download.Download(appSlug, downloadPath, downloadOptions)
			if err != nil && output == "" {
				return errors.Cause(err)
			} else if err != nil {
				downloadOutput.Error = fmt.Sprint(errors.Cause(err))
			} else {
				downloadOutput.Success = true
				downloadOutput.DownloadLocation = downloadPath
				downloadOutput.UploadCommand = fmt.Sprintf("kubectl kots upload --namespace %s --slug %s %s", v.GetString("namespace"), appSlug, downloadPath)
			}

			log := logger.NewCLILogger()
			if output == "json" {
				outputJSON, err := json.Marshal(downloadOutput)
				if err != nil {
					return errors.Wrap(err, "error marshaling JSON")
				}
				log.Info(string(outputJSON))
				return nil
			}

			log.ActionWithoutSpinner("")
			log.Info("The application manifests have been downloaded and saved in %s\n\nAfter editing these files, you can upload a new version using", downloadPath)
			log.Info("  %s", downloadOutput.UploadCommand)
			log.ActionWithoutSpinner("")

			return nil
		},
	}

	cmd.Flags().String("dest", ".", "the directory to store the application in")
	cmd.Flags().Bool("overwrite", false, "overwrite any local files, if present")
	cmd.Flags().String("slug", "", "the application slug to download")
	cmd.Flags().Bool("decrypt-password-values", false, "decrypt password values to plaintext")
	cmd.Flags().StringP("output", "o", "", "output format (currently supported: json)")

	return cmd
}
