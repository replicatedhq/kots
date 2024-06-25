package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	"github.com/replicatedhq/kots/pkg/archives"
	"github.com/replicatedhq/kots/pkg/auth"
	"github.com/replicatedhq/kots/pkg/automation"
	dockerregistry "github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/handlers"
	"github.com/replicatedhq/kots/pkg/identity"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	k8sutiltypes "github.com/replicatedhq/kots/pkg/k8sutil/types"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/metrics"
	preflighttypes "github.com/replicatedhq/kots/pkg/preflight/types"
	"github.com/replicatedhq/kots/pkg/print"
	"github.com/replicatedhq/kots/pkg/pull"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/tasks"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/troubleshoot/pkg/preflight"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	authorizationv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var client = &http.Client{
	Timeout: time.Second * 30,
}

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

			log := logger.NewCLILogger(cmd.OutOrStdout())

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

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get k8s clientset")
			}

			appSlug := ""
			if license != nil {
				appSlug = license.Spec.AppSlug
			}

			registryConfig, err := getRegistryConfig(v, clientset, appSlug)
			if err != nil {
				return errors.Wrap(err, "failed to get registry config")
			}

			if registryConfig.OverrideRegistry != "" && !v.GetBool("skip-registry-check") {
				log.ActionWithSpinner("Validating registry information")

				host, err := getHostFromEndpoint(registryConfig.OverrideRegistry)
				if err != nil {
					log.FinishSpinnerWithError()
					return errors.Wrap(err, "failed get host from endpoint")
				}

				if err := dockerregistry.CheckAccess(host, registryConfig.Username, registryConfig.Password); err != nil {
					log.FinishSpinnerWithError()
					return fmt.Errorf("Failed to test access to %q with user %q: %v", host, registryConfig.Username, err)
				}

				log.FinishSpinner()
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

			applicationMetadata := &replicatedapp.ApplicationMetadata{}
			if airgapBundle := v.GetString("airgap-bundle"); airgapBundle != "" {
				applicationMetadata, err = pull.GetAppMetadataFromAirgap(airgapBundle)
				if err != nil {
					return errors.Wrapf(err, "failed to get metadata from %s", airgapBundle)
				}
			} else if !v.GetBool("airgap") {
				applicationMetadata, err = pull.PullApplicationMetadata(upstream, v.GetString("app-version-label"))
				if err != nil {
					log.Info("Unable to pull application metadata. This can be ignored, but custom branding will not be available in the Admin Console until a license is installed. This may also cause the Admin Console to run without minimal role-based-access-control (RBAC) privileges, which may be required by the application.")
					applicationMetadata = &replicatedapp.ApplicationMetadata{}
				}
			}

			// checks kots version compatibility with the app
			if len(applicationMetadata.Manifest) > 0 && !v.GetBool("skip-compatibility-check") {
				kotsApp, err := kotsutil.LoadKotsAppFromContents(applicationMetadata.Manifest)
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

			isKurl, err := kurl.IsKurl(clientset)
			if err != nil {
				return errors.Wrap(err, "failed to check if cluster is kurl")
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
				Namespace:              namespace,
				Context:                v.GetString("context"),
				SharedPassword:         sharedPassword,
				ApplicationMetadata:    applicationMetadata.Manifest,
				UpstreamURI:            upstream,
				License:                license,
				ConfigValues:           configValues,
				Airgap:                 isAirgap,
				ProgressWriter:         os.Stdout,
				Timeout:                time.Minute * 2,
				HTTPProxyEnvValue:      v.GetString("http-proxy"),
				HTTPSProxyEnvValue:     v.GetString("https-proxy"),
				NoProxyEnvValue:        v.GetString("no-proxy"),
				SkipPreflights:         v.GetBool("skip-preflights"),
				SkipCompatibilityCheck: v.GetBool("skip-compatibility-check"),
				AppVersionLabel:        v.GetString("app-version-label"),
				EnsureRBAC:             v.GetBool("ensure-rbac"),
				SkipRBACCheck:          v.GetBool("skip-rbac-check"),
				UseMinimalRBAC:         v.GetBool("use-minimal-rbac"),
				InstallID:              m.InstallID,
				SimultaneousUploads:    simultaneousUploads,
				DisableImagePush:       v.GetBool("disable-image-push"),
				AirgapBundle:           v.GetString("airgap-bundle"),
				IncludeMinio:           v.GetBool("with-minio"),
				IncludeMinioSnapshots:  v.GetBool("with-minio"),
				StrictSecurityContext:  v.GetBool("strict-security-context"),

				RegistryConfig: *registryConfig,

				IdentityConfig: *identityConfig,
				IngressConfig:  *ingressConfig,
			}

			deployOptions.IsOpenShift = k8sutil.IsOpenShift(clientset)

			deployOptions.IsGKEAutopilot = k8sutil.IsGKEAutopilot(clientset)

			timeout, err := time.ParseDuration(v.GetString("wait-duration"))
			if err != nil {
				return errors.Wrap(err, "failed to parse timeout value")
			}
			deployOptions.Timeout = timeout

			preflightsTimeout, err := time.ParseDuration(v.GetString("preflights-wait-duration"))
			if err != nil {
				return errors.Wrap(err, "failed to parse timeout value")
			}
			deployOptions.PreflightsTimeout = preflightsTimeout

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

			if airgapArchive := v.GetString("airgap-bundle"); airgapArchive != "" {
				if deployOptions.License == nil {
					return errors.New("license is required when airgap bundle is specified")
				}

				deployOptions.AirgapBundle = airgapArchive
			}

			if v.GetBool("exclude-admin-console") || (isKurl && deployOptions.Namespace == metav1.NamespaceDefault) {
				deployOptions.ExcludeAdminConsole = true
				deployOptions.EnsureKotsadmConfig = true
				log.ActionWithoutSpinner("Deploying application")
			} else {
				log.ActionWithoutSpinner("Deploying Admin Console")
			}

			if err := kotsadm.Deploy(deployOptions, log); err != nil {
				if _, ok := errors.Cause(err).(*k8sutiltypes.ErrorTimeout); ok {
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
					if _, ok := errors.Cause(err).(*k8sutiltypes.ErrorTimeout); ok {
						return podName, errors.Errorf("kotsadm failed to start: %s. Use the --wait-duration flag to increase timeout.", err)
					}
					return podName, errors.Wrap(err, "failed to wait for web")
				}
				return podName, nil
			}

			stopCh := make(chan struct{})
			defer close(stopCh)

			localPort := viper.GetInt("port")
			pollForAdditionalPorts := true
			if localPort != 8800 {
				pollForAdditionalPorts = false
			}

			adminConsolePort, errChan, err := k8sutil.PortForward(localPort, 3000, namespace, getPodName, pollForAdditionalPorts, stopCh, log)
			if err != nil {
				return errors.Wrap(err, "failed to forward port")
			}

			apiEndpoint := fmt.Sprintf("http://localhost:%d/api/v1", adminConsolePort)

			authSlug, err := auth.GetOrCreateAuthSlug(clientset, deployOptions.Namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get kotsadm auth slug")
			}

			// upload branding
			if len(applicationMetadata.Branding) > 0 {
				_, err = uploadBrandingArchive(authSlug, apiEndpoint, applicationMetadata.Branding)
				if err != nil {
					return errors.Wrap(err, "failed to upload branding")
				}
			}

			if deployOptions.AirgapBundle != "" {
				log.ActionWithoutSpinner("Uploading app archive")

				var tryAgain bool
				var err error

				for i := 0; i < 5; i++ {
					tryAgain, err = uploadAirgapArchive(deployOptions, authSlug, apiEndpoint, "app.tar.gz")
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
			}

			go func() {
				select {
				case err := <-errChan:
					if err != nil {
						log.Error(err)
						// TODO: Why is this a negative exit code?
						os.Exit(-1)
					}
				case <-stopCh:
				}
			}()

			if deployOptions.License != nil {
				log.ActionWithSpinner("Waiting for installation to complete")
				status, err := ValidateAutomatedInstall(deployOptions, authSlug, apiEndpoint)
				if err != nil {
					log.FinishSpinnerWithError()
					return errors.Wrap(err, "failed to validate installation")
				}
				log.FinishSpinner()

				switch status {
				case storetypes.VersionPendingPreflight, storetypes.VersionPending:
					log.ActionWithSpinner("Waiting for preflight checks to complete")
					if err := ValidatePreflightStatus(deployOptions, authSlug, apiEndpoint); err != nil {
						perr := preflightError{}
						if errors.As(err, &perr) {
							log.FinishSpinner() // We succeeded waiting for the results. Don't finish with an error
							log.Errorf(perr.Msg)
							print.PreflightResults(perr.Results)
							cmd.SilenceErrors = true // Stop Cobra from printing the error, we format the message ourselves
						} else {
							log.FinishSpinnerWithError()
						}
						return err
					}
					log.FinishSpinner()
				case storetypes.VersionPendingConfig:
					log.ActionWithoutSpinnerWarning("Additional app configuration is required. Please login to the Admin Console to continue", nil)
				}
			}

			m.ReportInstallFinish()

			isPortForwarding := !v.GetBool("no-port-forward")
			if isPortForwarding {
				// if --no-port-forward not specififed, check deprecated method
				isPortForwarding = v.GetBool("port-forward")
			}

			if isPortForwarding && !deployOptions.ExcludeAdminConsole {
				log.ActionWithoutSpinner("")

				if adminConsolePort != localPort {
					log.ActionWithoutSpinner("Port %d is not available. The Admin Console is running on port %d", localPort, adminConsolePort)
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
	cmd.Flags().Int("port", 8800, "local port to listen on when port forwarding is enabled")
	cmd.Flags().String("wait-duration", "2m", "timeout to be used while waiting for individual components to be ready. must be in Go duration format (eg: 10s, 2m)")
	cmd.Flags().String("preflights-wait-duration", "15m", "timeout to be used while waiting for preflights to complete. must be in Go duration format (eg: 10s, 2m)")
	cmd.Flags().String("http-proxy", "", "sets HTTP_PROXY environment variable in all KOTS Admin Console components")
	cmd.Flags().String("https-proxy", "", "sets HTTPS_PROXY environment variable in all KOTS Admin Console components")
	cmd.Flags().String("no-proxy", "", "sets NO_PROXY environment variable in all KOTS Admin Console components")
	cmd.Flags().Bool("copy-proxy-env", false, "copy proxy environment variables from current environment into all KOTS Admin Console components")
	cmd.Flags().String("airgap-bundle", "", "path to the application airgap bundle where application metadata will be loaded from")
	cmd.Flags().Bool("airgap", false, "set to true to run install in airgapped mode. setting --airgap-bundle implies --airgap=true.")
	cmd.Flags().Bool("skip-preflights", false, "set to true to skip preflight checks")
	cmd.Flags().Bool("disable-image-push", false, "set to true to disable images from being pushed to private registry")
	cmd.Flags().Bool("skip-registry-check", false, "set to true to skip the connectivity test and validation of the provided registry information")
	cmd.Flags().Bool("strict-security-context", false, "set to explicitly enable explicit security contexts for all kots pods and containers (may not work for some storage providers)")
	cmd.Flags().Bool("skip-compatibility-check", false, "set to true to skip compatibility checks between the current kots version and the app")
	cmd.Flags().String("app-version-label", "", "the application version label to install. if not specified, the latest version will be installed")
	cmd.Flags().Bool("exclude-admin-console", false, "set to true to exclude the admin console and only install the application")

	cmd.Flags().String("repo", "", "repo uri to use when installing a helm chart")

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
	cmd.Flags().MarkHidden("storage-base-uri")

	cmd.Flags().Bool("ensure-rbac", true, "when set, kots will create the roles and rolebindings necessary to manage applications")
	cmd.Flags().Bool("use-minimal-rbac", false, "when set, kots will be namespace scoped if application supports namespace scoped installations")

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

func uploadBrandingArchive(authSlug string, apiEndpoint string, data []byte) (bool, error) {
	body := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(body)

	fileWriter, err := bodyWriter.CreateFormFile("brandingArchive", "branding.tar.gz")
	if err != nil {
		return false, errors.Wrap(err, "failed to create form from file")
	}

	reader := bytes.NewReader(data)

	_, err = io.Copy(fileWriter, reader)
	if err != nil {
		return false, errors.Wrap(err, "failed to copy branding archive")
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	url := fmt.Sprintf("%s/branding/install", apiEndpoint)
	newRequest, err := http.NewRequest("POST", url, body)
	if err != nil {
		return false, errors.Wrap(err, "failed to create upload request")
	}
	newRequest.Header.Add("Authorization", authSlug)
	newRequest.Header.Add("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(newRequest)
	if err != nil {
		return false, errors.Wrap(err, "failed to upload branding archive")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return true, errors.Errorf("unexpected response status: %v", resp.StatusCode)
	}

	return false, nil
}

func uploadAirgapArchive(deployOptions kotsadmtypes.DeployOptions, authSlug string, apiEndpoint string, filename string) (bool, error) {
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

	contents, err := archives.GetFileContentFromTGZArchive(filename, deployOptions.AirgapBundle)
	if err != nil {
		return false, errors.Wrap(err, "failed to get file from airgap")
	}

	if _, err := fileWriter.Write(contents); err != nil {
		return false, errors.Wrap(err, "failed to copy airgap archive")
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

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
		content, err := os.ReadFile(ingressConfigPath)
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
		content, err := os.ReadFile(identityConfigPath)
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

func getRegistryConfig(v *viper.Viper, clientset kubernetes.Interface, appSlug string) (*kotsadmtypes.RegistryConfig, error) {
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

	isAirgap := false
	if v.GetString("airgap-bundle") != "" || v.GetBool("airgap") {
		isAirgap = true
	}

	if registryEndpoint == "" && isAirgap {
		clientset, err := k8sutil.GetClientset()
		if err != nil {
			return nil, errors.Wrap(err, "failed to get clientset")
		}
		registryEndpoint, registryUsername, registryPassword = kotsutil.GetEmbeddedRegistryCreds(clientset)
		if registryNamespace == "" {
			registryNamespace = appSlug
		}
	}

	return &kotsadmtypes.RegistryConfig{
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

	license, err := kotsutil.LoadLicenseFromPath(ExpandDir(v.GetString("license-file")))
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

func ValidateAutomatedInstall(deployOptions kotsadmtypes.DeployOptions, authSlug string, apiEndpoint string) (storetypes.DownstreamVersionStatus, error) {
	url := fmt.Sprintf("%s/app/%s/automated/status", apiEndpoint, deployOptions.License.Spec.AppSlug)

	startTime := time.Now()

	for time.Since(startTime) < deployOptions.Timeout {
		taskStatus, err := getAutomatedInstallStatus(url, authSlug)
		if err != nil {
			return "", errors.Wrap(err, "failed to get automated install status")
		}
		taskMessage := automation.AutomateInstallTaskMessage{}
		err = json.Unmarshal([]byte(taskStatus.Message), &taskMessage)
		if err != nil {
			return "", errors.Wrap(err, "failed to unmarshal automated install task message")
		}

		switch taskStatus.Status {
		case automation.AutomatedInstallFailed:
			return "", errors.New(taskMessage.Error)
		case automation.AutomatedInstallSuccess:
			return taskMessage.VersionStatus, nil
		}
		time.Sleep(time.Second)
	}

	return "", errors.New("timeout waiting for automated install. Use the --wait-duration flag to increase timeout.")
}

func getAutomatedInstallStatus(url string, authSlug string) (*tasks.TaskStatus, error) {
	newReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	newReq.Header.Add("Content-Type", "application/json")
	newReq.Header.Add("Authorization", authSlug)

	resp, err := http.DefaultClient.Do(newReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	taskStatus := tasks.TaskStatus{}
	if err := json.Unmarshal(b, &taskStatus); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal task status")
	}

	return &taskStatus, nil
}

func ValidatePreflightStatus(deployOptions kotsadmtypes.DeployOptions, authSlug string, apiEndpoint string) error {
	url := fmt.Sprintf("%s/app/%s/preflight/result", apiEndpoint, deployOptions.License.Spec.AppSlug)

	startTime := time.Now()

	for time.Since(startTime) < deployOptions.PreflightsTimeout {
		response, err := getPreflightResponse(url, authSlug)
		if err != nil {
			return errors.Wrap(err, "failed to get preflight status")
		}

		preflightsComplete, err := checkPreflightsComplete(response)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal collect progress for preflights")
		}
		if !preflightsComplete {
			continue
		}

		resultsAvailable, err := checkPreflightResults(response, deployOptions.SkipPreflights)
		if err != nil {
			return err
		}
		if !resultsAvailable {
			continue
		}

		return nil
	}

	return errors.New("timeout waiting for preflights to finish. Use the --preflights-wait-duration flag to increase timeout.")
}

func getPreflightResponse(url string, authSlug string) (*handlers.GetPreflightResultResponse, error) {
	newReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	newReq.Header.Add("Content-Type", "application/json")
	newReq.Header.Add("Authorization", authSlug)

	resp, err := client.Do(newReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	var response handlers.GetPreflightResultResponse
	if err = json.Unmarshal(b, &response); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal the preflight response")
	}

	return &response, nil
}

func checkPreflightsComplete(response *handlers.GetPreflightResultResponse) (bool, error) {
	if response.PreflightProgress == "" {
		return true, nil
	}

	var collectProgress *preflight.CollectProgress
	err := json.Unmarshal([]byte(response.PreflightProgress), &collectProgress)
	if err != nil {
		return false, err
	}

	if collectProgress != nil && collectProgress.TotalCount < 1 {
		return false, nil
	}

	return true, nil
}

type preflightError struct {
	Msg     string
	Results preflighttypes.PreflightResults
}

func (e preflightError) Error() string {
	return e.Msg
}

func (e preflightError) Unwrap() error { return fmt.Errorf(e.Msg) }

func checkPreflightResults(response *handlers.GetPreflightResultResponse, skipPreflights bool) (bool, error) {
	if response.PreflightResult.Result == "" {
		return false, nil
	}

	var results preflighttypes.PreflightResults
	err := json.Unmarshal([]byte(response.PreflightResult.Result), &results)
	if err != nil {
		return false, errors.Wrap(err, fmt.Sprintf("failed to unmarshal upload preflight results from response: %v", response.PreflightResult.Result))
	}

	if len(results.Errors) > 0 {
		isRBAC := false
		for _, err := range results.Errors {
			if err.IsRBAC {
				isRBAC = true
				break
			}
		}
		msg := "There are preflight check errors for the application. The app was not deployed."
		if isRBAC {
			msg = "The Kubernetes RBAC policy that the Admin Console is running with does not have access to complete the Preflight Checks. It's recommended that you run these manually before proceeding. The app was not deployed."
		}
		return false, preflightError{
			Msg:     msg,
			Results: results,
		}
	}

	var isWarn, isFail bool
	for _, result := range results.Results {
		if skipPreflights && !result.Strict {
			// if we're skipping preflights, we should only check the strict preflight results
			continue
		}
		if result.IsWarn {
			isWarn = true
		}
		if result.IsFail {
			isFail = true
		}
	}

	if isWarn && isFail {
		return false, preflightError{
			Msg:     "There are preflight check failures and warnings for the application. The app was not deployed.",
			Results: results,
		}
	}

	if isWarn {
		return false, preflightError{
			Msg:     "There are preflight check warnings for the application. The app was not deployed.",
			Results: results,
		}
	}
	if isFail {
		return false, preflightError{
			Msg:     "There are preflight check failures for the application. The app was not deployed.",
			Results: results,
		}
	}

	return true, nil
}
