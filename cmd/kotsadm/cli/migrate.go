package cli

import (
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/filestore"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func MigrateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Trigger a migration",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}

	cmd.AddCommand(MigrateS3ToRqliteCmd())
	cmd.AddCommand(MigratePVCToRqliteCmd())

	return cmd
}

func MigrateS3ToRqliteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "s3-to-rqlite",
		Short:         "Migrate object storage from S3 to rqlite",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if required env vars are set
			if os.Getenv("RQLITE_URI") == "" {
				return errors.New("RQLITE_URI is not set")
			}
			if os.Getenv("S3_ENDPOINT") == "" {
				return errors.New("S3_ENDPOINT is not set")
			}
			if os.Getenv("S3_BUCKET_NAME") == "" {
				return errors.New("S3_BUCKET_NAME is not set")
			}
			if os.Getenv("S3_ACCESS_KEY_ID") == "" {
				return errors.New("S3_ACCESS_KEY_ID is not set")
			}
			if os.Getenv("S3_SECRET_ACCESS_KEY") == "" {
				return errors.New("S3_SECRET_ACCESS_KEY is not set")
			}

			// Migrate from S3 to rqlite
			if err := filestore.MigrateFromS3ToRqlite(cmd.Context()); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}

func MigratePVCToRqliteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "pvc-to-rqlite",
		Short:         "Migrate object storage from PVC to rqlite",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if required env vars are set
			if os.Getenv("RQLITE_URI") == "" {
				return errors.New("RQLITE_URI is not set")
			}

			// Check if PVC mount and the archives dir exist
			if _, err := os.Stat(filestore.ArchivesDir); err != nil {
				return errors.Wrap(err, "failed to stat archives dir")
			}

			// Migrate from PVC to rqlite
			if err := filestore.MigrateFromPVCToRqlite(cmd.Context()); err != nil {
				return err
			}

			return nil
		},
	}

	return cmd
}
