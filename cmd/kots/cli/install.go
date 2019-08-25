package cli

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path"
	"path/filepath"

	"github.com/ahmetalpbalkan/go-cursor"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/logger"
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

			fmt.Print(cursor.Hide())
			defer fmt.Print(cursor.Show())

			rootDir, err := ioutil.TempDir("", "kotsadm")
			if err != nil {
				return err
			}
			// defer os.RemoveAll(rootDir)

			pullOptions := pull.PullOptions{
				HelmRepoURI: v.GetString("repo"),
				RootDir:     rootDir,
				Overwrite:   false,
				Namespace:   v.GetString("namespace"),
				Downstreams: []string{
					"local", // this is the auto-generated operator downstream
				},
				LocalPath:   ExpandDir(v.GetString("local-path")),
				LicenseFile: ExpandDir(v.GetString("license-file")),
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

			log := logger.NewLogger()
			log.ActionWithoutSpinner("Deploying Admin Console")
			if err := kotsadm.Deploy(deployOptions); err != nil {
				return err
			}

			// upload the kots app to kotsadm
			uploadOptions := upload.UploadOptions{
				Namespace:    v.GetString("namespace"),
				Kubeconfig:   v.GetString("kubeconfig"),
				NewAppName:   v.GetString("name"),
				VersionLabel: "todo",
			}

			// get the first dir in rootDir and use that as the upload dir
			subdirs, err := ioutil.ReadDir(rootDir)
			if err != nil {
				return err
			}
			uploadRootDir := ""
			for _, subdir := range subdirs {
				if subdir.IsDir() {
					if subdir.Name() == "." {
						continue
					}
					if subdir.Name() == ".." {
						continue
					}

					uploadRootDir = path.Join(rootDir, subdir.Name())
					break
				}
			}
			if uploadRootDir == "" {
				return errors.New("unable to find directory in rootDir")
			}

			if err := upload.Upload(uploadRootDir, uploadOptions); err != nil {
				return errors.Cause(err)
			}

			// port forward
			podName, err := k8sutil.WaitForWeb(v.GetString("namespace"))
			if err != nil {
				return err
			}

			stopCh, err := k8sutil.PortForward(v.GetString("kubeconfig"), 8800, 3000, v.GetString("namespace"), podName)
			if err != nil {
				return err
			}
			defer close(stopCh)

			log.ActionWithoutSpinner("")
			log.ActionWithoutSpinner("Press Ctrl+C to exit")
			log.ActionWithoutSpinner("Go to http://localhost:8800 to access the Admin Console")
			log.ActionWithoutSpinner("")

			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, os.Interrupt)

			<-signalChan

			log.ActionWithoutSpinner("Cleaning up")
			log.ActionWithoutSpinner("")
			log.ActionWithoutSpinner("To access the Admin Console again, run kubectl kots admin-console %s", v.GetString("namespace"))
			log.ActionWithoutSpinner("")

			return nil
		},
	}

	cmd.Flags().String("kubeconfig", filepath.Join(homeDir(), ".kube", "config"), "the kubeconfig to use")
	cmd.Flags().String("namespace", "default", "the namespace to deploy to")
	cmd.Flags().Bool("include-ship", false, "include the shipinit/edit/update and watch components")
	cmd.Flags().Bool("include-github", false, "set up for github login")
	cmd.Flags().String("shared-password", "", "shared password to apply")
	cmd.Flags().String("service-type", "ClusterIP", "the service type to create")
	cmd.Flags().Int32("node-port", 0, "the nodeport to assign to the service, when service-type is set to NodePort")
	cmd.Flags().String("hostname", "localhost:8800", "the hostname to that the admin console will be exposed on")
	cmd.Flags().StringP("name", "n", "", "name of the application to use in the Admin Console")
	cmd.Flags().String("local-path", "", "specify a local-path to test the behavior of rendering a replicated app locally (only supported on replicated app types currently)")
	cmd.Flags().String("license-file", "", "path to a license file to use when download a replicated app")

	cmd.Flags().String("repo", "", "repo uri to use when installing a helm chart")
	cmd.Flags().StringArray("set", []string{}, "values to pass to helm when running helm template")

	return cmd
}
