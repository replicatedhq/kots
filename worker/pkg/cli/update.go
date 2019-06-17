package cli

import (
	"context"
	"io"

	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/kubernetes"
	"github.com/replicatedhq/ship-cluster/worker/pkg/logger"
	"github.com/replicatedhq/ship-cluster/worker/pkg/store"
	"github.com/replicatedhq/ship-cluster/worker/pkg/updateworker"
	"github.com/spf13/cobra"
)

func Update(c *config.Config, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: "run the update worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := store.NewSQLStore(c)
			if err != nil {
				return err
			}

			k, err := kubernetes.NewClient()
			if err != nil {
				return err
			}

			worker := &updateworker.Worker{
				Config:    c,
				Logger:    logger.New(c, out),
				Store:     s,
				K8sClient: k,
			}

			return worker.Run(context.Background())
		},
	}

	return cmd
}
