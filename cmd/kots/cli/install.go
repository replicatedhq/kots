package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	cursor "github.com/ahmetalpbalkan/go-cursor"
	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/identity"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/metrics"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
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
		RunE: func(cmd *cobra.Command, args []string) (finalError error) {
			v := viper.GetViper()

			if len(args) == 0 {
				cmd.Help()
				os.Exit(1)
			}

			fmt.Print(cursor.Hide())
			defer fmt.Print(cursor.Show())

			log := logger.NewCLILogger()

			signalChan := make(chan os.Signal, 1)

			finalMessage := ""
			go func() {
				signal.Notify(signalChan, os.Interrupt)
				<-signalChan

				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner("Cleaning up")
				if finalMessage != "" {
					log.ActionWithoutSpinner("")
					log.ActionWithoutSpinner(finalMessage)
					log.ActionWithoutSpinner("")
				}

				fmt.Print(cursor.Show())
				os.Exit(0)
			}()

			if !v.GetBool("skip-rbac-check") && v.GetBool("ensure-rbac") {
				err := CheckRBAC()
				if err == RBACError {
					log.Errorf("Current user has insufficient privileges to install Admin Console.\nFor more information, please visit https://kots.io/vendor/packaging/rbac\nTo bypass this check, use the --skip-rbac-check flag")
					return errors.New("insufficient privileges")
				} else if err != nil {
					return errors.Wrap(err, "failed to check RBAC")
				}
			}

			license, err := getLicense(v)
			if err != nil {
				return errors.Wrap(err, "failed to get license")
			}

			registryConfig, err := getRegistryConfig(v)
			if err != nil {
				return errors.Wrap(err, "failed to get registry config")
			}

			isAirgap := false
			if v.GetString("airgap-bundle") != "" || v.GetBool("airgap") {
				isAirgap = true
			}

			disableOutboundConnections := registryConfig.OverrideRegistry != "" || isAirgap

			m := metrics.InitInstallMetrics(license, disableOutboundConnections)
			m.ReportInstallStart()

			// only handle reporting install failures in a defer statement.
			// install finish is reported at the end of the function since the function might not exist because of port forwarding.
			defer func() {
				if finalError != nil {
					cause := strings.Split(finalError.Error(), ":")[0]
					m.ReportInstallFail(cause)
				}
			}()

			upstream := pull.RewriteUpstream(args[0])

			namespace := v.GetString("namespace")

			if namespace == "" {
				enteredNamespace, err := promptForNamespace(upstream)
				if err != nil {
					return errors.Wrap(err, "failed to prompt for namespace")
				}

				namespace = enteredNamespace
			}
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			var applicationMetadata []byte
			if airgapBundle := v.GetString("airgap-bundle"); airgapBundle != "" {
				applicationMetadata, err = pull.GetAppMetadataFromAirgap(airgapBundle)
				if err != nil {
					return errors.Wrapf(err, "failed to get metadata from %s", airgapBundle)
				}
			} else if !v.GetBool("airgap") {
				applicationMetadata, err = pull.PullApplicationMetadata(upstream)
				if err != nil {
					log.Info("Unable to pull application metadata. This can be ignored, but custom branding will not be available in the Admin Console until a license is installed. This may also cause the Admin Console to run without minimal role-based-access-control (RBAC) privileges, which may be required by the application.")
				}
			}

			// checks kots version compatibility with the app
			if len(applicationMetadata) > 0 && !v.GetBool("skip-compatibility-check") {
				kotsApp, err := kotsutil.LoadKotsAppFromContents(applicationMetadata)
				if err != nil {
					return errors.Wrap(err, "failed to load kots app from metadata")
				}
				if kotsApp != nil {
					isCompatible := kotsutil.IsKotsVersionCompatibleWithApp(*kotsApp, true)
					if !isCompatible {
						return errors.New(kotsutil.GetIncompatbileKotsVersionMessage(*kotsApp, true))
					}
				}
			}

			var configValues *kotsv1beta1.ConfigValues
			if filepath := v.GetString("config-values"); filepath != "" {
				parsedConfigValues, err := pull.ParseConfigValuesFromFile(ExpandDir(filepath))
				if err != nil {
					return errors.Wrap(err, "failed to parse config values")
				}

				configValues = parsedConfigValues
			}

			// alpha enablement here
			// if deploy minio is set and there's no storage base uri, set it
			// this is likely not going to be the final state of how this is configured
			if v.GetBool("with-dockerdistribution") {
				if v.GetString("storage-base-uri") == "" {
					v.Set("storage-base-uri", "docker://kotsadm-storage-registry:5000")
					v.Set("storage-base-uri-plainhttp", true)
				}
			}

			isKurl, err := kotsadm.IsKurl()
			if err != nil {
				return errors.Wrap(err, "failed to check kURL")
			}

			sharedPassword := v.GetString("shared-password")

			ingressConfig, err := getIngressConfig(v)
			if err != nil {
				return errors.Wrap(err, "failed to get ingress spec")
			}

			identityConfig, err := getIdentityConfig(v)
			if err != nil {
				return errors.Wrap(err, "failed to get identity spec")
			}

			if identityConfig.Spec.Enabled {
				if err := identity.ValidateConfig(cmd.Context(), namespace, *identityConfig, *ingressConfig); err != nil {
					return errors.Wrap(err, "failed to validate identity config")
				}
			}

			simultaneousUploads, _ := strconv.Atoi(v.GetString("airgap-upload-parallelism"))

			deployOptions := kotsadmtypes.DeployOptions{
				Namespace:                 namespace,
				Context:                   v.GetString("context"),
				SharedPassword:            sharedPassword,
				ApplicationMetadata:       applicationMetadata,
				UpstreamURI:               upstream,
				License:                   license,
				ConfigValues:              configValues,
				Airgap:                    isAirgap,
				ProgressWriter:            os.Stdout,
				StorageBaseURI:            v.GetString("storage-base-uri"),
				StorageBaseURIPlainHTTP:   v.GetBool("storage-base-uri-plainhttp"),
				IncludeDockerDistribution: v.GetBool("with-dockerdistribution"),
				Timeout:                   time.Minute * 2,
				HTTPProxyEnvValue:         v.GetString("http-proxy"),
				HTTPSProxyEnvValue:        v.GetString("https-proxy"),
				NoProxyEnvValue:           v.GetString("no-proxy"),
				SkipPreflights:            v.GetBool("skip-preflights"),
				SkipCompatibilityCheck:    v.GetBool("skip-compatibility-check"),
				EnsureRBAC:                v.GetBool("ensure-rbac"),
				SkipRBACCheck:             v.GetBool("skip-rbac-check"),
				InstallID:                 m.InstallID,
				SimultaneousUploads:       simultaneousUploads,
				DisableImagePush:          v.GetBool("disable-image-push"),
				AirgapBundle:              v.GetString("airgap-bundle"),
				IncludeMinio:              v.GetBool("with-minio"),
				IncludeMinioSnapshots:     v.GetBool("with-minio"),
				StrictSecurityContext:     v.GetBool("strict-security-context"),

				KotsadmOptions: *registryConfig,

				IdentityConfig: *identityConfig,
				IngressConfig:  *ingressConfig,
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}
			deployOptions.IsOpenShift = k8sutil.IsOpenShift(clientset)

			timeout, err := time.ParseDuration(v.GetString("wait-duration"))
			if err != nil {
				return errors.Wrap(err, "failed to parse timeout value")
			}
			deployOptions.Timeout = timeout

			if v.GetBool("copy-proxy-env") {
				deployOptions.HTTPProxyEnvValue = os.Getenv("HTTP_PROXY")
				if deployOptions.HTTPProxyEnvValue == "" {
					deployOptions.HTTPProxyEnvValue = os.Getenv("http_proxy")
				}
				deployOptions.HTTPSProxyEnvValue = os.Getenv("HTTPS_PROXY")
				if deployOptions.HTTPSProxyEnvValue == "" {
					deployOptions.HTTPSProxyEnvValue = os.Getenv("https_proxy")
				}
				deployOptions.NoProxyEnvValue = os.Getenv("NO_PROXY")
				if deployOptions.NoProxyEnvValue == "" {
					deployOptions.NoProxyEnvValue = os.Getenv("no_proxy")
				}
			}

			if isKurl && deployOptions.Namespace == metav1.NamespaceDefault {
				deployOptions.ExcludeAdminConsole = true
				deployOptions.EnsureKotsadmConfig = true
			}

			if airgapArchive := v.GetString("airgap-bundle"); airgapArchive != "" {
				if deployOptions.License == nil {
					return errors.New("license is required when airgap bundle is specified")
				}

				log.ActionWithoutSpinner("Extracting airgap bundle")

				airgapRootDir, err := ioutil.TempDir("", "kotsadm-airgap")
				if err != nil {
					return errors.Wrap(err, "failed to create temp dir")
				}
				defer os.RemoveAll(airgapRootDir)

				err = kotsadm.ExtractAppAirgapArchive(airgapArchive, airgapRootDir, v.GetBool("disable-image-push"), deployOptions.ProgressWriter)
				if err != nil {
					return errors.Wrap(err, "failed to extract images")
				}

				deployOptions.AirgapRootDir = airgapRootDir
			}

			log.ActionWithoutSpinner("Deploying Admin Console")
			if err := kotsadm.Deploy(deployOptions); err != nil {
				if _, ok := errors.Cause(err).(*types.ErrorTimeout); ok {
					return errors.Errorf("Failed to deploy: %s. Use the --wait-duration flag to increase timeout.", err)
				}
				return errors.Wrap(err, "failed to deploy")
			}

			if deployOptions.ExcludeAdminConsole && sharedPassword != "" {
				if err := setKotsadmPassword(sharedPassword, namespace); err != nil {
					return errors.Wrap(err, "failed to set new password")
				}
			}

			// port forward
			getPodName := func() (string, error) {
				podName, err := k8sutil.WaitForKotsadm(clientset, namespace, timeout)
				if err != nil {
					if _, ok := errors.Cause(err).(*types.ErrorTimeout); ok {
						return podName, errors.Errorf("kotsadm failed to start: %s. Use the --wait-duration flag to increase timeout.", err)
					}
					return podName, errors.Wrap(err, "failed to wait for web")
				}
				return podName, nil
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			adminConsolePort, errChan, err := k8sutil.PortForward(8800, 3000, namespace, getPodName, true, stopCh, log)
			if err != nil {
				return errors.Wrap(err, "failed to forward port")
			}

			if deployOptions.AirgapRootDir != "" {
				log.ActionWithoutSpinner("Uploading app archive")

				var tryAgain bool
				var err error

				apiEndpoint := fmt.Sprintf("http://localhost:%d/api/v1", adminConsolePort)
				for i := 0; i < 5; i++ {
					tryAgain, err = uploadAirgapArchive(deployOptions, clientset, apiEndpoint, filepath.Join(deployOptions.AirgapRootDir, "app.tar.gz"))
					if err == nil {
						break
					}

					if tryAgain {
						time.Sleep(10 * time.Second)
						log.ActionWithoutSpinner("Retrying upload...")
						continue
					}

					if err != nil {
						return errors.Wrap(err, "failed to upload app.tar.gz")
					}
				}

				if tryAgain {
					return errors.Wrap(err, "giving up uploading app.tar.gz")
				}

				// remove here in case CLI is killed and defer doesn't run
				_ = os.RemoveAll(deployOptions.AirgapRootDir)
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

			m.ReportInstallFinish()

			isPortForwarding := !v.GetBool("no-port-forward")
			if isPortForwarding {
				// if --no-port-forward not specififed, check deprecated method
				isPortForwarding = v.GetBool("port-forward")
			}

			if isPortForwarding && !deployOptions.ExcludeAdminConsole {
				log.ActionWithoutSpinner("")

				if adminConsolePort != 8800 {
					log.ActionWithoutSpinner("Port 8800 is not available. The Admin Console is running on port %d", adminConsolePort)
					log.ActionWithoutSpinner("")
				}

				log.ActionWithoutSpinner("Press Ctrl+C to exit")
				log.ActionWithoutSpinner("Go to http://localhost:%d to access the Admin Console", adminConsolePort)
				log.ActionWithoutSpinner("")

				finalMessage = fmt.Sprintf("To access the Admin Console again, run kubectl kots admin-console --namespace %s", namespace)

				// pause indefinitely and let Ctrl+C handle termination
				<-make(chan struct{})
			} else if !deployOptions.ExcludeAdminConsole {
				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner("To access the Admin Console, run kubectl kots admin-console --namespace %s", namespace)
				log.ActionWithoutSpinner("")
			} else {
				log.ActionWithoutSpinner("")
				log.ActionWithoutSpinner("Done")
				log.ActionWithoutSpinner("")
			}

			return nil
		},
	}

	cmd.Flags().String("shared-password", "", "shared password to apply")
	cmd.Flags().String("name", "", "name of the application to use in the Admin Console")
	cmd.Flags().String("local-path", "", "specify a local-path to test the behavior of rendering a replicated app locally (only supported on replicated app types currently)")
	cmd.Flags().String("license-file", "", "path to a license file to use when download a replicated app")
	cmd.Flags().String("config-values", "", "path to a manifest containing config values (must be apiVersion: kots.io/v1beta1, kind: ConfigValues)")
	cmd.Flags().Bool("port-forward", true, "set to false to disable automatic port forward")
	cmd.Flags().MarkDeprecated("port-forward", "please use --no-port-forward instead")
	cmd.Flags().Bool("no-port-forward", false, "set to true to disable automatic port forward")
	cmd.Flags().String("wait-duration", "2m", "timeout out to be used while waiting for individual components to be ready.  must be in Go duration format (eg: 10s, 2m)")
	cmd.Flags().String("http-proxy", "", "sets HTTP_PROXY environment variable in all KOTS Admin Console components")
	cmd.Flags().String("https-proxy", "", "sets HTTPS_PROXY environment variable in all KOTS Admin Console components")
	cmd.Flags().String("no-proxy", "", "sets NO_PROXY environment variable in all KOTS Admin Console components")
	cmd.Flags().Bool("copy-proxy-env", false, "copy proxy environment variables from current environment into all KOTS Admin Console components")
	cmd.Flags().String("airgap-bundle", "", "path to the application airgap bundle where application metadata will be loaded from")
	cmd.Flags().Bool("airgap", false, "set to true to run install in airgapped mode. setting --airgap-bundle implies --airgap=true.")
	cmd.Flags().Bool("skip-preflights", false, "set to true to skip preflight checks")
	cmd.Flags().Bool("disable-image-push", false, "set to true to disable images from being pushed to private registry")
	cmd.Flags().Bool("strict-security-context", false, "set to explicitly enable explicit security contexts for all kots pods and containers (may not work for some storage providers)")
	cmd.Flags().Bool("skip-compatibility-check", false, "set to true to skip compatibility checks between the current kots version and the app")

	cmd.Flags().String("repo", "", "repo uri to use when installing a helm chart")
	cmd.Flags().StringSlice("set", []string{}, "values to pass to helm when running helm template")

	registryFlags(cmd.Flags())

	// the following group of flags are experiemental and can be used to pull and push images during install time
	cmd.Flags().Bool("rewrite-images", false, "set to true to force all container images to be rewritten and pushed to a local registry")
	cmd.Flags().String("image-namespace", "", "the namespace/org in the docker registry to push images to (required when --rewrite-images is set)")
	// set this to http://127.0.0.1:30000/api/v1 in dev environment
	cmd.Flags().String("registry-endpoint", "", "the endpoint of the local docker registry to use when pushing images (required when --rewrite-images is set)")
	cmd.Flags().MarkHidden("rewrite-images")
	cmd.Flags().MarkHidden("image-namespace")
	cmd.Flags().MarkHidden("registry-endpoint")

	// options for the alpha feature of using a reg instead of s3 for storage
	cmd.Flags().String("storage-base-uri", "", "an s3 or oci-registry uri to use for kots persistent storage in the cluster")
	cmd.Flags().Bool("with-minio", true, "when set, kots will deploy a local minio instance for storage")
	cmd.Flags().Bool("with-dockerdistribution", false, "when set, kots install will deploy a local instance of docker distribution for storage")
	cmd.Flags().Bool("storage-base-uri-plainhttp", false, "when set, use plain http (not https) connecting to the local oci storage")
	cmd.Flags().MarkHidden("storage-base-uri")
	cmd.Flags().MarkHidden("with-dockerdistribution")
	cmd.Flags().MarkHidden("storage-base-uri-plainhttp")

	cmd.Flags().Bool("ensure-rbac", true, "when set, kots will create the roles and rolebindings necessary to manage applications")

	cmd.Flags().String("airgap-upload-parallelism", "", "the number of chunks to upload in parallel when installing or updating in airgap mode")
	cmd.Flags().MarkHidden("airgap-upload-parallelism")

	cmd.Flags().Bool("enable-identity-service", false, "when set, the KOTS identity service will be enabled")
	cmd.Flags().MarkHidden("enable-identity-service")
	cmd.Flags().String("identity-config", "", "path to a manifest containing the KOTS identity service configuration (must be apiVersion: kots.io/v1beta1, kind: IdentityConfig)")
	cmd.Flags().MarkHidden("identity-config")

	cmd.Flags().Bool("enable-ingress", false, "when set, ingress will be enabled for the KOTS Admin Console")
	cmd.Flags().MarkHidden("enable-ingress")
	cmd.Flags().String("ingress-config", "", "path to a kots.Ingress resource file")
	cmd.Flags().MarkHidden("ingress-config")

	// option to check if the user has cluster-wide previliges to install application
	cmd.Flags().Bool("skip-rbac-check", false, "set to true to bypass rbac check")
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
		Validate:  validateNamespace,
		AllowEdit: true,
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

func uploadAirgapArchive(deployOptions kotsadmtypes.DeployOptions, clientset *kubernetes.Clientset, apiEndpoint string, filename string) (bool, error) {
	body := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(body)

	metadataPart, err := bodyWriter.CreateFormField("appSlug")
	if err != nil {
		return false, errors.Wrap(err, "failed to add metadata")
	}
	if _, err := io.Copy(metadataPart, bytes.NewReader([]byte(deployOptions.License.Spec.AppSlug))); err != nil {
		return false, errors.Wrap(err, "failed to copy metadata")
	}

	fileWriter, err := bodyWriter.CreateFormFile("appArchive", filepath.Base(filename))
	if err != nil {
		return false, errors.Wrap(err, "failed to create form from file")
	}

	fileReader, err := os.Open(filename)
	if err != nil {
		return false, errors.Wrap(err, "failed to open app archive")
	}
	defer fileReader.Close()

	_, err = io.Copy(fileWriter, fileReader)
	if err != nil {
		return false, errors.Wrap(err, "failed to copy app archive")
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	authSlug, err := auth.GetOrCreateAuthSlug(clientset, deployOptions.Namespace)
	if err != nil {
		return false, errors.Wrap(err, "failed to get kotsadm auth slug")
	}

	url := fmt.Sprintf("%s/airgap/install", apiEndpoint)
	newRequest, err := http.NewRequest("POST", url, body)
	if err != nil {
		return false, errors.Wrap(err, "failed to create upload request")
	}
	newRequest.Header.Add("Authorization", authSlug)
	newRequest.Header.Add("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(newRequest)
	if err != nil {
		return false, errors.Wrap(err, "failed to get from kotsadm")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return true, errors.Errorf("unexpected response status: %v", resp.StatusCode)
	}

	return false, nil
}

func getIngressConfig(v *viper.Viper) (*kotsv1beta1.IngressConfig, error) {
	ingressConfigPath := v.GetString("ingress-config")
	enableIngress := v.GetBool("enable-ingress") || ingressConfigPath != ""

	if !enableIngress {
		return &kotsv1beta1.IngressConfig{}, nil
	}

	ingressConfig := kotsv1beta1.IngressConfig{}
	if ingressConfigPath != "" {
		content, err := ioutil.ReadFile(ingressConfigPath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read ingress service config file")
		}

		s, err := kotsutil.LoadIngressConfigFromContents(content)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decoce ingress service config")
		}
		ingressConfig = *s
	}

	ingressConfig.Spec.Enabled = true

	return &ingressConfig, nil
}

func getIdentityConfig(v *viper.Viper) (*kotsv1beta1.IdentityConfig, error) {
	identityConfigPath := v.GetString("identity-config")
	enableIdentityService := v.GetBool("enable-identity-service") || identityConfigPath != ""

	if !enableIdentityService {
		return &kotsv1beta1.IdentityConfig{}, nil
	}

	identityConfig := kotsv1beta1.IdentityConfig{}
	if identityConfigPath != "" {
		content, err := ioutil.ReadFile(identityConfigPath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read identity service config file")
		}

		s, err := kotsutil.LoadIdentityConfigFromContents(content)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decoce identity service config")
		}
		identityConfig = *s
	}

	identityConfig.Spec.Enabled = true

	return &identityConfig, nil
}

func registryFlags(flagset *pflag.FlagSet) {
	flagset.String("kotsadm-registry", "", "set to override the registry of kotsadm images. used for airgapped installations.")
	flagset.String("registry-username", "", "username to use to authenticate with the application registry. used for airgapped installations.")
	flagset.String("registry-password", "", "password to use to authenticate with the application registry. used for airgapped installations.")

	// the following group of flags are useful for testing, but we don't want to pollute the help screen with them
	flagset.String("kotsadm-tag", "", "set to override the tag of kotsadm. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	flagset.String("kotsadm-namespace", "", "set to override the namespace of kotsadm images. used for airgapped installations.")
	flagset.MarkHidden("kotsadm-tag")
}

func getRegistryConfig(v *viper.Viper) (*kotsadmtypes.KotsadmOptions, error) {
	registryEndpoint := v.GetString("kotsadm-registry")
	registryNamespace := v.GetString("kotsadm-namespace")
	registryUsername := v.GetString("registry-username")
	registryPassword := v.GetString("registry-password")

	if registryNamespace == "" {
		parts := strings.Split(registryEndpoint, "/")
		if len(parts) > 1 {
			registryEndpoint = parts[0]
			registryNamespace = strings.Join(parts[1:], "/")
		}
	}

	isKurl, err := kotsadm.IsKurl()
	if err != nil {
		return nil, errors.Wrap(err, "failed to check kURL")
	}

	isAirgap := false
	if v.GetString("airgap-bundle") != "" || v.GetBool("airgap") {
		isAirgap = true
	}

	if registryEndpoint == "" && isKurl && isAirgap {
		license, err := getLicense(v)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get license")
		}
		registryEndpoint, registryUsername, registryPassword, err = kotsutil.GetKurlRegistryCreds()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get kURL registry info")
		}
		if registryNamespace == "" && license != nil {
			registryNamespace = license.Spec.AppSlug
		}
	}
	return &kotsadmtypes.KotsadmOptions{
		OverrideVersion:   v.GetString("kotsadm-tag"),
		OverrideRegistry:  registryEndpoint,
		OverrideNamespace: registryNamespace,
		Username:          registryUsername,
		Password:          registryPassword,
	}, nil
}

