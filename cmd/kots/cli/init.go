package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/upstream"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func InitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "init [upstream uri]",
		Short:         "",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
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

			writeUpstreamOptions := upstream.WriteOptions{
				RootDir:      v.GetString("rootdir"),
				CreateAppDir: true,
				Overwrite:    v.GetBool("overwrite"),
			}
			if err := u.WriteUpstream(writeUpstreamOptions); err != nil {
				return err
			}

			renderOptions := base.RenderOptions{
				SplitMultiDocYAML: true,
				Namespace:         v.GetString("namespace"),
			}
			b, err := base.RenderUpstream(u, &renderOptions)
			if err != nil {
				return err
			}

			writeBaseOptions := base.WriteOptions{
				BaseDir:   u.GetBaseDir(writeUpstreamOptions),
				Overwrite: v.GetBool("overwrite"),
			}
			if err := b.WriteBase(writeBaseOptions); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringArray("set", []string{}, "values to pass to helm when running helm template")
	cmd.Flags().String("repo", "", "repo name=uri to use when downloading a helm chart")
	cmd.Flags().String("rootdir", homeDir(), "root directory that will be used to write the yaml to")
	cmd.Flags().Bool("overwrite", false, "if the upstream already exists, overwrite it")
	cmd.Flags().String("namespace", "default", "namespace to render the upstream to in the base")

	return cmd
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}
