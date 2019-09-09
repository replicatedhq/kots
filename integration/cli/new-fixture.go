package cli

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/integration/replicated/pull"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func NewFixture() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "new-fixture [path]",
		Short:         "",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if len(args) == 0 {
				return errors.New("need to specify the path to the app bundle")
			}

			if v.GetString("type") == "replicated" {
				if err := pull.GenerateTest(v.GetString("name"), args[0]); err != nil {
					return errors.Cause(err)
				}
				return nil
			}

			return errors.New("not implemented")
		},
	}

	cmd.Flags().String("type", "", "the app type to fixture")
	cmd.Flags().String("name", "", "the name of the test")

	cmd.MarkFlagRequired("type")
	cmd.MarkFlagRequired("name")

	return cmd
}
