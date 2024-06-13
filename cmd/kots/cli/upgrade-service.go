package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/replicatedhq/kots/pkg/upgradeservice"
	"github.com/replicatedhq/kots/pkg/upgradeservice/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

func UpgradeServiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "upgrade-service",
		Short:  "KOTS Upgrade Service",
		Hidden: true,
	}

	cmd.AddCommand(UpgradeServiceStartCmd())

	return cmd
}

func UpgradeServiceStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "start [params-file]",
		Short:         "Starts a KOTS upgrade service using the provided params file",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				cmd.Help()
				os.Exit(1)
			}

			paramsYAML, err := readUpgradeServiceParams(args[0])
			if err != nil {
				return fmt.Errorf("failed to read config file: %v", err)
			}

			var params types.UpgradeServiceParams
			if err := yaml.Unmarshal(paramsYAML, &params); err != nil {
				return fmt.Errorf("failed to unmarshal config file: %v", err)
			}

			if err := upgradeservice.Serve(params); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func readUpgradeServiceParams(path string) ([]byte, error) {
	if path == "-" {
		b, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, fmt.Errorf("read stdin: %w", err)
		}
		return b, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}
	return b, nil
}
