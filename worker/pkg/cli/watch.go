package cli

import (
	"context"
	"io"

	"github.com/replicatedhq/kotsadm/worker/pkg/config"
	"github.com/replicatedhq/kotsadm/worker/pkg/logger"
	"github.com/replicatedhq/kotsadm/worker/pkg/store"
	"github.com/replicatedhq/kotsadm/worker/pkg/watchworker"
	"github.com/spf13/cobra"
)

func Watch(c *config.Config, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "watch",
		Short: "run the watch worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := store.NewSQLStore(c)
			if err != nil {
				return err
			}

			worker := &watchworker.Worker{
				Config: c,
				Logger: logger.New(c, out),
				Store:  s,
			}

			return worker.Run(context.Background())
		},
	}

	return cmd
}
