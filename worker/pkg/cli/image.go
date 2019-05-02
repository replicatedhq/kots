package cli

import (
	"context"
	"io"

	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/imageworker"
	"github.com/spf13/cobra"
)

func Image(c *config.Config, out io.Writer) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "image",
		Short: "run the image worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			w, err := imageworker.Get(c, out)
			if err != nil {
				return err
			}

			return w.Run(context.Background())
		},
	}

	return cmd
}
