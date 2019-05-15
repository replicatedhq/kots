package cli

import (
	"context"
	"io"

	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/editworker"
	"github.com/spf13/cobra"
)

func Edit(c *config.Config, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "run the edit worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := editworker.Get(c, out)
			if err != nil {
				return err
			}

			return w.Run(context.Background())
		},
	}

	return cmd
}
