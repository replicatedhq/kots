package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/replicatedhq/kots/pkg/version"
)

func VersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the current version and exit",
		Long:  `Print the current version and exit`,
		RunE: func(cmd *cobra.Command, args []string) error {
			build := version.GetBuild()
			versionInfo, err := json.MarshalIndent(build, "", "    ")
			if err != nil {
				return err
			}
			fmt.Println(string(versionInfo))
			return nil
		},
	}
	return cmd
}
