package cli

import (
	"io/ioutil"
	"os"
	"path"

	"github.com/replicatedhq/kots/pkg/helm"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func MakeHelmChartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "make-helm-chart [upstream uri]",
		Short:         "",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: true,
		Hidden:        true, // Because this might be a bad idea
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if len(args) == 0 {
				cmd.Help()
				os.Exit(1)
			}

			// registry host should not have the scheme (https).  need to
			// strip it if included or else the rewrite images will fail

			// pull into a temp dir
			pullDir, err := ioutil.TempDir("", "kots")
			if err != nil {
				return err
			}
			pullOptions := pull.PullOptions{
				HelmRepoURI:         v.GetString("repo"),
				RootDir:             pullDir,
				Namespace:           v.GetString("namespace"),
				Downstreams:         v.GetStringSlice("downstream"),
				LocalPath:           ExpandDir(v.GetString("local-path")),
				LicenseFile:         ExpandDir(v.GetString("license-file")),
				ExcludeKotsKinds:    v.GetBool("exclude-kots-kinds"),
				ExcludeAdminConsole: v.GetBool("exclude-admin-console"),
				SharedPassword:      v.GetString("shared-password"),
				CreateAppDir:        true,
				HelmOptions:         v.GetStringSlice("set"),
				RewriteImages:       v.GetBool("rewrite-images"),
				RewriteImageOptions: pull.RewriteImageOptions{
					Host:      v.GetString("registry-endpoint"),
					Namespace: v.GetString("image-namespace"),
				},
			}
			renderDir, err := pull.Pull(args[0], pullOptions)
			if err != nil {
				return err
			}
			makeHelmChartOptions := helm.MakeHelmChartOptions{
				KotsAppDir:       renderDir,
				KustomizationDir: path.Join(renderDir, "overlays", "midstream"),
				RenderDir:        ExpandDir(v.GetString("rootdir")),
			}
			err = helm.MakeHelmChart(makeHelmChartOptions)
			if err != nil {
				return err
			}

			log := logger.NewLogger()
			log.Initialize()

			_, name := path.Split(renderDir)
			chartPath := path.Join(ExpandDir(v.GetString("rootdir")), name)
			log.Info("Helm chart created in %s", chartPath)

			return nil
		},
	}

	cmd.Flags().StringSlice("set", []string{}, "values to pass to helm when running helm template")
	cmd.Flags().String("repo", "", "repo uri to use when downloading a helm chart")
	cmd.Flags().String("rootdir", homeDir(), "root directory that will be used to write the yaml to")
	cmd.Flags().String("namespace", "default", "namespace to render the upstream to in the base")
	cmd.Flags().StringSlice("downstream", []string{}, "the list of any downstreams to create/update")
	cmd.Flags().String("local-path", "", "specify a local-path to pull a locally available replicated app (only supported on replicated app types currently)")
	cmd.Flags().String("license-file", "", "path to a license file to use when download a replicated app")
	cmd.Flags().Bool("exclude-kots-kinds", true, "set to true to exclude rendering kots custom objects to the base directory")
	cmd.Flags().Bool("exclude-admin-console", false, "set to true to exclude the admin console (replicated apps only)")
	cmd.Flags().String("shared-password", "", "shared password to use when deploying the admin console")
	cmd.Flags().Bool("rewrite-images", false, "set to true to force all container images to be rewritten and pushed to a local registry")
	cmd.Flags().String("image-namespace", "", "the namespace/org in the docker registry to push images to (required when --rewrite-images is set)")
	cmd.Flags().String("registry-endpoint", "", "the endpoint of the local docker registry to use when pushing images (required when --rewrite-images is set)")

	return cmd
}
