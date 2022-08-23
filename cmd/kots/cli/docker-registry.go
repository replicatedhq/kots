package cli

import (
	"os"

	"github.com/pkg/errors"

	"github.com/distribution/distribution/v3/configuration"
	distributionregistry "github.com/distribution/distribution/v3/registry"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/filesystem" // this initializes the filesystem storage driver
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func DockerRegistryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "docker-registry",
		Short:  "KOTS Docker Registry interface",
		Hidden: true,
	}

	cmd.AddCommand(DockerRegistryServeCmd())

	return cmd
}

func DockerRegistryServeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "serve [config]",
		Short:         "Starts a docker registry service using the provided config.",
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

			configurationPath := args[0]

			fp, err := os.Open(configurationPath)
			if err != nil {
				return errors.Wrapf(err, "failed to open configuration file %s", configurationPath)
			}
			defer fp.Close()

			config, err := configuration.Parse(fp)
			if err != nil {
				return errors.Wrap(err, "failed to parse registry config yaml")
			}

			reg, err := distributionregistry.NewRegistry(cmd.Context(), config)
			if err != nil {
				return errors.Wrap(err, "failed to initialize registry")
			}

			if err := reg.ListenAndServe(); err != nil {
				return errors.Wrap(err, "failed to start registry")
			}

			return nil
		},
	}

	return cmd
}
