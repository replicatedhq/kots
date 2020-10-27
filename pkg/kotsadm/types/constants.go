package types

import (
	"fmt"
)

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

func GetDisasterRecoveryLabelTransformerYAML() string {
	// reference (selectors/matchLabels excluded): https://github.com/kubernetes-sigs/kustomize/blob/6b81ae9a93c06c2ef500a407e27a52c68b01e3d8/api/konfig/builtinpluginconsts/commonlabels.go

	// TODO: handle volumeClaimTemplates labels?
	// - path: spec/volumeClaimTemplates[]/metadata/labels
	// create: true
	// group: apps
	// kind: StatefulSet

	return fmt.Sprintf(`apiVersion: builtin
kind: LabelTransformer
metadata:
  name: dr-label-transformer
labels:
  %s: %s
fieldSpecs:
- path: metadata/labels
  create: true
- path: spec/template/metadata/labels
  create: true
  version: v1
  kind: ReplicationController
- path: spec/template/metadata/labels
  create: true
  kind: Deployment
- path: spec/template/metadata/labels
  create: true
  kind: ReplicaSet
- path: spec/template/metadata/labels
  create: true
  kind: DaemonSet
- path: spec/template/metadata/labels
  create: true
  group: apps
  kind: StatefulSet

- path: spec/template/metadata/labels
  create: true
  group: batch
  kind: Job
- path: spec/jobTemplate/metadata/labels
  create: true
  group: batch
  kind: CronJob
- path: spec/jobTemplate/spec/template/metadata/labels
  create: true
  group: batch
  kind: CronJob
`, BackupLabel, BackupLabelValue)
}
