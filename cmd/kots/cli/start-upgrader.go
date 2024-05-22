package cli

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/upgrader"
	"github.com/replicatedhq/kots/pkg/upgrader/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// TODO NOW: rename to updater?
func StartUpgraderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start-upgrader",
		Short: "Starts the KOTS upgrader service",
		Long:  `Starts the KOTS upgrader service`,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			params := types.ServerParams{
				Port: fmt.Sprintf("%d", viper.GetInt("port")),

				AppID:       v.GetString("app-id"),
				AppSlug:     v.GetString("app-slug"),
				AppIsAirgap: v.GetBool("app-is-airgap"),
				AppIsGitOps: v.GetBool("app-is-gitops"),
				AppLicense:  v.GetString("app-license"),

				BaseArchive:  v.GetString("base-archive"),
				BaseSequence: v.GetInt64("base-sequence"),
				NextSequence: v.GetInt64("next-sequence"),

				UpdateCursor: v.GetString("update-cursor"),

				RegistryEndpoint:   v.GetString("registry-endpoint"),
				RegistryUsername:   v.GetString("registry-username"),
				RegistryPassword:   v.GetString("registry-password"),
				RegistryNamespace:  v.GetString("registry-namespace"),
				RegistryIsReadOnly: v.GetBool("registry-is-readonly"),
			}
			if err := upgrader.Serve(params); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().IntP("port", "p", 30000, "local port to listen on")

	// app flags
	cmd.Flags().String("app-id", "", "the app id")
	cmd.Flags().String("app-slug", "", "the app slug")
	cmd.Flags().Bool("app-is-airgap", false, "whether the app is airgap")
	cmd.Flags().Bool("app-is-gitops", false, "whether the app is gitops")
	cmd.Flags().String("app-license", "", "the app license")

	// app version flags
	cmd.Flags().String("base-archive", "", "path to the base app version archive")
	cmd.Flags().String("base-sequence", "", "the local base app version sequence")
	cmd.Flags().String("next-sequence", "", "the local next app version sequence")

	// release flags
	cmd.Flags().String("update-cursor", "", "the channel sequence of the target release")

	// registry flags
	cmd.Flags().String("registry-endpoint", "", "the registry endpoint")
	cmd.Flags().String("registry-username", "", "the registry username")
	cmd.Flags().String("registry-password", "", "the registry password")
	cmd.Flags().String("registry-namespace", "", "the registry namespace")
	cmd.Flags().Bool("registry-is-readonly", false, "whether the registry is read-only")

	return cmd
}
