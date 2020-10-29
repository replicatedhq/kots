package types

const KotsadmKey = "kots.io/kotsadm"
const KotsadmLabelValue = "true"

const ClusterTokenSecret = "kotsadm-cluster-token"
const PrivateKotsadmRegistrySecret = "kotsadm-private-registry"
const KotsadmConfigMap = "kotsadm-confg"

const ExcludeLabel = "velero.io/exclude-from-backup"
const ExcludeLabelValue = "true"

const BackupLabel = "kots.io/backup"
const BackupLabelValue = "velero"

func GetKotsadmLabels(additionalLabels ...map[string]string) map[string]string {
	labels := map[string]string{
		KotsadmKey:  KotsadmLabelValue,
		BackupLabel: BackupLabelValue,
	}

	for _, l := range additionalLabels {
		for k, v := range l {
			labels[k] = v
		}
	}

	return labels
}
