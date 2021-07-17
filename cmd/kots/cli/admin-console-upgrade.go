package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func AdminConsoleUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "upgrade",
		Short:         "Upgrade the admin console to the latest version",
		Long:          "Upgrade the admin console to the latest version",
		SilenceUsage:  true,
		SilenceErrors: false,
		PreRun: func(cmd *cobra.Command, args []string) {
			viper.BindPFlags(cmd.Flags())
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			v := viper.GetViper()

			if v.GetBool("force-upgrade-kurl") {
				prompt := promptui.Prompt{
					Label:     fmt.Sprintf("Upgrading a kotsadm instance created by kURL can result in data loss. Do you want to continue"),
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
						break
					}
				}
			}

			namespace := v.GetString("namespace")

			clientset, err := k8sutil.GetClientset()
			if err != nil {
				return errors.Wrap(err, "failed to get clientset")
			}

			includeMinio := v.GetBool("with-minio")
			_, err = clientset.AppsV1().StatefulSets(namespace).Get(cmd.Context(), "kotsadm", metav1.GetOptions{})
			if err == nil {
				// reverse migration is not supported
				includeMinio = false
			}

			simultaneousUploads, _ := strconv.Atoi(v.GetString("airgap-upload-parallelism"))

			upgradeOptions := kotsadmtypes.UpgradeOptions{
				Namespace:                 namespace,
				ForceUpgradeKurl:          v.GetBool("force-upgrade-kurl"),
				EnsureRBAC:                v.GetBool("ensure-rbac"),
				SimultaneousUploads:       simultaneousUploads,
				StorageBaseURI:            v.GetString("storage-base-uri"),
				StorageBaseURIPlainHTTP:   v.GetBool("storage-base-uri-plainhttp"),
				IncludeMinio:              includeMinio,
				IncludeDockerDistribution: v.GetBool("with-dockerdistribution"),

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

			upgradeOptions.Timeout = timeout

			log := logger.NewCLILogger()
			if !v.GetBool("skip-rbac-check") && v.GetBool("ensure-rbac") {
				err := CheckRBAC()
				if err != nil {
					log.Errorf("Current user has insufficient privileges to upgrade Admin Console.\nFor more information, please visit https://kots.io/vendor/packaging/rbac\nTo bypass this check, use the --skip-rbac-check flag")
					return errors.New("insufficient privileges")
				}
			}

			if upgradeOptions.Namespace != "default" {
				log.ActionWithoutSpinner("Upgrading Admin Console")
			} else {
				log.ActionWithoutSpinner("Upgrading Admin Console in the default namespace")
			}
			if err := kotsadm.Upgrade(clientset, upgradeOptions); err != nil {
				return errors.Wrap(err, "failed to upgrade")
			}

			log.ActionWithoutSpinner("")
			log.ActionWithoutSpinner("The Admin Console is running the latest version")
			log.ActionWithoutSpinner("To access the Admin Console, run kubectl kots admin-console --namespace %s", namespace)
			log.ActionWithoutSpinner("")

			return nil
		},
	}

	cmd.Flags().Bool("force-upgrade-kurl", false, "set to force upgrade even if this is a kurl cluster")
	cmd.Flags().String("kotsadm-tag", "", "set to override the tag of kotsadm. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().String("kotsadm-registry", "", "set to override the registry of kotsadm images. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().String("registry-username", "", "user name to use to authenticate with the registry")
	cmd.Flags().String("registry-password", "", "password to use to authenticate with the registry")
	cmd.Flags().String("kotsadm-namespace", "", "set to override the namespace of kotsadm images. this may create an incompatible deployment because the version of kots and kotsadm are designed to work together")
	cmd.Flags().String("wait-duration", "3m", "timeout out to be used while waiting for individual components to be ready.  must be in Go duration format (eg: 10s, 2m)")
	cmd.Flags().Bool("ensure-rbac", true, "when set, kots will create the roles and rolebindings necessary to manage applications")
	cmd.Flags().String("airgap-upload-parallelism", "", "the number of chunks to upload in parallel when installing or updating in airgap mode")
	cmd.Flags().MarkHidden("force-upgrade-kurl")
	cmd.Flags().MarkHidden("kotsadm-tag")
	cmd.Flags().MarkHidden("kotsadm-namespace")
	cmd.Flags().MarkHidden("airgap-upload-parallelism")

	// options for the alpha feature of using a reg instead of s3 for storage
	cmd.Flags().String("storage-base-uri", "", "an s3 or oci-registry uri to use for kots persistent storage in the cluster")
	cmd.Flags().Bool("with-minio", true, "when set, kots will deploy a local minio instance for storage")
	cmd.Flags().Bool("with-dockerdistribution", false, "when set, kots install will deploy a local instance of docker distribution for storage")
	cmd.Flags().Bool("storage-base-uri-plainhttp", false, "when set, use plain http (not https) connecting to the local oci storage")
	cmd.Flags().MarkHidden("storage-base-uri")
	cmd.Flags().MarkHidden("with-dockerdistribution")
	cmd.Flags().MarkHidden("storage-base-uri-plainhttp")

	// option to check if the user has cluster-wide previliges to install application
	cmd.Flags().Bool("skip-rbac-check", false, "set to true to bypass rbac check")
	return cmd
}
