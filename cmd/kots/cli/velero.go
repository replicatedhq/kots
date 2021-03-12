package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/validation"

	"github.com/fatih/color"
	"github.com/manifoldco/promptui"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/print"
	"github.com/replicatedhq/kots/pkg/snapshot"
	snapshottypes "github.com/replicatedhq/kots/pkg/snapshot/types"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
)

func VeleroCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "velero",
		Short: "KOTS Velero interface",
	}

	cmd.AddCommand(VeleroEnsurePermissionsCmd())
	cmd.AddCommand(VeleroConfigureNFSCmd())
	cmd.AddCommand(VeleroConfigureHostPathCmd())
	cmd.AddCommand(VeleroPrintFileSystemInstructionsCmd())

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

			kotsadmNamespace := v.GetString("namespace")
			if err := validateNamespace(kotsadmNamespace); err != nil {
				return err
			}

			clientset, err := k8sutil.GetClientset(kubernetesConfigFlags)
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

	cmd.Flags().StringP("namespace", "n", "", "namespace in which kots/kotsadm is installed")
	cmd.Flags().String("velero-namespace", "", "namespace in which velero is installed")

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

			registryOptions, err := getRegistryConfig(v)
			if err != nil {
				return errors.Wrap(err, "failed to get registry config")
			}

			log := logger.NewCLILogger()

			opts := VeleroConfigureFileSystemOptions{
				Namespace:        namespace,
				RegistryOptions:  registryOptions,
				FileSystemConfig: fileSystemConfig,
				Output:           v.GetString("output"),
				ForceReset:       v.GetBool("force-reset"),
				SkipValidation:   v.GetBool("skip-validation"),
			}
			return veleroConfigureFileSystem(cmd.Context(), log, opts)
		},
	}

	cmd.Flags().String("nfs-path", "", "the path that is exported by the NFS server")
	cmd.Flags().String("nfs-server", "", "the hostname or IP address of the NFS server")
	cmd.Flags().StringP("namespace", "n", "", "the namespace in which kots/kotsadm is installed")
	cmd.Flags().StringP("output", "o", "", "output format. supported values: json")
	cmd.Flags().Bool("force-reset", false, "bypass the reset prompt and force resetting the nfs path")
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

			hostPath := v.GetString("hostpath")

			if hostPath == "" {
				return errors.New("--hostpath option is required")
			}

			fileSystemConfig := snapshottypes.FileSystemConfig{
				HostPath: &hostPath,
			}

			registryOptions, err := getRegistryConfig(v)
			if err != nil {
				return errors.Wrap(err, "failed to get registry config")
			}

			log := logger.NewCLILogger()

			opts := VeleroConfigureFileSystemOptions{
				Namespace:        namespace,
				RegistryOptions:  registryOptions,
				FileSystemConfig: fileSystemConfig,
				Output:           v.GetString("output"),
				ForceReset:       v.GetBool("force-reset"),
				SkipValidation:   v.GetBool("skip-validation"),
			}
			return veleroConfigureFileSystem(cmd.Context(), log, opts)
		},
	}

	cmd.Flags().String("hostpath", "", "a local host path on the node")
	cmd.Flags().StringP("namespace", "n", "", "the namespace in which kots/kotsadm is installed")
	cmd.Flags().StringP("output", "o", "", "output format. supported values: json")
	cmd.Flags().Bool("force-reset", false, "bypass the reset prompt and force resetting the host path directory")
	cmd.Flags().Bool("skip-validation", false, "skip the validation of the backup store endpoint/bucket")
	cmd.Flags().MarkHidden("skip-validation")

	registryFlags(cmd.Flags())

	return cmd
}

type VeleroConfigureFileSystemOptions struct {
	Namespace        string
	RegistryOptions  *kotsadmtypes.KotsadmOptions
	FileSystemConfig snapshottypes.FileSystemConfig
	Output           string
	ForceReset       bool
	SkipValidation   bool
}

