package cli

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	cursor "github.com/ahmetalpbalkan/go-cursor"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/upload"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/util/validation"
)

func InstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "install [upstream uri]",
		Short:         "Install an application to a cluster",
		Long:          `Pull Kubernetes manifests from the remote upstream, deploy them to the specified cluster, then setup port forwarding to make the kotsadm admin console accessible.`,
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

			log := logger.NewLogger()

			rootDir, err := ioutil.TempDir("", "kotsadm")
			if err != nil {
				return errors.Wrap(err, "failed to create temp dir")
			}
			defer os.RemoveAll(rootDir)

			upstream := pull.RewriteUpstream(args[0])

			namespace := v.GetString("namespace")
			if namespace != "" {
				if strings.Contains(namespace, "_") {
					return errors.New("a namespace should not contain the _ character")
				}

				errs := validation.IsValidLabelValue(namespace)
				if len(errs) > 0 {
					return errors.New(errs[0])
				}
			} else {
				enteredNamespace, err := promptForNamespace(upstream)
				if err != nil {
					return errors.Wrap(err, "failed to prompt for namespace")
				}

				namespace = enteredNamespace
			}

			kotsadm.OverrideVersion = v.GetString("kotsadm-tag")
			kotsadm.OverrideRegistry = v.GetString("kotsadm-registry")
			kotsadm.OverrideNamespace = v.GetString("kotsadm-namespace")

			pullOptions := pull.PullOptions{
				HelmRepoURI: v.GetString("repo"),
				RootDir:     rootDir,
				Namespace:   namespace,
				Downstreams: []string{
					"this-cluster", // this is the auto-generated operator downstream
				},
				LocalPath:           ExpandDir(v.GetString("local-path")),
				LicenseFile:         ExpandDir(v.GetString("license-file")),
				ExcludeAdminConsole: true,
				ExcludeKotsKinds:    true,
				HelmOptions:         v.GetStringSlice("set"),
				RewriteImages:       v.GetBool("rewrite-images"),
				RewriteImageOptions: pull.RewriteImageOptions{
					Host:      v.GetString("registry-endpoint"),
					Namespace: v.GetString("image-namespace"),
				},
			}

			canPull, err := pull.CanPullUpstream(upstream, pullOptions)
			if err != nil {
				return errors.Wrap(err, "failed to check upstream")
			}

			if canPull {
				if _, err := pull.Pull(upstream, pullOptions); err != nil {
					return errors.Wrap(err, "failed to pull app")
				}
			}

			if !v.GetBool("exclude-admin-console") {
				applicationMetadata, err := pull.PullApplicationMetadata(upstream)
				if err != nil {
					return errors.Wrap(err, "failed to pull app metadata")
				}

				deployOptions := kotsadm.DeployOptions{
					Namespace:           namespace,
					Kubeconfig:          v.GetString("kubeconfig"),
					Context:             v.GetString("context"),
					IncludeShip:         v.GetBool("include-ship"),
					IncludeGitHub:       v.GetBool("include-github"),
					SharedPassword:      v.GetString("shared-password"),
					ServiceType:         v.GetString("service-type"),
					NodePort:            v.GetInt32("node-port"),
					Hostname:            v.GetString("hostname"),
					ApplicationMetadata: applicationMetadata,
				}

				log.ActionWithoutSpinner("Deploying Admin Console")
				if err := kotsadm.Deploy(deployOptions); err != nil {
					return errors.Wrap(err, "failed to deploy")
				}
			}

			// upload the kots app to kotsadm
			uploadOptions := upload.UploadOptions{
				Namespace:   namespace,
				Kubeconfig:  v.GetString("kubeconfig"),
				NewAppName:  v.GetString("name"),
				UpstreamURI: upstream,
				Endpoint:    "http://localhost:3000",
				RegistryOptions: registry.RegistryOptions{
					Endpoint:  v.GetString("registry-endpoint"),
					Namespace: v.GetString("image-namespace"),
				},
			}

			if v.GetString("registry-endpoint") != "" {
				registryUser, registryPass, err := registry.LoadAuthForRegistry(v.GetString("registry-endpoint"))
				if err != nil {
					return errors.Wrap(err, "failed to load registry auth info")
				}
				uploadOptions.RegistryOptions.Username = registryUser
				uploadOptions.RegistryOptions.Password = registryPass
			}

			if canPull {
				stopCh := make(chan struct{})
				defer close(stopCh)

				localPort, errChan, err := upload.StartPortForward(uploadOptions.Namespace, uploadOptions.Kubeconfig, stopCh, log)
				if err != nil {
					return err
				}

				uploadOptions.Endpoint = fmt.Sprintf("http://localhost:%d", localPort)
				go func() {
					select {
					case err := <-errChan:
						if err != nil {
							log.Error(err)
							os.Exit(-1)
						}
					case <-stopCh:
					}
				}()

				if err := upload.Upload(rootDir, uploadOptions); err != nil {
					return err
				}
			}

			// port forward
			podName, err := k8sutil.WaitForWeb(namespace, time.Minute*3)
			if err != nil {
				return errors.Wrap(err, "failed to wait for web")
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			_, errChan, err := k8sutil.PortForward(v.GetString("kubeconfig"), 8800, 3000, namespace, podName, true, stopCh, log)
			if err != nil {
				return errors.Wrap(err, "failed to forward port")
			}

			go func() {
				select {
				case err := <-errChan:
					if err != nil {
						log.Error(err)
						os.Exit(-1)
					}
				case <-stopCh:
				}
			}()

			log.ActionWithoutSpinner("")
			log.ActionWithoutSpinner("Press Ctrl+C to exit")
			log.ActionWithoutSpinner("Go to http://localhost:8800 to access the Admin Console")
			log.ActionWithoutSpinner("")

			signalChan := make(chan os.Signal, 1)
			signal.Notify(signalChan, os.Interrupt)

			<-signalChan

			log.ActionWithoutSpinner("Cleaning up")
			log.ActionWithoutSpinner("")
			log.ActionWithoutSpinner("To access the Admin Console again, run kubectl kots admin-console --namespace %s", namespace)
			log.ActionWithoutSpinner("")

			return nil
		},
	}

	cmd.Flags().String("kubeconfig", defaultKubeConfig(), "the kubeconfig to use")
	cmd.Flags().StringP("namespace", "n", "", "the namespace to deploy to")
	cmd.Flags().Bool("include-ship", false, "include the shipinit/edit/update and watch components")
	cmd.Flags().Bool("include-github", false, "set up for github login")
	cmd.Flags().String("shared-password", "", "shared password to apply")
	cmd.Flags().String("service-type", "ClusterIP", "the service type to create")
	cmd.Flags().Int32("node-port", 0, "the nodeport to assign to the service, when service-type is set to NodePort")
	cmd.Flags().String("hostname", "localhost:8800", "the hostname to that the admin console will be exposed on")
	cmd.Flags().String("name", "", "name of the application to use in the Admin Console")
	cmd.Flags().String("local-path", "", "specify a local-path to test the behavior of rendering a replicated app locally (only supported on replicated app types currently)")
	cmd.Flags().String("license-file", "", "path to a license file to use when download a replicated app")
	cmd.Flags().Bool("exclude-admin-console", false, "set to true to exclude the admin console (replicated apps only)")

	cmd.Flags().String("repo", "", "repo uri to use when installing a helm chart")
	cmd.Flags().StringSlice("set", []string{}, "values to pass to helm when running helm template")

	cmd.Flags().String("kotsadm-tag", "", "set to override the tag of kotsadm. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().String("kotsadm-registry", "", "set to override the registry of kotsadm image. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().String("kotsadm-namespace", "", "set to override the namespace of kotsadm image. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().MarkHidden("kotsadm-tag")
	cmd.Flags().MarkHidden("kotsadm-registry")
	cmd.Flags().MarkHidden("kotsadm-namespace")

	cmd.Flags().Bool("rewrite-images", false, "set to true to force all container images to be rewritten and pushed to a local registry")
	cmd.Flags().String("image-namespace", "", "the namespace/org in the docker registry to push images to (required when --rewrite-images is set)")
	cmd.Flags().String("registry-endpoint", "", "the endpoint of the local docker registry to use when pushing images (required when --rewrite-images is set)")

	return cmd
}

func promptForNamespace(upstreamURI string) (string, error) {
	u, err := url.ParseRequestURI(upstreamURI)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse uri")
	}

	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	prompt := promptui.Prompt{
		Label:     "Enter the namespace to deploy to:",
		Templates: templates,
		Default:   u.Hostname(),
		Validate: func(input string) error {
			if len(input) == 0 {
				return errors.New("invalid namespace")
			}

			if strings.Contains(input, "_") {
				return errors.New("a namespace should not contain the _ character")
			}

			errs := validation.IsValidLabelValue(input)
			if len(errs) > 0 {
				return errors.New(errs[0])
			}

			return nil
		},
	}

	for {
		result, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				os.Exit(-1)
			}
			continue
		}

		return result, nil
	}
}
