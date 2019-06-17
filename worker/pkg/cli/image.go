package cli

import (
	"context"
	"io"

	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/imageworker"
	"github.com/replicatedhq/ship-cluster/worker/pkg/logger"
	"github.com/replicatedhq/ship-cluster/worker/pkg/store"
	"github.com/spf13/cobra"
)

func Image(c *config.Config, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "run the image worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := store.NewSQLStore(c)
			if err != nil {
				return err
			}

			worker := &imageworker.Worker{
				Config: c,
				Logger: logger.New(c, out),
				Store:  s,
			}

			return worker.Run(context.Background())
		},
	}

	return cmd
}
