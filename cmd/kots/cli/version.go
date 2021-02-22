package cli

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/spf13/cobra"
)

func VersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the current version and exit",
		Long:  `Print the current version and exit`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// print basic version info
			fmt.Printf("Replicated Kots %s\n", buildversion.Version())

			// check if this is the latest release, and display possible upgrade instructions
			isLatest, latestVer, err := buildversion.IsLatestRelease()
			if err == nil && !isLatest {
				fmt.Printf("\nVersion %s is available for kots. To install updates, run\n  $ curl https://kots.io/install | bash\n", latestVer)
			}

			return nil
		},
	}
	return cmd
}
