package cli

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func AdminGenerateManifestsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "generate-manifests",
		Short:         "Generate the Admin Console manifests and store in the local filesystem",
		Long:          "Generate the Admin Console manifests and store in the local filesystem, so they can be edited before deploying them to a cluster.",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			namespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get namespace")
			}

			// set defaults for variables requiring cluster context
			isOpenShift, isGKEAutopilot := false, false
			migrateToMinioXl, currentMinioImage := false, ""
			registryConfig := kotsadmtypes.RegistryConfig{
				OverrideVersion:   v.GetString("kotsadm-tag"),
				OverrideRegistry:  v.GetString("kotsadm-registry"),
				OverrideNamespace: v.GetString("kotsadm-namespace"),
				Username:          v.GetString("registry-username"),
				Password:          v.GetString("registry-password"),
			}

			log := logger.NewCLILogger(cmd.OutOrStdout())

			if clientset, err := k8sutil.GetClientset(); err == nil {
				isOpenShift, isGKEAutopilot = k8sutil.IsOpenShift(clientset), k8sutil.IsGKEAutopilot(clientset)
				migrateToMinioXl, currentMinioImage, _ = kotsadm.IsMinioXlMigrationNeeded(clientset, namespace)
				if newRegistryConfig, err := getRegistryConfig(v, clientset, ""); err == nil {
					registryConfig = *newRegistryConfig
				} else {
					log.Error(errors.Wrap(err, "failed to get registry config"))
				}
			}

			renderDir := ExpandDir(v.GetString("rootdir"))
			options := upstreamtypes.WriteOptions{
				Namespace:            namespace,
				SharedPassword:       v.GetString("shared-password"),
				HTTPProxyEnvValue:    v.GetString("http-proxy"),
				HTTPSProxyEnvValue:   v.GetString("https-proxy"),
				NoProxyEnvValue:      v.GetString("no-proxy"),
				IncludeMinio:         v.GetBool("with-minio"),
				MigrateToMinioXl:     migrateToMinioXl,
				CurrentMinioImage:    currentMinioImage,
				IsMinimalRBAC:        v.GetBool("minimal-rbac"),
				AdditionalNamespaces: v.GetStringSlice("additional-namespaces"),
				IsOpenShift:          isOpenShift,
				IsGKEAutopilot:       isGKEAutopilot,
				RegistryConfig:       registryConfig,
			}
			adminConsoleFiles, err := upstream.GenerateAdminConsoleFiles(renderDir, options)
			if err != nil {
				return errors.Wrap(err, "failed to generate admin console files")
			}

			for _, file := range adminConsoleFiles {
				fileRenderPath := filepath.Join(renderDir, file.Path)
				d, _ := filepath.Split(fileRenderPath)
				if _, err := os.Stat(d); os.IsNotExist(err) {
					if err := os.MkdirAll(d, 0744); err != nil {
						return errors.Wrap(err, "failed to mkdir")
					}
				}

				if err := ioutil.WriteFile(fileRenderPath, file.Content, 0644); err != nil {
					return errors.Wrapf(err, "failed to write file %s", fileRenderPath)
				}
			}
			log.Info("Admin Console manifests created in %s", filepath.Join(renderDir, "admin-console"))

			return nil
		},
	}

	cmd.Flags().String("rootdir", ".", "root directory that will be used to write the yaml to")
	cmd.Flags().String("http-proxy", "", "sets HTTP_PROXY environment variable in all KOTS Admin Console components")
	cmd.Flags().String("https-proxy", "", "sets HTTPS_PROXY environment variable in all KOTS Admin Console components")
	cmd.Flags().String("no-proxy", "", "sets NO_PROXY environment variable in all KOTS Admin Console components")
	cmd.Flags().String("shared-password", "", "shared password to use when deploying the admin console")
	cmd.Flags().Bool("with-minio", true, "set to true to include a local minio instance to be used for storage")
	cmd.Flags().Bool("minimal-rbac", false, "set to true to use the namespaced role and bindings instead of cluster-level permissions")
	cmd.Flags().StringSlice("additional-namespaces", []string{}, "Comma separate list to specify additional namespace(s) managed by KOTS outside where it is to be deployed. Ignored without with '--minimal-rbac=true'")

	registryFlags(cmd.Flags())

	return cmd
}