func getLicense(v *viper.Viper) (*kotsv1beta1.License, error) {
	if v.GetString("license-file") == "" {
		return nil, nil
	}

	license, err := pull.ParseLicenseFromFile(ExpandDir(v.GetString("license-file")))
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse license file")
	}

	return license, nil
}

func getHttpProxyEnv(v *viper.Viper) map[string]string {
	env := make(map[string]string)

	if v.GetBool("copy-proxy-env") {
		env["HTTP_PROXY"] = os.Getenv("HTTP_PROXY")
		env["http_proxy"] = os.Getenv("http_proxy")
		env["HTTPS_PROXY"] = os.Getenv("HTTPS_PROXY")
		env["https_proxy"] = os.Getenv("https_proxy")
		env["NO_PROXY"] = os.Getenv("NO_PROXY")
		env["no_proxy"] = os.Getenv("no_proxy")
		return env
	}

	env["HTTP_PROXY"] = v.GetString("http-proxy")
	env["HTTPS_PROXY"] = v.GetString("https-proxy")
	env["NO_PROXY"] = v.GetString("no-proxy")
	return env

}

var (
	RBACError = errors.New("attempting to grant RBAC permissions not currently held")
)

func CheckRBAC() error {
	clientConfig, err := k8sutil.GetClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create clientset")
	}

	sar := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Namespace:   "",
				Verb:        "*",
				Group:       "*",
				Version:     "*",
				Resource:    "*",
				Subresource: "",
				Name:        "",
			},
			NonResourceAttributes: nil,
		},
	}

	resp, err := clientset.AuthorizationV1().SelfSubjectAccessReviews().Create(context.Background(), sar, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to run subject review")
	}

	if !resp.Status.Allowed {
		return RBACError
	}
	return nil
}
