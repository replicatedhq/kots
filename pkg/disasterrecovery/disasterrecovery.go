package disasterrecovery

import (
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/api/resid"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

const (
	LabelTransformerFileName = "backup-label-transformer.yaml"
)

type LabelTransformer struct {
	APIVersion string                     `json:"apiVersion"  yaml:"apiVersion"`
	Kind       string                     `json:"kind"  yaml:"kind"`
	Metadata   OverlySimpleMetadata       `json:"metadata"  yaml:"metadata"`
	Labels     map[string]string          `json:"labels,omitempty" yaml:"labels,omitempty"`
	FieldSpecs []kustomizetypes.FieldSpec `json:"fieldSpecs,omitempty" yaml:"fieldSpecs,omitempty"`
}

type OverlySimpleMetadata struct {
	Name string `yaml:"name"`
}

func GetLabelTransformerYAML(additionalLabels map[string]string) ([]byte, error) {
	labels := map[string]string{
		kotsadmtypes.BackupLabel: kotsadmtypes.BackupLabelValue,
	}

	for k, v := range additionalLabels {
		labels[k] = v
	}

	// References (selectors, matchLabels, and PVCs that are part of a StatefulSet are excluded)
	// CommonLabels list: https://github.com/kubernetes-sigs/kustomize/blob/6b81ae9a93c06c2ef500a407e27a52c68b01e3d8/api/konfig/builtinpluginconsts/commonlabels.go
	// LabelTransformer example: https://github.com/kubernetes-sigs/kustomize/blob/73cb5961223b80b233a9a0684d4133207881c103/plugin/builtin/labeltransformer/LabelTransformer_test.go

	labelTransformer := LabelTransformer{
		APIVersion: "builtin",
		Kind:       "LabelTransformer",
		Metadata: OverlySimpleMetadata{
			Name: "backup-label-transformer",
		},
		Labels: labels,
		FieldSpecs: []kustomizetypes.FieldSpec{
			{
				Path:               "metadata/labels",
				CreateIfNotPresent: true,
			},
			{
				Path: "spec/template/metadata/labels",
				Gvk: resid.Gvk{
					Version: "v1",
					Kind:    "ReplicationController",
				},
				CreateIfNotPresent: true,
			},
			{
				Path: "spec/template/metadata/labels",
				Gvk: resid.Gvk{
					Kind: "Deployment",
				},
				CreateIfNotPresent: true,
			},
			{
				Path: "spec/template/metadata/labels",
				Gvk: resid.Gvk{
					Kind: "ReplicaSet",
				},
				CreateIfNotPresent: true,
			},
			{
				Path: "spec/template/metadata/labels",
				Gvk: resid.Gvk{
					Kind: "DaemonSet",
				},
				CreateIfNotPresent: true,
			},
			{
				Path: "spec/template/metadata/labels",
				Gvk: resid.Gvk{
					Group: "apps",
					Kind:  "StatefulSet",
				},
				CreateIfNotPresent: true,
			},
			{
				Path: "spec/template/metadata/labels",
				Gvk: resid.Gvk{
					Group: "batch",
					Kind:  "Job",
				},
				CreateIfNotPresent: true,
			},
			{
				Path: "spec/jobTemplate/metadata/labels",
				Gvk: resid.Gvk{
					Group: "batch",
					Kind:  "CronJob",
				},
				CreateIfNotPresent: true,
			},
			{
				Path: "spec/jobTemplate/spec/template/metadata/labels",
				Gvk: resid.Gvk{
					Group: "batch",
					Kind:  "CronJob",
				},
				CreateIfNotPresent: true,
			},
		},
	}

	b, err := yaml.Marshal(labelTransformer)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal disaster recovery label transformer")
	}

	return b, nil
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
		drLabelTransformerYAML, err := GetLabelTransformerYAML(additionalLabels)
		if err != nil {
			return errors.Wrap(err, "failed to get disaster recovery label transformer yaml")
		}

		absFilename := filepath.Join(archiveDir, "overlays", "midstream", LabelTransformerFileName)

		if err := ioutil.WriteFile(absFilename, drLabelTransformerYAML, 0644); err != nil {
			return errors.Wrap(err, "failed to write disaster recovery label transformer yaml file")
		}

		k.Transformers = append(k.Transformers, LabelTransformerFileName)

		if err := k8sutil.WriteKustomizationToFile(*k, filepath.Join(archiveDir, "overlays", "midstream", "kustomization.yaml")); err != nil {
			return errors.Wrap(err, "failed to write kustomization file to midstream")
		}
	}

	return nil
}
