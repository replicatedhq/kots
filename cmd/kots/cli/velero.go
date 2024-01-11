package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/print"
	"github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/replicatedhq/kots/pkg/snapshot/providers"
	snapshottypes "github.com/replicatedhq/kots/pkg/snapshot/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	resticRepoBase = "/var/velero-local-volume-provider"
)

func VeleroCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "velero",
		Short: "KOTS Velero interface",
	}

	cmd.AddCommand(VeleroEnsurePermissionsCmd())
	cmd.AddCommand(VeleroConfigureInternalCmd())
	cmd.AddCommand(VeleroConfigureAmazonS3Cmd())
	cmd.AddCommand(VeleroConfigureOtherS3Cmd())
	cmd.AddCommand(VeleroConfigureGCPCmd())
	cmd.AddCommand(VeleroConfigureAzureCmd())
	cmd.AddCommand(VeleroConfigureNFSCmd())
	cmd.AddCommand(VeleroConfigureHostPathCmd())
	cmd.AddCommand(VeleroPrintFileSystemInstructionsCmd())
	cmd.AddCommand(VeleroMigrateMinioFileSystemCmd())

	return cmd
}

func VeleroEnsurePermissionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "ensure-permissions",
		Short:         "Ensures the necessary permissions that enables the Admin Console to access Velero.",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			veleroNamespace := v.GetString("velero-namespace")
			if err := validateVeleroNamespace(veleroNamespace); err != nil {
				return err
			}

			kotsadmNamespace, err := getNamespaceOrDefault(v.GetString("namespace"))
			if err != nil {
				return errors.Wrap(err, "failed to get kotsadm namespace")
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			if err := snapshot.EnsureVeleroPermissions(cmd.Context(), clientset, veleroNamespace, kotsadmNamespace); err != nil {
				return err
			}

			if err := snapshot.EnsureVeleroNamespaceConfigMap(cmd.Context(), clientset, veleroNamespace, kotsadmNamespace); err != nil {
				return err
			}

			return nil
		},
	}

	cmd.Flags().String("velero-namespace", "", "namespace in which velero is installed")

	return cmd
}

func VeleroConfigureInternalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "configure-internal",
		Short:         "Configure snapshots to use the default object store provided in embedded clusters as storage",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			isKurl, err := kurl.IsKurl(clientset)
			if err != nil {
				return errors.Wrap(err, "failed to check if cluster is kurl")
			}

			if !isKurl {
				return errors.New("configuring snapshots to use the internal store is only supported for embedded clusters")
			}

			namespace := metav1.NamespaceDefault

			veleroStatus, err := snapshot.DetectVelero(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to detect velero")
			}
			if veleroStatus == nil {
				return errors.New("velero not found")
			}
			if !veleroStatus.ContainsPlugin("velero-plugin-for-aws") {
				return errors.New("velero does not have the 'velero-plugin-for-aws' plugin installed")
			}

			registryConfig, err := kotsadm.GetRegistryConfigFromCluster(namespace, clientset)
			if err != nil {
				return errors.Wrap(err, "failed to get registry options from cluster")
			}

			configureStoreOptions := snapshot.ConfigureStoreOptions{
				Internal:          true,
				KotsadmNamespace:  namespace,
				RegistryConfig:    &registryConfig,
				SkipValidation:    v.GetBool("skip-validation"),
				ValidateUsingAPod: true,
				IsMinioDisabled:   !v.GetBool("with-minio"),
			}
			_, err = snapshot.ConfigureStore(cmd.Context(), configureStoreOptions)
			if err != nil {
				return errors.Wrap(err, "failed to configure store")
			}

			log := logger.NewCLILogger(cmd.OutOrStdout())
			log.Info("\nStore Configured Successfully")

			return nil
		},
	}

	cmd.Flags().Bool("skip-validation", false, "skip the validation of the internal store endpoint/bucket")
	cmd.Flags().Bool("with-minio", true, "when set, kots will deploy minio for INTERNAL snapshot locations")

	return cmd
}

func VeleroConfigureAmazonS3Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure-aws-s3",
		Short: "Configure snapshots to use AWS S3 storage",
	}

	cmd.AddCommand(VeleroConfigureAmazonS3AccessKeyCmd())
	cmd.AddCommand(VeleroConfigureAmazonS3InstanceRoleCmd())

	return cmd
}

