package cli

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type VersionOutput struct {
	Version       string `json:"version"`
	LatestVersion string `json:"latestVersion,omitempty"`
	InstallLatest string `json:"installLatest,omitempty"`
}

func VersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the current version and exit",
		Long:  `Print the current version and exit`,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			output := v.GetString("output")

			isLatest, latestVer, err := buildversion.IsLatestRelease()
			versionOutput := VersionOutput{
				Version: buildversion.Version(),
			}
			if err == nil && !isLatest {
				versionOutput.LatestVersion = latestVer
				versionOutput.InstallLatest = "curl https://kots.io/install | bash"
			}

			if output != "json" && output != "" {
				return errors.Errorf("output format %s not supported (allowed formats are: json)", output)
			} else if output == "json" {
				// marshal JSON
				outputJSON, err := json.Marshal(versionOutput)
				if err != nil {
					return errors.Wrap(err, "error marshaling JSON")
				}
				fmt.Println(string(outputJSON))
			} else {
				// print basic version info
				fmt.Printf("Replicated KOTS %s\n", buildversion.Version())

				// check if this is the latest release, and display possible upgrade instructions
				if versionOutput.LatestVersion != "" {
					fmt.Printf("\nVersion %s is available for kots. To install updates, run\n  $ %s\n", versionOutput.LatestVersion, versionOutput.InstallLatest)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringP("output", "o", "", "output format (currently supported: json)")

	return cmd
}
