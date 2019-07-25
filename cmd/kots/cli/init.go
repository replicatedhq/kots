package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/replicatedhq/kots/pkg/upstream"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func InitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [upstream uri]",
		Short: "",
		Long:  ``,
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if len(args) == 0 {
				cmd.Help()
				os.Exit(1)
			}

			fetchOptions := upstream.FetchOptions{}
			if v.GetString("repo") != "" {
				repoSplit := strings.Split(v.GetString("repo"), "=")
				if len(repoSplit) != 2 {
					return fmt.Errorf("failed to parse %q as a name=uri format", v.GetString("repo"))
				}

				fetchOptions.HelmRepoName = repoSplit[0]
				fetchOptions.HelmRepoURI = repoSplit[1]
			}

			upstream, err := upstream.FetchUpstream(args[0], &fetchOptions)
			if err != nil {
				return err
			}

			// validating
			for _, f := range upstream.Files {
				if strings.Contains(f.Path, "Chart.yaml") {
					fmt.Printf("%s\n", f.Content)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringArray("set", []string{}, "values to pass to helm when running helm template")
	cmd.Flags().String("repo", "", "repo name=uri to use when downloading a helm chart")

	return cmd
}
