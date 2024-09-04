package types

import "github.com/replicatedhq/kots/pkg/util"

const KotsadmKey = "kots.io/kotsadm"
const KotsadmLabelValue = "true"

const ClusterTokenSecret = "kotsadm-cluster-token"
const PrivateKotsadmRegistrySecret = "kotsadm-private-registry"
const KotsadmConfigMap = "kotsadm-confg"

const ExcludeKey = "velero.io/exclude-from-backup"
const ExcludeValue = "true"

const BackupLabel = "kots.io/backup"
const BackupLabelValue = "velero"

const DisasterRecoveryLabel = "replicated.com/disaster-recovery"
const DisasterRecoveryLabelValueInfra = "infra"
const DisasterRecoveryLabelValueApp = "app"
const DisasterRecoveryChartLabel = "replicated.com/disaster-recovery-chart"
const DisasterRecoveryChartValue = "admin-console"

const TroubleshootKey = "troubleshoot.sh/kind"
const TroubleshootValue = "support-bundle"

const DefaultSupportBundleSpecKey = "default"
const ClusterSpecificSupportBundleSpecKey = "cluster-specific"
const VendorSpecificSupportBundleSpecKey = "vendor"

// TODO: additional labels in many places
func GetKotsadmLabels(additionalLabels ...map[string]string) map[string]string {
	labels := map[string]string{
		KotsadmKey:  KotsadmLabelValue,
		BackupLabel: BackupLabelValue,
	}

	if util.IsEmbeddedCluster() {
		labels[DisasterRecoveryLabel] = DisasterRecoveryLabelValueInfra
		labels[DisasterRecoveryChartLabel] = DisasterRecoveryChartValue
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
