package cli

import (
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/multichannel"
	"github.com/replicatedhq/kots/pkg/pull"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func PullCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "pull [upstream uri]",
		Short:         "Pull Kubernetes manifests from remote upstream to the local filesystem",
		Long:          `Pull Kubernetes manifests from the remote upstream and save them to the local filesystem, so they can be edited before deploying them to a cluster.`,
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

			// registry host should not have the scheme (https).  need to
			// strip it if included or else the rewrite images will fail

			if v.GetBool("load-apiversions-from-server") {
				if err := k8sutil.InitHelmCapabilities(); err != nil {
					return errors.Wrap(err, "failed to initialize helm")
				}
			}

			license, err := getLicense(v)
			if err != nil {
				return errors.Wrap(err, "failed to get license")
			}

			appSlug := getAppSlugForPull(args[0], license)

			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			pullOptions := pull.PullOptions{
				AppSlug:             appSlug,
				RootDir:             ExpandDir(v.GetString("rootdir")),
				Namespace:           namespace,
				Downstreams:         v.GetStringSlice("downstream"),
				LocalPath:           ExpandDir(v.GetString("local-path")),
				LicenseFile:         ExpandDir(v.GetString("license-file")),
				ConfigFile:          ExpandDir(v.GetString("config-values")),
				IdentityConfigFile:  ExpandDir(v.GetString("identity-config")),
				ExcludeKotsKinds:    v.GetBool("exclude-kots-kinds"),
				ExcludeAdminConsole: v.GetBool("exclude-admin-console"),
				SharedPassword:      v.GetString("shared-password"),
				CreateAppDir:        true,
				SkipHelmChartCheck:  true, // this check cannot be performed from CLI because previous release is not available
				RewriteImages:       v.GetBool("rewrite-images"),
				RewriteImageOptions: registrytypes.RegistrySettings{
					Hostname:  v.GetString("registry-endpoint"),
					Namespace: v.GetString("image-namespace"),
					Username:  v.GetString("registry-username"),
					Password:  v.GetString("registry-password"),
				},
				StorageClassName:       v.GetString("storage-class"),
				HTTPProxyEnvValue:      v.GetString("http-proxy"),
				HTTPSProxyEnvValue:     v.GetString("https-proxy"),
				NoProxyEnvValue:        v.GetString("no-proxy"),
				IncludeMinio:           v.GetBool("with-minio"),
				SkipCompatibilityCheck: v.GetBool("skip-compatibility-check"),
			}

			if v.GetBool("copy-proxy-env") {
				pullOptions.HTTPProxyEnvValue = os.Getenv("HTTP_PROXY")
				if pullOptions.HTTPProxyEnvValue == "" {
					pullOptions.HTTPProxyEnvValue = os.Getenv("http_proxy")
				}
				pullOptions.HTTPSProxyEnvValue = os.Getenv("HTTPS_PROXY")
				if pullOptions.HTTPSProxyEnvValue == "" {
					pullOptions.HTTPSProxyEnvValue = os.Getenv("https_proxy")
				}
				pullOptions.NoProxyEnvValue = os.Getenv("NO_PROXY")
				if pullOptions.NoProxyEnvValue == "" {
					pullOptions.NoProxyEnvValue = os.Getenv("no_proxy")
				}
			}

			upstream := pull.RewriteUpstream(args[0])
			preferredChannelSlug, err := extractPreferredChannelSlug(upstream)
			if err != nil {
				return errors.Wrap(err, "failed to extract preferred channel slug")
			}

			log := logger.NewCLILogger(cmd.OutOrStdout())
			log.Initialize()

			// If we are passed a multi-channel license, verify that the requested channel is in the license
			// so that we can warn the user immediately if it is not.
			license, err = multichannel.VerifyAndUpdateLicense(log, license, preferredChannelSlug, false)
			if err != nil {
				return errors.Wrap(err, "failed to verify and update license")
			}

			renderDir, err := pull.Pull(upstream, pullOptions)
			if err != nil {
				return errors.Wrap(err, "failed to pull")
			}

			log.Info("Kubernetes application files created in %s", renderDir)
			if len(v.GetStringSlice("downstream")) == 0 {
				log.Info("To deploy, run kubectl apply -k %s", path.Join(renderDir, "overlays", "midstream"))
			} else if len(v.GetStringSlice("downstream")) == 1 {
				log.Info("To deploy, run kubectl apply -k %s", path.Join(renderDir, "overlays", "downstreams", v.GetStringSlice("downstream")[0]))
			} else {
				log.Info("To deploy, run kubectl apply -k from the downstream directory you would like to deploy")
			}

			return nil
		},
	}

	cmd.Flags().String("rootdir", ".", "root directory that will be used to write the yaml to")
	cmd.Flags().StringP("namespace", "n", "default", "namespace to render the upstream to in the base")
	cmd.Flags().StringSlice("downstream", []string{}, "the list of any downstreams to create/update")
	cmd.Flags().String("local-path", "", "specify a local-path to pull a locally available replicated app (only supported on replicated app types currently)")
	cmd.Flags().String("license-file", "", "path to a license file to use when download a replicated app")
	cmd.Flags().String("config-values", "", "path to a manifest containing config values (must be apiVersion: kots.io/v1beta1, kind: ConfigValues)")
	cmd.Flags().String("identity-config", "", "path to a manifest containing the identity service configuration for the application (must be apiVersion: kots.io/v1beta1, kind: IdentityConfig)")
	cmd.Flags().Bool("exclude-kots-kinds", true, "set to true to exclude rendering kots custom objects to the base directory")
	cmd.Flags().Bool("exclude-admin-console", false, "set to true to exclude the admin console (replicated apps only)")
	cmd.Flags().String("shared-password", "", "shared password to use when deploying the admin console")
	cmd.Flags().String("storage-class", "", "sets the storage class to use for the KOTS Admin Console components (if unset, the default storage class will be used)")
	cmd.Flags().String("http-proxy", "", "sets HTTP_PROXY environment variable in all KOTS Admin Console components")
	cmd.Flags().String("https-proxy", "", "sets HTTPS_PROXY environment variable in all KOTS Admin Console components")
	cmd.Flags().String("no-proxy", "", "sets NO_PROXY environment variable in all KOTS Admin Console components")
	cmd.Flags().Bool("copy-proxy-env", false, "copy proxy environment variables from current environment into all KOTS Admin Console components")
	cmd.Flags().Bool("rewrite-images", false, "set to true to force all container images to be rewritten and pushed to a local registry")
	cmd.Flags().String("image-namespace", "", "the namespace/org in the docker registry to push images to (required when --rewrite-images is set)")
	cmd.Flags().String("registry-endpoint", "", "the endpoint of the local docker registry to use when pushing images (required when --rewrite-images is set)")
	cmd.Flags().String("registry-username", "", "the username of the local docker registry to use when pushing images (with --rewrite-images)")
	cmd.Flags().String("registry-password", "", "the password of the local docker registry to use when pushing images (with --rewrite-images)")
	cmd.Flags().Bool("with-minio", true, "set to true to include a local minio instance to be used for storage")
	cmd.Flags().Bool("skip-compatibility-check", false, "set to true to skip compatibility checks between the current kots version and the app")
	cmd.Flags().Bool("load-apiversions-from-server", false, "load supported k8s api versions from cluster for Helm charts with useHelmInstall flag set to true")
	cmd.Flags().MarkHidden("load-apiversions-from-server")

	return cmd
}

func getAppSlugForPull(uri string, license *kotsv1beta1.License) string {
	appSlug := strings.Split(uri, "/")[0]
	if license == nil {
		return appSlug
	}
	return license.Spec.AppSlug
}
