package cli

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/upload"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func InstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "install [upstream uri]",
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

			rootDir, err := ioutil.TempDir("", "kotsadm")
			if err != nil {
				return err
			}
			defer os.RemoveAll(rootDir)

			pullOptions := pull.PullOptions{
				HelmRepoURI: v.GetString("repo"),
				RootDir:     rootDir,
				Overwrite:   false,
				Namespace:   v.GetString("namespace"),
				Downstreams: []string{
					"local", // this is the auto-generated operator downstream
				},
			}
			if err := pull.Pull(args[0], pullOptions); err != nil {
				return err
			}

			deployOptions := kotsadm.DeployOptions{
				Namespace:      v.GetString("namespace"),
				Kubeconfig:     v.GetString("kubeconfig"),
				IncludeShip:    v.GetBool("include-ship"),
				IncludeGitHub:  v.GetBool("include-github"),
				SharedPassword: v.GetString("shared-password"),
				ServiceType:    v.GetString("service-type"),
				NodePort:       v.GetInt32("node-port"),
				Hostname:       v.GetString("hostname"),
			}
			if err := kotsadm.Deploy(deployOptions); err != nil {
				return err
			}

			// upload the kots app to kotsadm
			uploadOptions := upload.UploadOptions{
				Namespace:  v.GetString("namespace"),
				Kubeconfig: v.GetString("kubeconfig"),
			}

			if err := upload.Upload(rootDir, uploadOptions); err != nil {
				return errors.Cause(err)
			}

			return nil
		},
	}

	cmd.Flags().String("kubeconfig", filepath.Join(homeDir(), ".kube", "config"), "the kubeconfig to use")
	cmd.Flags().String("namespace", "default", "the namespace to deploy to")
	cmd.Flags().Bool("include-ship", false, "include the shipinit/edit/update and watch components")
	cmd.Flags().Bool("include-github", false, "set up for github login")
	cmd.Flags().String("shared-password", "", "shared password to apply")
	cmd.Flags().String("service-type", "", "the service type to create")
	cmd.Flags().Int32("node-port", 0, "the nodeport to assign to the service, when service-type is set to NodePort")
	cmd.Flags().String("hostname", "", "the hostname to that the admin console will be exposed on")

	cmd.Flags().String("repo", "", "repo uri to use when installing a helm chart")
	cmd.Flags().StringArray("set", []string{}, "values to pass to helm when running helm template")

	return cmd
}