func VeleroConfigureAmazonS3AccessKeyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "access-key",
		Short:         "Configure snapshots to use AWS S3 storage using Access Key auth",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()
			log := logger.NewCLILogger(cmd.OutOrStdout())

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			registryConfig, err := kotsadm.GetRegistryConfigFromCluster(namespace, clientset)
			if err != nil {
				return errors.Wrap(err, "failed to get registry options from cluster")
			}

			veleroStatus, err := snapshot.DetectVelero(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to detect velero")
			}
			if veleroStatus == nil {
				print.VeleroInstallationInstructionsForCLI(log, snapshottypes.VeleroAWSPlugin, &registryConfig, strings.Join(os.Args, " "))
				return nil
			}
			if !veleroStatus.ContainsPlugin("velero-plugin-for-aws") {
				return errors.New("velero does not have the 'velero-plugin-for-aws' plugin installed")
			}

			configureStoreOptions := snapshot.ConfigureStoreOptions{
				Provider: "aws",
				Bucket:   v.GetString("bucket"),
				Path:     v.GetString("path"),
				AWS: &snapshottypes.StoreAWS{
					Region:          v.GetString("region"),
					AccessKeyID:     v.GetString("access-key-id"),
					SecretAccessKey: v.GetString("secret-access-key"),
					UseInstanceRole: false,
				},
				KotsadmNamespace: namespace,
				RegistryConfig:   &registryConfig,
				SkipValidation:   v.GetBool("skip-validation"),
			}
			_, err = snapshot.ConfigureStore(cmd.Context(), configureStoreOptions)
			if err != nil {
				return errors.Wrap(err, "failed to configure store")
			}

			log.Info("\nStore Configured Successfully")

			return nil
		},
	}

	cmd.Flags().String("bucket", "", "name of the object storage bucket where backups should be stored (required)")
	cmd.Flags().String("path", "", "path to a subdirectory in the object store bucket")
	cmd.Flags().String("region", "", "the region where the bucket exists (required)")
	cmd.Flags().String("access-key-id", "", "the aws access key id to use for accessing the bucket (required)")
	cmd.Flags().String("secret-access-key", "", "the aws secret access key to use for accessing the bucket (required)")
	cmd.Flags().Bool("skip-validation", false, "skip the validation of the aws s3 endpoint/bucket")

	cmd.MarkFlagRequired("bucket")
	cmd.MarkFlagRequired("region")
	cmd.MarkFlagRequired("access-key-id")
	cmd.MarkFlagRequired("secret-access-key")

	return cmd
}

func VeleroConfigureAmazonS3InstanceRoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "instance-role",
		Short:         "Configure snapshots to use AWS S3 storage with Instance Role auth",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()
			log := logger.NewCLILogger(cmd.OutOrStdout())

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			registryConfig, err := kotsadm.GetRegistryConfigFromCluster(namespace, clientset)
			if err != nil {
				return errors.Wrap(err, "failed to get registry options from cluster")
			}

			veleroStatus, err := snapshot.DetectVelero(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to detect velero")
			}
			if veleroStatus == nil {
				print.VeleroInstallationInstructionsForCLI(log, snapshottypes.VeleroAWSPlugin, &registryConfig, strings.Join(os.Args, " "))
				return nil
			}
			if !veleroStatus.ContainsPlugin("velero-plugin-for-aws") {
				return errors.New("velero does not have the 'velero-plugin-for-aws' plugin installed")
			}

			configureStoreOptions := snapshot.ConfigureStoreOptions{
				Provider: "aws",
				Bucket:   v.GetString("bucket"),
				Path:     v.GetString("path"),
				AWS: &snapshottypes.StoreAWS{
					Region:          v.GetString("region"),
					UseInstanceRole: true,
				},
				KotsadmNamespace: namespace,
				RegistryConfig:   &registryConfig,
				SkipValidation:   v.GetBool("skip-validation"),
			}
			_, err = snapshot.ConfigureStore(cmd.Context(), configureStoreOptions)
			if err != nil {
				return errors.Wrap(err, "failed to configure store")
			}

			log.Info("\nStore Configured Successfully")

			return nil
		},
	}

	cmd.Flags().String("bucket", "", "name of the object storage bucket where backups should be stored (required)")
	cmd.Flags().String("path", "", "path to a subdirectory in the object store bucket")
	cmd.Flags().String("region", "", "the region where the bucket exists (required)")
	cmd.Flags().Bool("skip-validation", false, "skip the validation of the aws s3 endpoint/bucket")

	cmd.MarkFlagRequired("bucket")
	cmd.MarkFlagRequired("region")

	return cmd
}

