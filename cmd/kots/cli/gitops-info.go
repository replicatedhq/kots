package cli

import (
	"fmt"
	"os"

	"github.com/replicatedhq/kots/pkg/gitops"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func GitopsInfoCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "gitops-info",
		Short:         "Print the public and private keys kotsadm uses for gitops",
		SilenceUsage:  true,
		SilenceErrors: false,
		Hidden:        true,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()
			log := logger.NewLogger()

			namespace := v.GetString("namespace")
			if namespace == "" {
				log.Info("A namespace is required")
				os.Exit(1)
			}

			keypairs, err := gitops.GetGitopsKeypairs(namespace)
			if err != nil {
				log.Error(err)
				os.Exit(1)
			}

			for provider, keys := range keypairs {
				fmt.Printf("Provider: %s\n", provider)
				fmt.Printf("public key:\n%s", keys.PublicKeySSH)
				fmt.Printf("private key:\n%s", keys.PrivateKeyPEM)
			}

			return nil
		},
	}

	return cmd
}
