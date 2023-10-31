package types

const KotsadmKey = "kots.io/kotsadm"
const KotsadmLabelValue = "true"

const ClusterTokenSecret = "kotsadm-cluster-token"
const PrivateKotsadmRegistrySecret = "kotsadm-private-registry"
const KotsadmConfigMap = "kotsadm-confg"

const ExcludeKey = "velero.io/exclude-from-backup"
const ExcludeValue = "true"

const BackupLabel = "kots.io/backup"
const BackupLabelValue = "velero"

const TroubleshootKey = "troubleshoot.sh/kind"
const TroubleshootValue = "support-bundle"

const DefaultSupportBundleSpecKey = "default"
const ClusterSpecificSupportBundleSpecKey = "cluster-specific"
const VendorSpecificSupportBundleSpecKey = "vendor"

func GetKotsadmLabels(additionalLabels ...map[string]string) map[string]string {
	labels := map[string]string{
		KotsadmKey:  KotsadmLabelValue,
		BackupLabel: BackupLabelValue,
	}

	for _, l := range additionalLabels {
		labels = MergeLabels(labels, l)
	}

	return labels
}

func GetTroubleshootLabels(additionalLabels ...map[string]string) map[string]string {
	labels := map[string]string{
		TroubleshootKey: TroubleshootValue,
	}

	for _, l := range additionalLabels {
		labels = MergeLabels(labels, l)
	}

	return labels
}

func MergeLabels(labels ...map[string]string) map[string]string {
	mergedLabels := map[string]string{}

	for _, label := range labels {
		for k, v := range label {
			mergedLabels[k] = v
		}
	}

	return mergedLabels
}
