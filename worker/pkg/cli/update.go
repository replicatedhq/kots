package cli

import (
	"context"
	"io"

	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/updateworker"
	"github.com/spf13/cobra"
)

func Update(c *config.Config, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "run the update worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := updateworker.Get(c, out)
			if err != nil {
				return err
			}

			return w.Run(context.Background())
		},
	}

	return cmd
}
