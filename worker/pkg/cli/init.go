package cli

import (
	"context"
	"io"

	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/initworker"
	"github.com/spf13/cobra"
)

func Init(c *config.Config, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "run the init worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := initworker.Get(c, out)
			if err != nil {
				return err
			}

			return w.Run(context.Background())
		},
	}

	return cmd
}