func veleroConfigureFileSystem(ctx context.Context, log *logger.CLILogger, opts VeleroConfigureFileSystemOptions) error {
	log.ActionWithSpinner("Setting up File System Minio")

	clientset, err := k8sutil.GetClientset(kubernetesConfigFlags)
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	deployOptions := snapshot.FileSystemDeployOptions{
		Namespace:        opts.Namespace,
		IsOpenShift:      k8sutil.IsOpenShift(clientset),
		FileSystemConfig: opts.FileSystemConfig,
		ForceReset:       opts.ForceReset,
	}
	if err := snapshot.DeployFileSystemMinio(ctx, clientset, deployOptions, *opts.RegistryOptions); err != nil {
		if _, ok := errors.Cause(err).(*snapshot.ResetFileSystemError); ok {
			log.FinishSpinnerWithWarning(color.New(color.FgHiRed))
			forceReset := promptForFileSystemReset(log, err.Error())
			if forceReset {
				log.ActionWithSpinner("Re-configuring File System Minio")
				deployOptions.ForceReset = true
				if err := snapshot.DeployFileSystemMinio(ctx, clientset, deployOptions, *opts.RegistryOptions); err != nil {
					log.FinishSpinnerWithError()
					return errors.Wrap(err, "failed to force deploy file system minio")
				}
			}
		} else {
			log.FinishSpinnerWithError()
			return errors.Wrap(err, "failed to deploy file system minio")
		}
	}

	log.FinishSpinner()
	log.ActionWithSpinner("Waiting for File System Minio to be ready")

	err = k8sutil.WaitForDeploymentReady(ctx, clientset, opts.Namespace, snapshot.FileSystemMinioDeploymentName, time.Minute*2)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to wait for file system minio")
	}

	log.FinishSpinner()
	log.ActionWithSpinner("Creating Default Bucket")

	err = snapshot.CreateFileSystemMinioBucket(ctx, clientset, opts.Namespace, *opts.RegistryOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to create default bucket")
	}

	log.FinishSpinner()

	veleroNamespace, err := snapshot.DetectVeleroNamespace(ctx, clientset, opts.Namespace)
	if err != nil {
		return errors.Wrap(err, "failed to detect velero namespace")
	}
	if veleroNamespace == "" {
		c, err := buildPrintableFileSystemVeleroConfig(ctx, clientset, opts.Namespace)
		if err != nil {
			return errors.Wrap(err, "failed to get printable file system velero config")
		}
		if opts.Output != "json" {
			log.ActionWithoutSpinner("file system configuration for the Admin Console is successful, but no Velero installation has been detected.")
		}
		print.FileSystemMinioVeleroInfo(c, opts.Output, log)
		return nil
	}

	log.ActionWithSpinner("Configuring Velero")

	configureStoreOptions := snapshot.ConfigureStoreOptions{
		FileSystem:        true,
		KotsadmNamespace:  opts.Namespace,
		RegistryOptions:   opts.RegistryOptions,
		SkipValidation:    opts.SkipValidation,
		ValidateUsingAPod: true,
	}
	_, err = snapshot.ConfigureStore(ctx, configureStoreOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to configure store")
	}

	log.FinishSpinner()

	return nil
}

func VeleroPrintFileSystemInstructionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "print-fs-instructions",
		Short:         "Print instructions for setting up Velero with the current file system configuration (e.g. NFS, Host Path, etc..)",
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

			clientset, err := k8sutil.GetClientset(kubernetesConfigFlags)
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			c, err := buildPrintableFileSystemVeleroConfig(cmd.Context(), clientset, namespace)
			if err != nil {
				return errors.Wrap(err, "failed to get file system minio velero config")
			}

			log := logger.NewCLILogger()
			print.FileSystemMinioVeleroInfo(c, v.GetString("output"), log)

			return nil
		},
	}

	cmd.Flags().StringP("namespace", "n", "", "the namespace in which kots/kotsadm is installed")
	cmd.Flags().StringP("output", "o", "", "output format. supported values: json")

	return cmd
}

func buildPrintableFileSystemVeleroConfig(ctx context.Context, clientset kubernetes.Interface, namespace string) (*print.FileSystemVeleroConfig, error) {
	fileSystemStore, err := snapshot.BuildStoreFileSystem(ctx, clientset, namespace)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build file system store")
	}

	creds, err := snapshot.BuildAWSCredentials(fileSystemStore.AccessKeyID, fileSystemStore.SecretAccessKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to format credentials")
	}

	publicURL := fmt.Sprintf("http://%s:%d", fileSystemStore.ObjectStoreClusterIP, snapshot.FileSystemMinioServicePort)
	s3URL := fileSystemStore.Endpoint

	c := print.FileSystemVeleroConfig{
		Provider:    "aws",
		Plugins:     []string{"velero/velero-plugin-for-aws:v1.1.0"},
		Credentials: creds,
		Bucket:      snapshot.FileSystemMinioBucketName,
		BackupLocationConfig: map[string]string{
			"region":           snapshot.FileSystemMinioRegion,
			"s3Url":            s3URL,
			"publicUrl":        publicURL,
			"s3ForcePathStyle": "true",
		},
		SnapshotLocationConfig: map[string]string{
			"region": snapshot.FileSystemMinioRegion,
		},
		UseRestic: true,
	}

	return &c, nil
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
