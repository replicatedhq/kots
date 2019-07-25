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

			u, err := upstream.FetchUpstream(args[0], &fetchOptions)
			if err != nil {
				return err
			}

			writeOptions := upstream.WriteOptions{
				RootDir:         v.GetString("rootdir"),
				CreateAppDir:    true,
				DeleteIfPresent: true,
			}
			if err := u.WriteUpstream(writeOptions); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringArray("set", []string{}, "values to pass to helm when running helm template")
	cmd.Flags().String("repo", "", "repo name=uri to use when downloading a helm chart")
	cmd.Flags().String("rootdir", homeDir(), "root directory that will be used to write the yaml to")

	return cmd
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}
