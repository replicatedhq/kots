package print

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/replicatedhq/kots/pkg/imageutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	"github.com/replicatedhq/kots/pkg/logger"
	snapshottypes "github.com/replicatedhq/kots/pkg/snapshot/types"
)

func VeleroInstallationInstructionsForCLI(log *logger.CLILogger, plugin snapshottypes.VeleroPlugin, registryConfig *kotsadmtypes.RegistryConfig, kotsConfigureCommand string) {
	blue := color.New(color.FgHiBlue).SprintFunc()
	red := color.New(color.FgHiRed).SprintFunc()

	if registryConfig.OverrideRegistry == "" {
		// this is an online installation
		veleroOnlineCommand := fmt.Sprintf(`velero install \
		--no-default-backup-location \
		--no-secret \
		--use-node-agent \
		--uploader-type=restic \
		--use-volume-snapshots=false \
		--plugins %s`, plugin)

		log.ActionWithoutSpinner("No Velero installation has been detected.")
		log.ActionWithoutSpinner("Follow these instructions to set up Velero:\n")
		log.Info("[1] Install the latest Velero CLI: %s", blue("https://velero.io/docs/v1.10/basic-install/#install-the-cli"))
		log.Info("[2] Install Velero: \n\n%s", veleroOnlineCommand)
		log.Info("[3] If you're using RancherOS, OpenShift, Microsoft Azure, or VMware Tanzu Kubernetes Grid Integrated Edition (formerly VMware Enterprise PKS), please refer to the following Velero doc to complete node agent configuration: %s", blue("https://velero.io/docs/v1.10/file-system-backup/#configure-node-agent-daemonset-spec"))
		log.Info("[4] Configure the backup storage location: \n\n%s", kotsConfigureCommand)
		log.ActionWithoutSpinner("")
		return
	}

	// this is an airgapped installation
	registry := kotsadmversion.KotsadmRegistry(*registryConfig)
	pluginName := strings.Split(strings.Split(string(plugin), "/")[1], ":")[0]
	pluginTag, _ := imageutil.GetTag(string(plugin))

	veleroAirgapCommand := fmt.Sprintf(`velero install \
	--no-default-backup-location \
	--no-secret \
	--use-node-agent \
	--uploader-type=restic \
	--use-volume-snapshots=false \
	--image %s/velero:%s \
	--plugins %s/%s:%s`, registry, red("<velero-version>"), registry, pluginName, pluginTag)

	log.ActionWithoutSpinner("No Velero installation has been detected.")
	log.ActionWithoutSpinner("Follow these instructions to set up Velero:\n")
	log.Info("[1] Install the latest Velero CLI: %s", blue("https://velero.io/docs/v1.10/basic-install/#install-the-cli"))
	log.Info("[2] Install Velero")
	log.Info("	* Prepare velero images (you will need %s for plugins): %s", red(plugin), blue("https://velero.io/docs/v1.10/on-premises/#air-gapped-deployments"))
	log.Info("	* Install velero (replace <velero-version> with actual version): \n\n%s", veleroAirgapCommand)
	log.Info("	* Configure the restore helper to use the prepared image: %s", blue("https://velero.io/docs/v1.10/file-system-backup/#customize-restore-helper-container"))
	log.Info("[3] If you're using RancherOS, OpenShift, Microsoft Azure, or VMware Tanzu Kubernetes Grid Integrated Edition (formerly VMware Enterprise PKS), please refer to the following Velero doc to complete node agent configuration: %s", blue("https://velero.io/docs/v1.10/file-system-backup/#configure-node-agent-daemonset-spec"))
	log.Info("[4] Configure the backup storage location: \n\n%s", kotsConfigureCommand)
	log.ActionWithoutSpinner("")
}

type VeleroInstallationInstruction struct {
	Title  string                            `json:"title"`
	Action string                            `json:"action"`
	Type   VeleroInstallationInstructionType `json:"type"`
}

type VeleroInstallationInstructionType string

const (
	VeleroInstallationInstructionCommand VeleroInstallationInstructionType = "command"
	VeleroInstallationInstructionLink    VeleroInstallationInstructionType = "link"
)

func VeleroInstallationInstructionsForUI(plugin snapshottypes.VeleroPlugin, registryConfig *kotsadmtypes.RegistryConfig, kotsConfigureCommand string) []VeleroInstallationInstruction {
	if registryConfig.OverrideRegistry == "" {
		// this is an online installation
		veleroOnlineCommand := fmt.Sprintf(`velero install --no-default-backup-location --no-secret --use-node-agent --uploader-type=restic --use-volume-snapshots=false --plugins %s`, plugin)

		return []VeleroInstallationInstruction{
			{
				Title:  "Install Velero",
				Action: veleroOnlineCommand,
				Type:   "command",
			},
			{
				Title:  "If you're using RancherOS, OpenShift, Microsoft Azure, or VMware Tanzu Kubernetes Grid Integrated Edition (formerly VMware Enterprise PKS), please refer to the following Velero doc to complete node agent configuration",
				Action: "https://velero.io/docs/v1.10/file-system-backup/#configure-node-agent-daemonset-spec",
				Type:   "link",
			},
			{
				Title:  "Configure the backup storage location",
				Action: kotsConfigureCommand,
				Type:   "command",
			},
		}
	}

	// this is an airgapped installation
	registry := kotsadmversion.KotsadmRegistry(*registryConfig)
	pluginName := strings.Split(strings.Split(string(plugin), "/")[1], ":")[0]
	pluginTag, _ := imageutil.GetTag(string(plugin))

	veleroAirgapCommand := fmt.Sprintf(`velero install --no-default-backup-location --no-secret --use-node-agent --uploader-type=restic --use-volume-snapshots=false --image %s/velero:%s --plugins %s/%s:%s`, registry, "<velero-version>", registry, pluginName, pluginTag)

	return []VeleroInstallationInstruction{
		{
			Title:  fmt.Sprintf("Prepare velero images (you will need %s for plugins)", plugin),
			Action: "https://velero.io/docs/v1.10/on-premises/#air-gapped-deployments",
			Type:   "link",
		},
		{
			Title:  "Install velero (replace <velero-version> with actual version)",
			Action: veleroAirgapCommand,
			Type:   "command",
		},
		{
			Title:  "Configure the restore helper to use the prepared image",
			Action: "https://velero.io/docs/v1.10/file-system-backup/#customize-restore-helper-container",
			Type:   "link",
		},
		{
			Title:  "If you're using RancherOS, OpenShift, Microsoft Azure, or VMware Tanzu Kubernetes Grid Integrated Edition (formerly VMware Enterprise PKS), please refer to the following Velero doc to complete node agent configuration",
			Action: "https://velero.io/docs/v1.10/file-system-backup/#configure-node-agent-daemonset-spec",
			Type:   "link",
		},
		{
			Title:  "Configure the backup storage location",
			Action: kotsConfigureCommand,
			Type:   "command",
		},
	}
}
