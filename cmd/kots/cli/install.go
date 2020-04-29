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
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/pull"
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

			applicationMetadata, err := pull.PullApplicationMetadata(upstream)
			if err != nil {
				log.Info("Unable to pull application metadata. This can be ignored, but custom branding will not be available in the Admin Console until a license is installed.")
			}

			var license *kotsv1beta1.License
			if v.GetString("license-file") != "" {
				parsedLicense, err := pull.ParseLicenseFromFile(ExpandDir(v.GetString("license-file")))
				if err != nil {
					return errors.Wrap(err, "failed to parse license file")
				}

				license = parsedLicense
			}

			var configValues *kotsv1beta1.ConfigValues
			if v.GetString("config-values") != "" {
				parsedConfigValues, err := pull.ParseConfigValuesFromFile(ExpandDir(v.GetString("config-values")))
				if err != nil {
					return errors.Wrap(err, "failed to parse config values")
				}

				configValues = parsedConfigValues
			}

			// alpha enablement here
			// if deploy minio is set and there's no storage base uri, set it
			// this is likely not going to be the final state of how this is configured
			if v.GetBool("deploy-dockerdistribution") {
				if v.GetString("storage-base-uri") == "" {
					v.Set("storage-base-uri", "docker://kotsadm-storage-registry:5000")
					v.Set("storage-base-uri-plainhttp", true)
				}
			}

			deployOptions := kotsadmtypes.DeployOptions{
				Namespace:                 namespace,
				KubernetesConfigFlags:     kubernetesConfigFlags,
				Context:                   v.GetString("context"),
				SharedPassword:            v.GetString("shared-password"),
				ServiceType:               v.GetString("service-type"),
				NodePort:                  v.GetInt32("node-port"),
				ApplicationMetadata:       applicationMetadata,
				License:                   license,
				ConfigValues:              configValues,
				StorageBaseURI:            v.GetString("storage-base-uri"),
				StorageBaseURIPlainHTTP:   v.GetBool("storage-base-uri-plainhttp"),
				IncludeMinio:              v.GetBool("deploy-minio"),
				IncludeDockerDistribution: v.GetBool("deploy-dockerdistribution"),
				Timeout:                   time.Minute * 2,

				KotsadmOptions: kotsadmtypes.KotsadmOptions{
					OverrideVersion:   v.GetString("kotsadm-tag"),
					OverrideRegistry:  v.GetString("kotsadm-registry"),
					OverrideNamespace: v.GetString("kotsadm-namespace"),
					Username:          v.GetString("registry-username"),
					Password:          v.GetString("registry-password"),
				},
			}

			timeout, err := time.ParseDuration(v.GetString("wait-duration"))
			if err != nil {
				return errors.Wrap(err, "failed to parse timeout value")
			}
			objectStoreOptions := kotsadmtypes.StorageOptions{
				ObjectStoreType: v.GetString("object-store"),
				AccessKeyID:     v.GetString("object-store-access-key-id"),
				SecretAccessKey: v.GetString("object-store-secret-access-key"),
				BucketName:      v.GetString("object-store-bucket-name"),
				Endpoint:        v.GetString("object-store-endpoint"),
				Region:          v.GetString("object-store-region"),
				BucketInPath:    v.GetBool("object-store-bucket-in-path"),

				StorageIncludeMinio:              v.GetBool("deploy-minio"),
				StorageIncludeDockerDistribution: v.GetBool("deploy-dockerdistribution"),
			}

			objectStoreConfig, err := kotsadmtypes.NewObjectStoreConfig(objectStoreOptions)
			if err != nil {
				return errors.Wrap(err, "validate object store")
			}

			deployOptions.ObjectStoreOptions = objectStoreConfig
			deployOptions.Timeout = timeout

			log.ActionWithoutSpinner("Deploying Admin Console")
			if err := kotsadm.Deploy(deployOptions); err != nil {
				if _, ok := errors.Cause(err).(*types.ErrorTimeout); ok {
					return errors.Errorf("Failed to deploy: %s. Use the --wait-duration flag to increase timeout.", err)
				}
				return errors.Wrap(err, "failed to deploy")
			}

			// port forward
			clientset, err := k8sutil.GetClientset(kubernetesConfigFlags)
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			podName, err := k8sutil.WaitForKotsadm(clientset, namespace, timeout)
			if err != nil {
				if _, ok := errors.Cause(err).(*types.ErrorTimeout); ok {
					return errors.Errorf("kotsadm failed to start: %s. Use the --wait-duration flag to increase timeout.", err)
				}
				return errors.Wrap(err, "failed to wait for web")
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			adminConsolePort, errChan, err := k8sutil.PortForward(kubernetesConfigFlags, 8800, 3000, namespace, podName, true, stopCh, log)
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

			if v.GetBool("port-forward") {
				log.ActionWithoutSpinner("")

				if adminConsolePort != 8800 {
					log.ActionWithoutSpinner("Port 8800 is not available. The Admin Console is running on port %d", adminConsolePort)
					log.ActionWithoutSpinner("")
				}

				log.ActionWithoutSpinner("Press Ctrl+C to exit")
				log.ActionWithoutSpinner("Go to http://localhost:%d to access the Admin Console", adminConsolePort)
				log.ActionWithoutSpinner("")

				signalChan := make(chan os.Signal, 1)
				signal.Notify(signalChan, os.Interrupt)

				<-signalChan

				log.ActionWithoutSpinner("Cleaning up")
				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner("To access the Admin Console again, run kubectl kots admin-console --namespace %s", namespace)
				log.ActionWithoutSpinner("")
			} else {
				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner("To access the Admin Console, run kubectl kots admin-console --namespace %s", namespace)
				log.ActionWithoutSpinner("")
			}

			return nil
		},
	}

	cmd.Flags().String("shared-password", "", "shared password to apply")
	cmd.Flags().String("service-type", "ClusterIP", "the service type to create")
	cmd.Flags().Int32("node-port", 0, "the nodeport to assign to the service, when service-type is set to NodePort")
	cmd.Flags().String("name", "", "name of the application to use in the Admin Console")
	cmd.Flags().String("local-path", "", "specify a local-path to test the behavior of rendering a replicated app locally (only supported on replicated app types currently)")
	cmd.Flags().String("license-file", "", "path to a license file to use when download a replicated app")
	cmd.Flags().String("config-values", "", "path to a manifest containing config values (must be apiVersion: kots.io/v1beta1, kind: ConfigValues)")
	cmd.Flags().Bool("port-forward", true, "set to false to disable automatic port forward")
	cmd.Flags().String("wait-duration", "2m", "timeout out to be used while waiting for individual components to be ready.  must be in Go duration format (eg: 10s, 2m)")

	cmd.Flags().String("repo", "", "repo uri to use when installing a helm chart")
	cmd.Flags().StringSlice("set", []string{}, "values to pass to helm when running helm template")

	cmd.Flags().String("kotsadm-registry", "", "set to override the registry of kotsadm images")
	cmd.Flags().String("registry-username", "", "user name to use to authenticate with the registry")
	cmd.Flags().String("registry-password", "", "password to use to authenticate with the registry")

	// the following group of flags are useful for testing, but we don't want to pollute the help screen with them
	cmd.Flags().String("kotsadm-tag", "", "set to override the tag of kotsadm. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().String("kotsadm-namespace", "", "set to override the namespace of kotsadm image. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().MarkHidden("kotsadm-tag")
	cmd.Flags().MarkHidden("kotsadm-namespace")

	cmd.Flags().String("object-store", kotsadmtypes.ObjectStoreTypeInternal, "(alpha) determine object store implementation, options are [internal,external]")
	cmd.Flags().String("object-store-access-key-id", "", "(alpha) when object-store=external, set an access key id for authentication")
	cmd.Flags().String("object-store-secret-access-key", "", "(alpha) when object-store=external, set a secret access key for authentication")
	cmd.Flags().String("object-store-bucket-name", "", "(alpha) when object-store=external, set a bucket name to use")
	cmd.Flags().String("object-store-endpoint", "", "(alpha) when object-store=external, set a custom endpoint to use")
	cmd.Flags().String("object-store-region", "us-east-1", "(alpha) when object-store=external, set a custom region to use")
	cmd.Flags().Bool("object-store-bucket-in-path", true, "(alpha) when object-store=external, determine whether to use a bucket name in the URL path vs using DNS-based bucket addressing")
	cmd.Flags().MarkHidden("object-store")
	cmd.Flags().MarkHidden("object-store-access-key-id")
	cmd.Flags().MarkHidden("object-store-secret-access-key")
	cmd.Flags().MarkHidden("object-store-bucket-name")
	cmd.Flags().MarkHidden("object-store-endpoint")
	cmd.Flags().MarkHidden("object-store-region")
	cmd.Flags().MarkHidden("object-store-bucket-in-path")

	// the following group of flags are experiemental and can be used to pull and push images during install time
	cmd.Flags().Bool("rewrite-images", false, "set to true to force all container images to be rewritten and pushed to a local registry")
	cmd.Flags().String("image-namespace", "", "the namespace/org in the docker registry to push images to (required when --rewrite-images is set)")
	cmd.Flags().String("registry-endpoint", "", "the endpoint of the local docker registry to use when pushing images (required when --rewrite-images is set)")
	cmd.Flags().MarkHidden("rewrite-images")
	cmd.Flags().MarkHidden("image-namespace")
	cmd.Flags().MarkHidden("registry-endpoint")

	// flags that are not fully supported or generally available yet
	cmd.Flags().MarkHidden("service-type")
	cmd.Flags().MarkHidden("node-port")

	// options for the alpha feature of using a reg instead of s3 for storage
	// this was introduced as alpha in kots 1.16.0
	cmd.Flags().String("storage-base-uri", "", "an s3 or oci-registry uri to use for kots persistent storage in the cluster")
	cmd.Flags().Bool("deploy-minio", true, "when set, kots install will deploy a local minio instance for storage")
	cmd.Flags().Bool("deploy-dockerdistribution", false, "when set, kots install will deploy a local instance of docker distribution for storage")
	cmd.Flags().Bool("storage-base-uri-plainhttp", false, "when set, use plain http (not https) connecting to the local oci storage")
	cmd.Flags().MarkHidden("storage-base-uri")
	cmd.Flags().MarkHidden("deploy-minio")
	cmd.Flags().MarkHidden("deploy-dockerdistribution")
	cmd.Flags().MarkHidden("storage-base-uri-plainhttp")

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
