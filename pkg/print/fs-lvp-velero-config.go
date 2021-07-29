package print

import (
	"encoding/json"
	"fmt"

	"github.com/fatih/color"
	"github.com/replicatedhq/kots/pkg/logger"
)

type LvpFileSystemVeleroConfig struct {
	IsHostPath           bool              `json:"isHostPath"`
	Provider             string            `json:"provider"`
	Bucket               string            `json:"bucket"`
	Prefix               string            `json:"prefix"`
	BackupLocationConfig map[string]string `json:"backupLocationConfig"`
}

func LvpFileSystemVeleroInfo(c *LvpFileSystemVeleroConfig, format string, log *logger.CLILogger) {
	switch format {
	case "json":
		printLvpFileSystemVeleroConfigJSON(c)
	default:
		printLvpFileSystemVeleroInstructions(c, log)
	}
}

func printLvpFileSystemVeleroConfigJSON(c *LvpFileSystemVeleroConfig) {
	str, _ := json.MarshalIndent(c, "", "    ")
	fmt.Println(string(str))
}

func printLvpFileSystemVeleroInstructions(c *LvpFileSystemVeleroConfig, log *logger.CLILogger) {
	blue := color.New(color.FgHiBlue).SprintFunc()
	red := color.New(color.FgHiRed).SprintFunc()
	bold := color.New(color.FgGreen, color.Bold).SprintFunc()

	var prefix string
	if c.Prefix != "" {
		prefix = fmt.Sprintf("\n\t--prefix %s \\\n", c.Prefix)
	} else {
		prefix = ""
	}

	var backupConfig string
	if c.IsHostPath {
		backupConfig = fmt.Sprintf("path=%s,resticRepoPrefix=%s", c.BackupLocationConfig["path"], c.BackupLocationConfig["resticRepo"])
	} else {
		backupConfig = fmt.Sprintf("path=%s,server=%s,resticRepoPrefix=%s", c.BackupLocationConfig["path"], c.BackupLocationConfig["server"], c.BackupLocationConfig["resticRepo"])
	}

	veleroOnlineCommand := fmt.Sprintf(`velero install \
	--no-secret \
	--provider %s \
	--plugins replicated/local-volume-provider:v0.1.0 \
	--bucket %s \%s
	--backup-location-config %s \
	--use-restic`, c.Provider, c.Bucket, prefix, backupConfig)

	lvpConfigMap := `
	apiVersion: v1
	kind: ConfigMap
	metadata:
	name: local-volume-provider-config
	namespace: velero
	labels:
	  velero.io/plugin-config: ""
	  replicated.com/nfs: ObjectStore
	  replicated.com/hostpath: ObjectStore
	data:
	  securityContextRunAsUser: "1001"
	  securityContextFsGroup: "1001"
	`

	veleroAirgapCommand := fmt.Sprintf(`velero install \
	--no-secret \
	--provider %s \
	--image %s/velero:%s \
	--plugins %s/local-volume-provider:v0.1.0 \
	--bucket %s \%s
	--backup-location-config %s \
	--use-restic`, c.Provider, red("<private.registry.host>"), red("<velero-version>"), red("<private.registry.host>"), c.Bucket, prefix, backupConfig)

	airgapLvpConfigMap := fmt.Sprintf(`
	apiVersion: v1
	kind: ConfigMap
	metadata:
	name: local-volume-provider-config
	namespace: velero
	labels:
	  velero.io/plugin-config: ""
	  replicated.com/nfs: ObjectStore
	  replicated.com/hostpath: ObjectStore
	data:
	  fileserverImage: %s/local-volume-fileserver:v0.1.0
	  securityContextRunAsUser: "1001"
	  securityContextFsGroup: "1001"
	`, red("<private.registry.host>"))

	log.ActionWithoutSpinner("Follow these instructions to set up Velero:\n")
	log.Info("[1] Install the latest Velero CLI by following these instructions: %s", blue("https://velero.io/docs/v1.6/basic-install/#install-the-cli"))
	log.Info("[2] Install Velero")
	log.Info("- For %s, follow these steps:", bold("online installations"))
	log.Info("	* Run the this Velero install command: \n\n%s", veleroOnlineCommand)
	log.Info("	* For all clusters EXCEPT Openshift, create the following ConfigMap: \n%s", lvpConfigMap)
	log.Info("- For %s, follow these steps:", bold("airgapped installations"))
	log.Info("	* Prepare velero images (you will need %s and %s images for plugins): %s",
		red("replicated/local-volume-provider:v0.1.0"),
		red("replicated/local-volume-filesever:v0.1.0"),
		blue("https://velero.io/docs/v1.6/on-premises/#air-gapped-deployments"))
	log.Info("	* Install velero (replace with actual values): \n\n%s", veleroAirgapCommand)
	log.Info("	* Configure restic restore helper to use the prepared image: %s", blue("https://velero.io/docs/v1.6/restic/#customize-restore-helper-container"))
	log.Info("	* Create the following ConfigMap. If you are installing into OpenShift, remove the 'securityContext*' lines:: \n%s", airgapLvpConfigMap)
	log.Info("[4] If you're using RancherOS, OpenShift, Microsoft Azure, or VMware Tanzu Kubernetes Grid Integrated Edition (formerly VMware Enterprise PKS), please refer to the following Velero doc to complete restic configuration: %s", blue("https://velero.io/docs/v1.6/restic/#configure-restic-daemonset-spec"))
	log.ActionWithoutSpinner("")
}