func VeleroConfigureOtherS3Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "configure-other-s3",
		Short:         "Configure snapshots to use an external s3 compatible storage",
		Long:          `Note that this command assumes that the bucket has already been created.`,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()
			log := logger.NewCLILogger(cmd.OutOrStdout())

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			registryConfig, err := getRegistryConfig(v, clientset, "")
			if err != nil {
				return errors.Wrap(err, "failed to get registry config")
			}
			if registryConfig.OverrideRegistry == "" {
				// check if there's already an existing registry configuration.
				rc, err := kotsadm.GetRegistryConfigFromCluster(namespace, clientset)
				if err != nil {
					return errors.Wrap(err, "failed to get registry options from cluster")
				}
				registryConfig = &rc
			}

			veleroStatus, err := snapshot.DetectVelero(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to detect velero")
			}
			if veleroStatus == nil {
				print.VeleroInstallationInstructionsForCLI(log, snapshottypes.VeleroAWSPlugin, registryConfig, strings.Join(os.Args, " "))
				return nil
			}
			if !veleroStatus.ContainsPlugin("velero-plugin-for-aws") {
				return errors.New("velero does not have the 'velero-plugin-for-aws' plugin installed")
			}

			var caCertData []byte
			caCertFile := v.GetString("cacert")
			if caCertFile != "" {
				realPath, err := filepath.Abs(caCertFile)
				if err != nil {
					return err
				}
				caCertData, err = os.ReadFile(realPath)
				if err != nil {
					return err
				}
			}

			if !v.GetBool("skip-validation") {
				log.Info("\nRunning a pod to test the connection to your S3 API and if the bucket exists...")
			}

			configureStoreOptions := snapshot.ConfigureStoreOptions{
				Provider: "aws",
				Bucket:   v.GetString("bucket"),
				Path:     v.GetString("path"),
				Other: &snapshottypes.StoreOther{
					Region:          v.GetString("region"),
					AccessKeyID:     v.GetString("access-key-id"),
					SecretAccessKey: v.GetString("secret-access-key"),
					Endpoint:        v.GetString("endpoint"),
				},
				KotsadmNamespace:  namespace,
				RegistryConfig:    registryConfig,
				SkipValidation:    v.GetBool("skip-validation"),
				ValidateUsingAPod: true,
				CACertData:        caCertData,
			}
			_, err = snapshot.ConfigureStore(cmd.Context(), configureStoreOptions)
			if err != nil {
				return errors.Wrap(err, "failed to configure store")
			}

			log.Info("\nStore Configured Successfully")

			return nil
		},
	}

	cmd.Flags().String("bucket", "", "name of the object storage bucket where backups should be stored (required)")
	cmd.Flags().String("path", "", "path to a subdirectory in the object store bucket")
	cmd.Flags().String("region", "", "the region where the bucket exists (required)")
	cmd.Flags().String("access-key-id", "", "the access key id to use for accessing the bucket (required)")
	cmd.Flags().String("secret-access-key", "", "the secret access key to use for accessing the bucket (required)")
	cmd.Flags().String("endpoint", "", "the s3 endpoint. (e.g. http://some-other-s3-endpoint, required)")
	cmd.Flags().String("cacert", "", "file containing a certificate bundle to use when verifying TLS connections to the object store.")
	cmd.Flags().Bool("skip-validation", false, "skip the validation of the s3 endpoint/bucket")

	cmd.MarkFlagRequired("bucket")
	cmd.MarkFlagRequired("region")
	cmd.MarkFlagRequired("access-key-id")
	cmd.MarkFlagRequired("secret-access-key")
	cmd.MarkFlagRequired("endpoint")

	registryFlags(cmd.Flags())

	return cmd
}

func VeleroConfigureGCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure-gcp",
		Short: "Configure snapshots to use GCP Object Storage",
	}

	cmd.AddCommand(VeleroConfigureGCPServiceAccount())
	cmd.AddCommand(VeleroConfigureGCPWorkloadIdentity())

	return cmd
}

