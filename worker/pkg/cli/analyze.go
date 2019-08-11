package cli

import (
	"context"
	"io"

	"github.com/replicatedhq/kotsadm/worker/pkg/analyzeworker"
	"github.com/replicatedhq/kotsadm/worker/pkg/config"
	"github.com/replicatedhq/kotsadm/worker/pkg/kubernetes"
	"github.com/replicatedhq/kotsadm/worker/pkg/logger"
	"github.com/replicatedhq/kotsadm/worker/pkg/store"
	"github.com/spf13/cobra"
)

func Analyze(c *config.Config, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "analyze",
		Short: "run the analyze worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := store.NewSQLStore(c)
			if err != nil {
				return err
			}

			k, err := kubernetes.NewClient()
			if err != nil {
				return err
			}

			worker := &analyzeworker.Worker{
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
