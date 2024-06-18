package cli

import (
	"os"
	"strings"

	"github.com/replicatedhq/kots/pkg/apiserver"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func APICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Starts the API server",
		Long:  ``,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if v.GetString("log-level") == "debug" {
				logger.SetDebug()
			}

			util.PodNamespace = os.Getenv("POD_NAMESPACE")
			util.KotsadmTargetNamespace = os.Getenv("KOTSADM_TARGET_NAMESPACE")

			params := apiserver.APIServerParams{
				Version:                buildversion.Version(),
				AutocreateClusterToken: os.Getenv("AUTO_CREATE_CLUSTER_TOKEN"),
			}

			apiserver.Start(&params)
			return nil
		},
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	return cmd
}
