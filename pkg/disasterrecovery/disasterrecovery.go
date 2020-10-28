package disasterrecovery

import (
	"filepath"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
)

const (
	LabelTransformerFileName = "dr-label-transformer.yaml"
)

func GetLabelTransformerYAML(additionalLabels map[string]string) string {
	labels := map[string]string{
		kotsadmtypes.BackupLabel: kotsadmtypes.BackupLabelValue,
	}

	for k, v := range additionalLabels {
		labels[k] = v
	}

	labelsStr := ""
	for k, v := range labels {
		labelsStr += fmt.Sprintf("  %s: %s\n", k, v) // TODO: fix this
	}
	labelsStr = strings.TrimSuffix(labelsStr, "\n")

	// reference (selectors/matchLabels excluded): https://github.com/kubernetes-sigs/kustomize/blob/6b81ae9a93c06c2ef500a407e27a52c68b01e3d8/api/konfig/builtinpluginconsts/commonlabels.go
	return fmt.Sprintf(`apiVersion: builtin
kind: LabelTransformer
metadata:
  name: dr-label-transformer
labels:
%s
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
`, labelsStr)
}

func EnsureLabelTransformer(archiveDir string, additionalLabels map[string]string) error {
	labelTransformerExists := false

	k, err := k8sutil.ReadKustomizationFromFile(filepath.Join(archiveDir, "overlays", "midstream", "kustomization.yaml"))
	if err != nil {
		return errors.Wrap(err, "failed to read kustomization file from midstream")
	}

	for _, transformer := range k.Transformers {
		if transformer == LabelTransformerFileName {
			labelTransformerExists = true
			break
		}
	}

	if !labelTransformerExists {
		drLabelTransformerYAML := GetLabelTransformerYAML(additionalLabels)

		absFilename := filepath.Join(archiveDir, "overlays", "midstream", LabelTransformerFileName)

		if err := ioutil.WriteFile(absFilename, []byte(drLabelTransformerYAML), 0644); err != nil {
			return errors.Wrap(err, "failed to write disaster recovery label transformer yaml file")
		}

		k.Transformers = append(k.Transformers, LabelTransformerFileName)

		if err := k8sutil.WriteKustomizationToFile(*k, filepath.Join(archiveDir, "overlays", "midstream", "kustomization.yaml")); err != nil {
			return errors.Wrap(err, "failed to write kustomization file to midstream")
		}
	}

	return nil
}
