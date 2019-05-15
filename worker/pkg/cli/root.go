package cli

import (
	"io"
	"os"

	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	pkgdebug "github.com/replicatedhq/ship-cluster/worker/pkg/debug"
	"github.com/replicatedhq/ship-cluster/worker/pkg/version"
	"github.com/spf13/cobra"
)

// RootCmd represents the base command when called without any subcommands
func RootCmd(c *config.Config, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:          "ship-cluster-worker",
		Short:        "run a ship cloud worker",
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			version.Init()
			pkgdebug.Init()
			return nil
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
			os.Exit(1)
		},
	}

	cmd.AddCommand(Init(c, out))
	cmd.AddCommand(Watch(c, out))
	cmd.AddCommand(Update(c, out))
	cmd.AddCommand(Image(c, out))
	cmd.AddCommand(Edit(c, out))

	return cmd
}
