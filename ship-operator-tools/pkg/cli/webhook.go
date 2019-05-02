package cli

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-operator-tools/pkg/webhook"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func Webhook() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook URL",
		Short: "webhook URL",
		Long:  `webhook URL`,
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()
			if len(args) == 0 {
				cmd.Help()
				return errors.New("Error: please supply a destination url")
			}

			r, err := webhook.NewRequest(v, args[0])
			if err != nil {
				return errors.Wrap(err, "creating request")
			}

			if err := r.Do(); err != nil {
				return errors.Wrap(err, "sending request")
			}

			return nil
		},
	}

	cmd.Flags().String("json-payload", "", "Payload to send")
	cmd.Flags().String("directory-payload", "", "Directory to send as a tar archive")
	cmd.Flags().String("secret-namespace", "", "Namespace with the secret containing the state")
	cmd.Flags().String("secret-name", "", "Name of the secret with the state")
	cmd.Flags().String("secret-key", "", "Key in the secret with the state")
	cmd.MarkFlagRequired("json-payload")
	cmd.MarkFlagRequired("directory-payload")
	cmd.MarkFlagRequired("secret-namespace")
	cmd.MarkFlagRequired("secret-name")
	cmd.MarkFlagRequired("secret-key")

	viper.BindPFlags(cmd.Flags())
	viper.BindPFlags(cmd.PersistentFlags())
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}
