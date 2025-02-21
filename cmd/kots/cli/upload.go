package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upload"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type UploadOutput struct {
	Success bool   `json:"success"`
	AppSlug string `json:"appSlug,omitempty"`
	Error   string `json:"error,omitempty"`
}

func UploadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "upload [source]",
		Short:         "Upload Kubernetes manifests from the local filesystem to your cluster",
		Long:          `Upload Kubernetes manifests from the local filesystem to a cluster, creating a new version of the application that can be deployed.`,
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

			log := logger.NewCLILogger(cmd.OutOrStdout())

			sourceDir := util.HomeDir()
			if len(args) > 0 {
				sourceDir = ExpandDir(args[0])
			}

			output := v.GetString("output")
			if output != "json" && output != "" {
				return errors.Errorf("output format %s not supported (allowed formats are: json)", output)
			}

			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			uploadOptions := upload.UploadOptions{
				Namespace:       namespace,
				ExistingAppSlug: v.GetString("slug"),
				NewAppName:      v.GetString("name"),
				UpstreamURI:     v.GetString("upstream-uri"),
				Endpoint:        "http://localhost:3000",
				Silent:          output != "",
				Deploy:          v.GetBool("deploy"),
				SkipPreflights:  v.GetBool("skip-preflights"),
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			localPort, errChan, err := upload.StartPortForward(uploadOptions.Namespace, stopCh, log)
			if err != nil {
				return errors.Wrap(err, "failed to port forward")
			}

			uploadOptions.Endpoint = fmt.Sprintf("http://localhost:%d", localPort)
			go func() {
				select {
				case err := <-errChan:
					if err != nil {
						log.Error(err)
						os.Exit(-1)
					}
				case <-stopCh:
				}
			}()

			var uploadOutput UploadOutput
			appSlug, err := upload.Upload(sourceDir, uploadOptions)
			if err != nil && output == "" {
				return errors.Cause(err)
			} else if err != nil {
				uploadOutput.Error = fmt.Sprint(errors.Cause(err))
			} else {
				uploadOutput.Success = true
				uploadOutput.AppSlug = appSlug
			}

			if output == "json" {
				outputJSON, err := json.Marshal(uploadOutput)
				if err != nil {
					return errors.Wrap(err, "error marshaling JSON")
				}
				log.Info("%s", string(outputJSON))
			}

			return nil
		},
	}

	cmd.Flags().String("slug", "", "the application slug to use. if not present, a new one will be created")
	cmd.Flags().String("name", "", "the name of the kotsadm application to create")
	cmd.Flags().String("upstream-uri", "", "the upstream uri that can be used to check for updates")
	cmd.Flags().StringP("output", "o", "", "output format (currently supported: json)")

	cmd.Flags().Bool("deploy", false, "when set, automatically deploy the uploaded version")
	cmd.Flags().Bool("skip-preflights", false, "set to true to skip preflight checks")

	return cmd
}
