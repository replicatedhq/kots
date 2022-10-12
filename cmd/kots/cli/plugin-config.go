package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/apiserver"
	"github.com/replicatedhq/kots/pkg/archives"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	helmgetter "helm.sh/helm/v3/pkg/getter"
)

func ConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "config [app URI]",
		Short:         "Start initial config server",
		Long:          `Run a server on local host to create initial application cinfguration`,
		SilenceUsage:  true,
		SilenceErrors: false,
		Args:          cobra.ExactValidArgs(1),
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			log := logger.NewCLILogger(cmd.OutOrStdout())
			v := viper.GetViper()
			v.BindPFlags(cmd.Flags())

			chartPath := args[0]
			version := v.GetString("version")
			localPort := viper.GetInt("port")

			log.ActionWithoutSpinner("Pulling application chart")

			chartURI := chartPath
			if version != "" {
				chartURI = fmt.Sprintf("%s:%s", chartURI, version)
			}

			chartRoot, err := ioutil.TempDir("", "extracted-chart-")
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to created temp dir")
			}
			defer os.RemoveAll(chartRoot)

			chartName, err := extractHelmChart(chartURI, chartRoot)
			if err != nil {
				log.FinishSpinnerWithError()
				return errors.Wrap(err, "failed to download chart archive")
			}

			os.Setenv("IS_HELM_MANAGED", "true")
			os.Setenv("IS_INITIAL_CONFIG_MODE", "true")

			apiserver.StartHelmConfigServer(localPort, chartName, chartPath, version, chartRoot)

			log.FinishSpinner()

			return nil
		},
	}

	cmd.Flags().Int("port", 8800, "local port to listen on")
	cmd.Flags().String("version", "", "application version")

	return cmd
}

func extractHelmChart(uri string, dstDir string) (string, error) {
	if strings.HasPrefix(uri, "oci://") {
		chartGetter, err := helmgetter.NewOCIGetter()
		if err != nil {
			return "", errors.Wrap(err, "failed to create chart getter")
		}

		chartData, err := chartGetter.Get(uri)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get chart %q", uri)
		}

		if err := archives.ExtractTGZArchiveFromReader(chartData, dstDir); err != nil {
			return "", errors.Wrapf(err, "failed to extract archive from reader")
		}
	} else {
		if err := archives.ExtractTGZArchiveFromFile(uri, dstDir); err != nil {
			return "", errors.Wrapf(err, "failed to extract archive from file")
		}
	}

	fileInfos, err := ioutil.ReadDir(dstDir)
	if err != nil {
		return "", errors.Wrapf(err, "failed to read dir")
	}

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			return fileInfo.Name(), nil
		}
	}

	return "", errors.New("no charts found in the archive")
}
