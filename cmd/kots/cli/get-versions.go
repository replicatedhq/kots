package cli

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func GetVersionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "versions",
		Aliases:       []string{"versions"},
		Short:         "Get App Versions",
		Long:          "",
		SilenceUsage:  false,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: getVersionsCmd,
	}

	cmd.Flags().StringP("output", "o", "", "output format. supported values: json")
	cmd.Flags().String("appslug", "", "app slug to retrieve config for")

	return cmd
}

func getVersionsCmd(cmd *cobra.Command, args []string) error {
	v := viper.GetViper()

	appSlug := v.GetString("appslug")
	if appSlug == "" {
		return errors.New("appslug is required")
	}

	appVersions, err := store.GetStore().GetAppVersionsAfter(appSlug, 0)
	if err != nil {
		return errors.Wrap(err, "failed to list app versions")
	}

	versionJson, err := json.Marshal(appVersions)
	if err != nil {
		return errors.Wrap(err, "failed to marshal config")
	}

	fmt.Print(string(versionJson))

	/* options := snapshot.ListInstanceBackupsOptions{
		Namespace: v.GetString("namespace"),
	}
	backups, err := snapshot.ListInstanceBackups(cmd.Context(), options)
	if err != nil {
		return errors.Wrap(err, "failed to list instance backups")
	}

	print.Backups(backups, v.GetString("output")) */

	return nil
}
