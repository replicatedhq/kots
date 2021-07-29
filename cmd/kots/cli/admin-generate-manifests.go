package cli

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AdminGenerateManifestsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "generate-manifests",
		Short:         "Generate the Admin Console manifests and store in the local filesystem",
		Long:          "Generate the Admin Console manifests and store in the local filesystem, so they can be edited before deploying them to a cluster.",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			renderDir := ExpandDir(v.GetString("rootdir"))
			options := upstreamtypes.WriteOptions{
				SharedPassword:     v.GetString("shared-password"),
				HTTPProxyEnvValue:  v.GetString("http-proxy"),
				HTTPSProxyEnvValue: v.GetString("https-proxy"),
				NoProxyEnvValue:    v.GetString("no-proxy"),
				IncludeMinio:       v.GetBool("with-minio"),
			}
			adminConsoleFiles, err := upstream.GenerateAdminConsoleFiles(renderDir, options)
			if err != nil {
				return errors.Wrap(err, "failed to generate admin console files")
			}

			for _, file := range adminConsoleFiles {
				fileRenderPath := filepath.Join(renderDir, file.Path)
				d, _ := filepath.Split(fileRenderPath)
				if _, err := os.Stat(d); os.IsNotExist(err) {
					if err := os.MkdirAll(d, 0744); err != nil {
						return errors.Wrap(err, "failed to mkdir")
					}
				}

				if err := ioutil.WriteFile(fileRenderPath, file.Content, 0644); err != nil {
					return errors.Wrapf(err, "failed to write file %s", fileRenderPath)
				}
			}

			log := logger.NewCLILogger()
			log.Info("Admin Console manifests created in %s", filepath.Join(renderDir, "admin-console"))

			return nil
		},
	}

	cmd.Flags().String("rootdir", homeDir(), "root directory that will be used to write the yaml to")
	cmd.Flags().String("http-proxy", "", "sets HTTP_PROXY environment variable in all KOTS Admin Console components")
	cmd.Flags().String("https-proxy", "", "sets HTTPS_PROXY environment variable in all KOTS Admin Console components")
	cmd.Flags().String("no-proxy", "", "sets NO_PROXY environment variable in all KOTS Admin Console components")
	cmd.Flags().String("shared-password", "", "shared password to use when deploying the admin console")
	cmd.Flags().Bool("with-minio", true, "set to true to include a local minio instance to be used for storage")

	return cmd
}
