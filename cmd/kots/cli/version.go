package cli

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/replicatedhq/kots/pkg/version"
)

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

			if v.GetBool("verbose") {
				// return full build info and exit
				build := version.GetLatestVersionBuild()
				versionInfo, err := json.MarshalIndent(build, "", "    ")
				if err != nil {
					return err
				}
				fmt.Println(string(versionInfo))
				return nil
			}

			// print basic version info
			fmt.Printf("Version %s built at %s\n", version.Version(), version.BuildTime().Format(time.RFC3339))

			// check if this is the latest release, and display possible upgrade instructions
			isLatest, latestVer, err := version.IsLatestRelease()
			if err != nil {
				fmt.Printf("Unable to check for newer releases: %s\n", err.Error())
			} else if isLatest {
				fmt.Printf("This is the latest version\n")
			} else {
				fmt.Printf("There is a newer version available, %s\n", latestVer)
			}

			return nil
		},
	}

	cmd.Flags().BoolP("verbose", "v", false, "include verbose version information")

	return cmd
}
