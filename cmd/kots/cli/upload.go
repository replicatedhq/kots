package cli

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upload"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

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

			log := logger.NewCLILogger()

			sourceDir := homeDir()
			if len(args) > 0 {
				sourceDir = ExpandDir(args[0])
			}

			uploadOptions := upload.UploadOptions{
				Namespace:             v.GetString("namespace"),
				KubernetesConfigFlags: kubernetesConfigFlags,
				ExistingAppSlug:       v.GetString("slug"),
				NewAppName:            v.GetString("name"),
				UpstreamURI:           v.GetString("upstream-uri"),
				Endpoint:              "http://localhost:3000",
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			localPort, errChan, err := upload.StartPortForward(uploadOptions.Namespace, kubernetesConfigFlags, stopCh, log)
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

			if err := upload.Upload(sourceDir, uploadOptions); err != nil {
				return errors.Cause(err)
			}

			return nil
		},
	}

	cmd.Flags().String("slug", "", "the application slug to use. if not present, a new one will be created")
	cmd.Flags().String("name", "", "the name of the kotsadm application to create")
	cmd.Flags().String("upstream-uri", "", "the upstream uri that can be used to check for updates")

	return cmd
}