func VeleroConfigureGCPServiceAccount() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "service-account",
		Short:         "Configure snapshots to use Google Cloud Storage using Service Account auth",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()
			log := logger.NewCLILogger(cmd.OutOrStdout())

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			registryConfig, err := kotsadm.GetRegistryConfigFromCluster(namespace, clientset)
			if err != nil {
				return errors.Wrap(err, "failed to get registry options from cluster")
			}

			veleroStatus, err := snapshot.DetectVelero(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to detect velero")
			}
			if veleroStatus == nil {
				print.VeleroInstallationInstructionsForCLI(log, snapshottypes.VeleroGCPPlugin, &registryConfig, strings.Join(os.Args, " "))
				return nil
			}
			if !veleroStatus.ContainsPlugin("velero-plugin-for-gcp") {
				return errors.New("velero does not have the 'velero-plugin-for-gcp' plugin installed")
			}

			jsonFile := ""
			if jsonFilePath := v.GetString("json-file"); jsonFilePath != "" {
				content, err := os.ReadFile(jsonFilePath)
				if err != nil {
					return errors.Wrap(err, "failed to read json file")
				}
				jsonFile = string(content)
			}

			configureStoreOptions := snapshot.ConfigureStoreOptions{
				Provider: "gcp",
				Bucket:   v.GetString("bucket"),
				Path:     v.GetString("path"),
				Google: &snapshottypes.StoreGoogle{
					JSONFile: jsonFile,
				},
				KotsadmNamespace: namespace,
				RegistryConfig:   &registryConfig,
				SkipValidation:   v.GetBool("skip-validation"),
			}
			_, err = snapshot.ConfigureStore(cmd.Context(), configureStoreOptions)
			if err != nil {
				return errors.Wrap(err, "failed to configure store")
			}

			log.Info("\nStore Configured Successfully")

			return nil
		},
	}

	cmd.Flags().String("bucket", "", "name of the object storage bucket where backups should be stored (required)")
	cmd.Flags().String("path", "", "path to a subdirectory in the object store bucket")
	cmd.Flags().String("json-file", "", "path to JSON credntials file for veloro (required)")
	cmd.Flags().Bool("skip-validation", false, "skip the validation of the bucket")

	cmd.MarkFlagRequired("bucket")
	cmd.MarkFlagRequired("json-file")

	return cmd
}

func VeleroConfigureGCPWorkloadIdentity() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "workload-identity",
		Short:         "Configure snapshots to use Google Cloud Storage with Workload Identity Auth",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()
			log := logger.NewCLILogger(cmd.OutOrStdout())

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			registryConfig, err := kotsadm.GetRegistryConfigFromCluster(namespace, clientset)
			if err != nil {
				return errors.Wrap(err, "failed to get registry options from cluster")
			}

			veleroStatus, err := snapshot.DetectVelero(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to detect velero")
			}
			if veleroStatus == nil {
				print.VeleroInstallationInstructionsForCLI(log, snapshottypes.VeleroGCPPlugin, &registryConfig, strings.Join(os.Args, " "))
				return nil
			}
			if !veleroStatus.ContainsPlugin("velero-plugin-for-gcp") {
				return errors.New("velero does not have the 'velero-plugin-for-gcp' plugin installed")
			}

			configureStoreOptions := snapshot.ConfigureStoreOptions{
				Provider: "gcp",
				Bucket:   v.GetString("bucket"),
				Path:     v.GetString("path"),
				Google: &snapshottypes.StoreGoogle{
					ServiceAccount: v.GetString("service-account"),
				},
				KotsadmNamespace: namespace,
				RegistryConfig:   &registryConfig,
				SkipValidation:   v.GetBool("skip-validation"),
			}
			_, err = snapshot.ConfigureStore(cmd.Context(), configureStoreOptions)
			if err != nil {
				return errors.Wrap(err, "failed to configure store")
			}

			log.Info("\nStore Configured Successfully")

			return nil
		},
	}

	cmd.Flags().String("bucket", "", "name of the object storage bucket where backups should be stored (required)")
	cmd.Flags().String("path", "", "path to a subdirectory in the object store bucket")
	cmd.Flags().String("service-account", "", "the service account to use if using Google Cloud instance role (required)")
	cmd.Flags().Bool("skip-validation", false, "skip the validation of the bucket")

	cmd.MarkFlagRequired("bucket")
	cmd.MarkFlagRequired("service-account")

	return cmd
}

func VeleroConfigureAzureCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure-azure",
		Short: "Configure snapshots to use Azure Blob Storage",
	}

	cmd.AddCommand(VeleroConfigureAzureServicePrincipleCmd())

	return cmd
}

func VeleroConfigureAzureServicePrincipleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "service-principle",
		Short:         "Configure snapshots to use Azure Blob Storage with Service Principle Auth",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()
			log := logger.NewCLILogger(cmd.OutOrStdout())

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			registryConfig, err := kotsadm.GetRegistryConfigFromCluster(namespace, clientset)
			if err != nil {
				return errors.Wrap(err, "failed to get registry options from cluster")
			}

			veleroStatus, err := snapshot.DetectVelero(cmd.Context(), namespace)
			if err != nil {
				return errors.Wrap(err, "failed to detect velero")
			}
			if veleroStatus == nil {
				print.VeleroInstallationInstructionsForCLI(log, snapshottypes.VeleroAzurePlugin, &registryConfig, strings.Join(os.Args, " "))
				return nil
			}
			if !veleroStatus.ContainsPlugin("velero-plugin-for-microsoft-azure") {
				return errors.New("velero does not have the 'velero-plugin-for-microsoft-azure' plugin installed")
			}

			configureStoreOptions := snapshot.ConfigureStoreOptions{
				Provider: "azure",
				Bucket:   v.GetString("container"),
				Path:     v.GetString("path"),
				Azure: &snapshottypes.StoreAzure{
					ResourceGroup:  v.GetString("resource-group"),
					StorageAccount: v.GetString("storage-account"),
					SubscriptionID: v.GetString("subscription-id"),
					TenantID:       v.GetString("tenant-id"),
					ClientID:       v.GetString("client-id"),
					ClientSecret:   v.GetString("client-secret"),
					CloudName:      v.GetString("cloud-name"),
				},
				KotsadmNamespace: namespace,
				RegistryConfig:   &registryConfig,
				SkipValidation:   v.GetBool("skip-validation"),
			}

			_, err = snapshot.ConfigureStore(cmd.Context(), configureStoreOptions)
			if err != nil {
				return errors.Wrap(err, "failed to configure store")
			}

			log.Info("\nStore Configured Successfully")

			return nil
		},
	}

	cmd.Flags().String("client-id", "", "the client ID of a Service Principle with access to the blob storage container (required)")
	cmd.Flags().String("client-secret", "", "the client secret of a Service Principle with access to the blob storage container (required)")
	cmd.Flags().String("cloud-name", providers.AzureDefaultCloud, "the Azure cloud target. Options: "+
		"AzurePublicCloud, AzureUSGovernmentCloud, AzureChinaCloud, AzureGermanCloud")
	cmd.Flags().String("container", "", "name of the Azure blob storage container where backups should be stored (required)")
	cmd.Flags().String("path", "", "path to a subdirectory in the blob storage container")
	cmd.Flags().String("resource-group", "", "the resource group name of the blob storage container (required)")
	cmd.Flags().Bool("skip-validation", false, "skip the validation of the blob storage container")
	cmd.Flags().String("storage-account", "", "the storage account name of the blob storage container (required)")
	cmd.Flags().String("subscription-id", "", "the subscription id associated with the blob storage container (required)")
	cmd.Flags().String("tenant-id", "", "the tenant ID associated with the blob storage container (required)")

	cmd.MarkFlagRequired("client-id")
	cmd.MarkFlagRequired("client-secret")
	cmd.MarkFlagRequired("container")
	cmd.MarkFlagRequired("resource-group")
	cmd.MarkFlagRequired("storage-account")
	cmd.MarkFlagRequired("subscription-id")
	cmd.MarkFlagRequired("tenant-id")

	return cmd
}

func VeleroConfigureNFSCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "configure-nfs",
		Short:         "Configure snapshots to use NFS as storage destination",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			nfsPath := v.GetString("nfs-path")
			nfsServer := v.GetString("nfs-server")

			if nfsPath == "" {
				return errors.New("--nfs-path is required")
			}
			if nfsServer == "" {
				return errors.New("--nfs-server is required")
			}

			fileSystemConfig := snapshottypes.FileSystemConfig{
				NFS: &snapshottypes.NFSConfig{
					Path:   nfsPath,
					Server: nfsServer,
				},
			}

			registryConfig, err := getRegistryConfig(v, clientset, "")
			if err != nil {
				return errors.Wrap(err, "failed to get registry config")
			}
			if registryConfig.OverrideRegistry == "" {
				// check if there's already an existing registry configuration.
				rc, err := kotsadm.GetRegistryConfigFromCluster(namespace, clientset)
				if err != nil {
					return errors.Wrap(err, "failed to get registry options from cluster")
				}
				registryConfig = &rc
			}

			log := logger.NewCLILogger(cmd.OutOrStdout())

			opts := VeleroConfigureFileSystemOptions{
				Namespace:        namespace,
				RegistryConfig:   registryConfig,
				FileSystemConfig: fileSystemConfig,
				ForceReset:       v.GetBool("force-reset"),
				SkipValidation:   v.GetBool("skip-validation"),
				IsMinioDisabled:  !v.GetBool("with-minio"),
			}
			return veleroConfigureFileSystem(cmd.Context(), clientset, log, opts)
		},
	}

	cmd.Flags().String("nfs-path", "", "the path that is exported by the NFS server")
	cmd.Flags().String("nfs-server", "", "the hostname or IP address of the NFS server")
	cmd.Flags().Bool("force-reset", false, "bypass the reset prompt and force resetting the nfs path")
	cmd.Flags().Bool("with-minio", true, "when set, kots will deploy minio for NFS snapshot locations")
	cmd.Flags().Bool("skip-validation", false, "skip the validation of the backup store endpoint/bucket")
	cmd.Flags().MarkHidden("skip-validation")

	registryFlags(cmd.Flags())

	return cmd
}

func VeleroConfigureHostPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "configure-hostpath",
		Short:         "Configure snapshots to use a host path as storage destination",
		Long:          ``,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			hostPath := v.GetString("hostpath")
			if hostPath == "" {
				return errors.New("--hostpath option is required")
			}

			fileSystemConfig := snapshottypes.FileSystemConfig{
				HostPath: &hostPath,
			}

			registryConfig, err := getRegistryConfig(v, clientset, "")
			if err != nil {
				return errors.Wrap(err, "failed to get registry config")
			}
			if registryConfig.OverrideRegistry == "" {
				// check if there's already an existing registry configuration.
				rc, err := kotsadm.GetRegistryConfigFromCluster(namespace, clientset)
				if err != nil {
					return errors.Wrap(err, "failed to get registry options from cluster")
				}
				registryConfig = &rc
			}

			log := logger.NewCLILogger(cmd.OutOrStdout())

			opts := VeleroConfigureFileSystemOptions{
				Namespace:        namespace,
				RegistryConfig:   registryConfig,
				FileSystemConfig: fileSystemConfig,
				ForceReset:       v.GetBool("force-reset"),
				SkipValidation:   v.GetBool("skip-validation"),
				IsMinioDisabled:  !v.GetBool("with-minio"),
			}
			return veleroConfigureFileSystem(cmd.Context(), clientset, log, opts)
		},
	}

	cmd.Flags().String("hostpath", "", "a local host path on the node")
	cmd.Flags().Bool("force-reset", false, "bypass the reset prompt and force resetting the host path directory")
	cmd.Flags().Bool("skip-validation", false, "skip the validation of the backup store endpoint/bucket")
	cmd.Flags().Bool("with-minio", true, "when set, kots will deploy minio for hostpath snapshot locations")

	cmd.Flags().MarkHidden("skip-validation")

	registryFlags(cmd.Flags())

	return cmd
}

type VeleroConfigureFileSystemOptions struct {
	Namespace          string
	RegistryConfig     *kotsadmtypes.RegistryConfig
	FileSystemConfig   snapshottypes.FileSystemConfig
	ForceReset         bool
	SkipValidation     bool
	IsMinioDisabled    bool
	IsLegacyDeployment bool
}

