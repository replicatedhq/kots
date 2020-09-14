package cli

import (
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/snapshot"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func RestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore [backup name]",
		Short: "Starts kotsadm snapshot restore",
		Long:  ``,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				cmd.Help()
				os.Exit(1)
			}

			v := viper.GetViper()
			if v.GetString("log-level") == "debug" {
				logger.SetDebug()
			}

			if err := kotsadm.Delete(&kotsadmtypes.DeleteOptions{}); err != nil {
				return errors.Wrap(err, "failed to delete kotsadm")
			}

			if err := snapshot.DeleteRestore(args[0]); err != nil {
				return errors.Wrap(err, "failed to delete restore")
			}

			if err := snapshot.CreateRestore(args[0]); err != nil {
				return errors.Wrap(err, "failed to create restore")
			}

			return nil
		},
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	return cmd
}
