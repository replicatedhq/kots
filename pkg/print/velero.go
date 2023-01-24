package print

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/replicatedhq/kots/pkg/image"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	"github.com/replicatedhq/kots/pkg/logger"
)

func VeleroInstallationInstructions(log *logger.CLILogger, plugin string, registryConfig *kotsadmtypes.RegistryConfig) {
	kotsCommand := strings.Join(os.Args, " ")

	blue := color.New(color.FgHiBlue).SprintFunc()
	red := color.New(color.FgHiRed).SprintFunc()

	if registryConfig.OverrideRegistry == "" {
		// this is an online installation
		veleroOnlineCommand := fmt.Sprintf(`velero install \
		--no-default-backup-location \
		--no-secret \
		--use-restic \
		--use-volume-snapshots=false \
		--plugins %s`, plugin)

		log.ActionWithoutSpinner("No Velero installation has been detected.")
		log.ActionWithoutSpinner("Follow these instructions to set up Velero:\n")
		log.Info("[1] Install the latest Velero CLI: %s", blue("https://velero.io/docs/v1.9/basic-install/#install-the-cli"))
		log.Info("[2] Install Velero: \n\n%s", veleroOnlineCommand)
		log.Info("[3] If you're using RancherOS, OpenShift, Microsoft Azure, or VMware Tanzu Kubernetes Grid Integrated Edition (formerly VMware Enterprise PKS), please refer to the following Velero doc to complete restic configuration: %s", blue("https://velero.io/docs/v1.9/restic/#configure-restic-daemonset-spec"))
		log.Info("[4] Configure the backup storage location: \n\n%s", kotsCommand)
		log.ActionWithoutSpinner("")
		return
	}

	// this is an airgapped installation
	registry := kotsadmversion.KotsadmRegistry(*registryConfig)
	pluginName := strings.Split(strings.Split(plugin, "/")[1], ":")[0]
	pluginTag, _ := image.GetTag(plugin)

	veleroAirgapCommand := fmt.Sprintf(`velero install \
	--no-default-backup-location \
	--no-secret \
	--use-restic \
	--use-volume-snapshots=false \
	--image %s/velero:%s \
	--plugins %s/%s:%s`, registry, red("<velero-version>"), registry, pluginName, pluginTag)

	log.ActionWithoutSpinner("No Velero installation has been detected.")
	log.ActionWithoutSpinner("Follow these instructions to set up Velero:\n")
	log.Info("[1] Install the latest Velero CLI: %s", blue("https://velero.io/docs/v1.9/basic-install/#install-the-cli"))
	log.Info("[2] Install Velero")
	log.Info("	* Prepare velero images (you will need %s for plugins): %s", red(plugin), blue("https://velero.io/docs/v1.9/on-premises/#air-gapped-deployments"))
	log.Info("	* Install velero (replace <velero-version> with actual version): \n\n%s", veleroAirgapCommand)
	log.Info("	* Configure restic restore helper to use the prepared image: %s", blue("https://velero.io/docs/v1.9/restic/#customize-restore-helper-container"))
	log.Info("[3] If you're using RancherOS, OpenShift, Microsoft Azure, or VMware Tanzu Kubernetes Grid Integrated Edition (formerly VMware Enterprise PKS), please refer to the following Velero doc to complete restic configuration: %s", blue("https://velero.io/docs/v1.9/restic/#configure-restic-daemonset-spec"))
	log.Info("[4] Configure the backup storage location: \n\n%s", kotsCommand)
	log.ActionWithoutSpinner("")
}