func veleroConfigureFileSystem(ctx context.Context, clientset kubernetes.Interface, log *logger.CLILogger, opts VeleroConfigureFileSystemOptions) error {
	// Check for existing status; bail if not enabled
	isMinioDisabled, err := snapshot.IsFileSystemMinioDisabled(opts.Namespace)
	if err != nil {
		return errors.Wrap(err, "failed to check if file system minio is disabled")
	}
	if isMinioDisabled {
		opts.IsMinioDisabled = isMinioDisabled
	}

	veleroStatus, err := snapshot.DetectVelero(ctx, opts.Namespace)
	if err != nil {
		return errors.Wrap(err, "failed to detect velero")
	}

	if opts.IsMinioDisabled {
		if veleroStatus == nil || !veleroStatus.ContainsPlugin("local-volume-provider") {
			print.VeleroInstallationInstructionsForCLI(log, image.Lvp, opts.RegistryConfig, strings.Join(os.Args, " "))
			return nil
		}
	} else {
		if veleroStatus == nil || !veleroStatus.ContainsPlugin("plugin-for-aws") {
			print.VeleroInstallationInstructionsForCLI(log, snapshottypes.VeleroAWSPlugin, opts.RegistryConfig, strings.Join(os.Args, " "))
			return nil
		}
	}

	log.ActionWithoutSpinner("Setting up File System")

	deployOptions := snapshot.FileSystemDeployOptions{
		Namespace:        opts.Namespace,
		IsOpenShift:      k8sutil.IsOpenShift(clientset),
		FileSystemConfig: opts.FileSystemConfig,
		ForceReset:       opts.ForceReset,
	}

	if !opts.IsMinioDisabled {
		// Minio Case
		err = deployVeleroMinioFileSystem(ctx, clientset, log, deployOptions, opts)
		if err != nil {
			return errors.Wrap(err, "could not deploy minio fs")
		}
	} else {
		// LVP Case
		// Peak to see if this is a legacy minio deployment that was migrated
		isLegacyMinioDeployment, _, err := snapshot.ValidateFileSystemDeployment(ctx, clientset, deployOptions, *opts.RegistryConfig)
		if err != nil {
			return errors.Wrap(err, "could not validate lvp file system")
		}
		opts.IsLegacyDeployment = isLegacyMinioDeployment

		if err := snapshot.DeployFileSystemLvp(ctx, clientset, deployOptions, *opts.RegistryConfig); err != nil {
			return errors.Wrap(err, "could not deploy lvp file system config")
		}
	}

	log.ActionWithSpinner("Configuring Velero")

	configureStoreOptions := snapshot.ConfigureStoreOptions{
		FileSystem:        &opts.FileSystemConfig,
		KotsadmNamespace:  opts.Namespace,
		RegistryConfig:    opts.RegistryConfig,
		SkipValidation:    opts.SkipValidation,
		ValidateUsingAPod: true,
		IsMinioDisabled:   opts.IsMinioDisabled,
	}
	_, err = snapshot.ConfigureStore(ctx, configureStoreOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to configure store")
	}

	log.FinishSpinner()

	return nil
}

func deployVeleroMinioFileSystem(ctx context.Context, clientset kubernetes.Interface, log *logger.CLILogger, deployOptions snapshot.FileSystemDeployOptions, opts VeleroConfigureFileSystemOptions) error {
	log.ChildActionWithSpinner("Deploying File System Minio")
	if err := snapshot.DeployFileSystemMinio(ctx, clientset, deployOptions, *opts.RegistryConfig); err != nil {
		if _, ok := errors.Cause(err).(*snapshot.ResetFileSystemError); ok {
			forceReset := promptForFileSystemReset(log, err.Error())
			if forceReset {
				log.FinishChildSpinner()
				log.ChildActionWithSpinner("Re-configuring File System Minio")
				deployOptions.ForceReset = true
				if err := snapshot.DeployFileSystemMinio(ctx, clientset, deployOptions, *opts.RegistryConfig); err != nil {
					log.FinishChildSpinner()
					return errors.Wrap(err, "failed to force deploy file system minio")
				}
			}

		} else {
			log.FinishChildSpinner()
			return errors.Wrap(err, "failed to deploy file system minio")
		}
	}

	log.FinishChildSpinner()
	log.ChildActionWithSpinner("Waiting for File System Minio to be ready")

	err := k8sutil.WaitForDeploymentReady(ctx, clientset, opts.Namespace, snapshot.FileSystemMinioDeploymentName, time.Minute*2)
	if err != nil {
		log.FinishChildSpinner()
		return errors.Wrap(err, "failed to wait for file system minio")
	}

	log.FinishChildSpinner()
	log.ChildActionWithSpinner("Creating Default Bucket")

	err = snapshot.CreateFileSystemMinioBucket(ctx, clientset, opts.Namespace, *opts.RegistryConfig)
	if err != nil {
		log.FinishChildSpinner()
		return errors.Wrap(err, "failed to create default bucket")
	}

	log.FinishChildSpinner()
	return nil
}

