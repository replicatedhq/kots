package print

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/replicatedhq/kots/pkg/logger"
)

type FileSystemVeleroConfig struct {
	Provider               string            `json:"provider"`
	Plugins                []string          `json:"plugins"`
	Credentials            []byte            `json:"credentials"`
	Bucket                 string            `json:"bucket"`
	BackupLocationConfig   map[string]string `json:"backupLocationConfig"`
	SnapshotLocationConfig map[string]string `json:"snapshotLocationConfig"`
	UseRestic              bool              `json:"useRestic"`
}

func FileSystemMinioVeleroInfo(c *FileSystemVeleroConfig, format string, log *logger.CLILogger) {
	switch format {
	case "json":
		printFileSystemVeleroConfigJSON(c)
	default:
		printFileSystemVeleroInstructions(c, log)
	}
}

func printFileSystemVeleroConfigJSON(c *FileSystemVeleroConfig) {
	str, _ := json.MarshalIndent(c, "", "    ")
	fmt.Println(string(str))
}

func printFileSystemVeleroInstructions(c *FileSystemVeleroConfig, log *logger.CLILogger) {
	green := color.New(color.FgGreen).SprintFunc()
	blue := color.New(color.FgHiBlue).SprintFunc()
	red := color.New(color.FgHiRed).SprintFunc()
	bold := color.New(color.FgGreen, color.Bold).SprintFunc()

	veleroOnlineCommand := fmt.Sprintf(`velero install \
	--secret-file %s \
	--provider aws \
	--plugins velero/velero-plugin-for-aws:v1.2.0 \
	--bucket %s \
	--backup-location-config region=%s,s3ForcePathStyle=true,s3Url=%s,publicUrl=%s \
	--snapshot-location-config region=%s \
	--use-restic`, red("<path/to/credentials-file>"), c.Bucket, c.BackupLocationConfig["region"], c.BackupLocationConfig["s3Url"], c.BackupLocationConfig["publicUrl"], c.SnapshotLocationConfig["region"])

	veleroAirgapCommand := fmt.Sprintf(`velero install \
	--secret-file %s \
	--provider aws \
	--image %s/velero:%s \
	--plugins %s/velero-plugin-for-aws:%s \
	--bucket %s \
	--backup-location-config region=%s,s3ForcePathStyle=true,s3Url=%s,publicUrl=%s \
	--snapshot-location-config region=%s \
	--use-restic`, red("<path/to/credentials-file>"), red("<private.registry.host>"), red("<velero-version>"), red("<private.registry.host>"), red("<plugin-version>"), c.Bucket, c.BackupLocationConfig["region"], c.BackupLocationConfig["s3Url"], c.BackupLocationConfig["publicUrl"], c.SnapshotLocationConfig["region"])

	log.ActionWithoutSpinner("Follow these instructions to set up Velero:\n")
	log.Info("[1] Save the following credentials in a file:\n\n%s", green(strings.TrimSpace(string(c.Credentials))))
	log.Info("[2] Install the latest Velero CLI by following these instructions: %s", blue("https://velero.io/docs/v1.6/basic-install/#install-the-cli"))
	log.Info("[3] Install Velero")
	log.Info("- For %s, run the following command (replace <path/to/credentials-file> with the actual path created from step [1]):\n\n%s", bold("online installations"), veleroOnlineCommand)
	log.Info("- For %s, follow these steps:", bold("airgapped installations"))
	log.Info("	* Prepare velero images: %s", blue("https://velero.io/docs/v1.6/on-premises/#air-gapped-deployments"))
	log.Info("	* Install velero (replace with actual values): \n\n%s", veleroAirgapCommand)
	log.Info("	* Configure restic restore helper to use the prepared image: %s", blue("https://velero.io/docs/v1.6/restic/#customize-restore-helper-container"))
	log.Info("[4] If you're using RancherOS, OpenShift, Microsoft Azure, or VMware Tanzu Kubernetes Grid Integrated Edition (formerly VMware Enterprise PKS), please refer to the following Velero doc to complete restic configuration: %s", blue("https://velero.io/docs/v1.6/restic/#configure-restic-daemonset-spec"))
	log.ActionWithoutSpinner("")
}
