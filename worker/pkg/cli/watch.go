package cli

import (
	"context"
	"io"

	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/watchworker"
	"github.com/spf13/cobra"
)

func Watch(c *config.Config, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "run the watch worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := watchworker.Get(c, out)
			if err != nil {
				return err
			}

			return w.Run(context.Background())
		},
	}

	return cmd
}