// (DEPRECATED) VeleroPrintFileSystemInstrunctions prints instructions for setting up a file system (e.g. NFS, Host Path) as the snapshots storage destination
func VeleroPrintFileSystemInstructionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "print-fs-instructions",
		Short:         "Prints instructions for setting up a file system as the snapshots storage destination (e.g. NFS, Host Path, etc..)",
		Long:          `This command is deprecated and will be removed in a future release. Please us configure-hostpath or configure-nfs instead.`,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()
			log := logger.NewCLILogger(cmd.OutOrStdout())

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			registryConfig, err := getRegistryConfig(v, clientset, "")
			if err != nil {
				return errors.Wrap(err, "failed to get registry config")
			}
			if registryConfig.OverrideRegistry == "" {
				// check if there's already an existing registry configuration.
				rc, err := kotsadm.GetRegistryConfigFromCluster(namespace, clientset)
				if err != nil {
					return errors.Wrap(err, "failed to get registry options from cluster")
				}
				registryConfig = &rc
			}

			isMinioDisabled, err := snapshot.IsFileSystemMinioDisabled(namespace)
			if err != nil {
				return errors.Wrap(err, "failed to check for existing snapshot preference")
			}

			blue := color.New(color.FgHiBlue).SprintFunc()
			kotsConfigureCommand := fmt.Sprintf("	* To configure a host path as the storage destination, please refer to: %s\n", blue("https://docs.replicated.com/reference/kots-cli-velero-configure-hostpath"))
			kotsConfigureCommand += fmt.Sprintf("	* To configure NFS as the storage destination, please refer to: %s", blue("https://docs.replicated.com/reference/kots-cli-velero-configure-nfs"))

			if !v.GetBool("with-minio") || isMinioDisabled {
				print.VeleroInstallationInstructionsForCLI(log, image.Lvp, registryConfig, kotsConfigureCommand)
			} else {
				print.VeleroInstallationInstructionsForCLI(log, snapshottypes.VeleroAWSPlugin, registryConfig, kotsConfigureCommand)
			}

			return nil
		},
	}

	cmd.Flags().Bool("with-minio", true, "when set, kots will deploy minio for hostpath snapshot locations")

	registryFlags(cmd.Flags())

	return cmd
}

func promptForFileSystemReset(log *logger.CLILogger, warningMsg string) bool {
	// this is a workaround to avoid this issue: https://github.com/manifoldco/promptui/issues/122
	red := color.New(color.FgHiRed).SprintFunc()
	log.Info(fmt.Sprintf("\n%s", red(warningMsg)))

	prompt := promptui.Prompt{
		Label:     "Would you like to continue",
		IsConfirm: true,
	}

	for {
		resp, err := prompt.Run()
		if err == promptui.ErrInterrupt {
			os.Exit(-1)
		}
		if strings.ToLower(resp) == "n" {
			os.Exit(-1)
		}
		if strings.ToLower(resp) == "y" {
			log.ActionWithoutSpinner("")
			return true
		}
	}
}

func validateVeleroNamespace(namespace string) error {
	if namespace == "" {
		return errors.New("velero-namespace is required")
	}
	if strings.Contains(namespace, "_") {
		return errors.New("velero-namespace should not contain the _ character")
	}

	errs := validation.IsValidLabelValue(namespace)
	if len(errs) > 0 {
		return errors.New(errs[0])
	}

	return nil
}

// VeleroMigrateMinioFileSystemCmd is an internal command used by kURL to migrate
// minio filesystem snapshots to using the LVP plugin. kURL does not run the upgrade command
// directly
func VeleroMigrateMinioFileSystemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "migrate-minio-filesystems",
		Short:         "Migrates from Minio to Velero Local-Volume-Provider (LVP) plugin for filesystem snapshots (e.g. NFS, Host Path, etc..)",
		Long:          ``,
		Hidden:        true,
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			log := logger.NewCLILogger(cmd.OutOrStdout())

			namespace := v.GetString("namespace")
			if err := validateNamespace(namespace); err != nil {
				return err
			}

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			deployOptions, err := kotsadm.ReadDeployOptionsFromCluster(namespace, clientset)
			if err != nil {
				return errors.Wrap(err, "failed to read deploy options")
			}

			if !deployOptions.IncludeMinioSnapshots {
				// Already migrated, so this is a no-op
				log.Info("Snapshot migration not required")
				return nil
			}
			deployOptions.IncludeMinioSnapshots = false

			if err = kotsadm.MigrateExistingMinioFilesystemDeployments(log, deployOptions); err != nil {
				return errors.Wrap(err, "failed to complete migration")
			}

			log.Info("Saving snapshot preferences.")
			// Write back the new preference into the config. Needed for new kurl installs
			if err = kotsadm.EnsureConfigMaps(*deployOptions, clientset); err != nil {
				return errors.Wrap(err, "failed to update kotsadm config with new snapshot preference")
			}

			return nil
		},
	}

	return cmd
}
