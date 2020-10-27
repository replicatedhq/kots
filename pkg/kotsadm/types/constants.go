package types

import (
	"encoding/json"

	"github.com/pkg/errors"
)

const KotsadmKey = "kots.io/kotsadm"
const KotsadmLabelValue = "true"

const ClusterTokenSecret = "kotsadm-cluster-token"
const PrivateKotsadmRegistrySecret = "kotsadm-private-registry"
const KotsadmConfigMap = "kotsadm-confg"

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

func GetDisasterRecoveryPatches(additionalLabels ...map[string]string) ([]string, error) {
	labels := map[string]string{
		BackupLabel: BackupLabelValue,
	}
	for _, l := range additionalLabels {
		for k, v := range l {
			labels[k] = v
		}
	}

	// top level object labels patch
	type Metadata struct {
		Labels map[string]string `json:"labels"`
	}
	type TopLevelPatch struct {
		Metadata Metadata `json:"metadata"`
	}
	p1 := TopLevelPatch{
		Metadata: Metadata{
			Labels: labels,
		},
	}
	p1Str, err := json.Marshal(p1)
	if err != nil {
		return []string{}, errors.Wrap(err, "failed to marshal disaster recovery top level patch")
	}

	// template level labels patch
	type Template struct {
		Metadata Metadata `json:"metadata"`
	}
	type Spec struct {
		Template Template `json:"template"`
	}
	type TemplatePatch struct {
		Spec Spec `json:"spec"`
	}
	p2 := TemplatePatch{
		Spec: Spec{
			Template: Template{
				Metadata: Metadata{
					Labels: labels,
				},
			},
		},
	}
	p2Str, err := json.Marshal(p2)
	if err != nil {
		return []string{}, errors.Wrap(err, "failed to marshal disaster template level patch")
	}

	return []string{string(p1Str), string(p2Str)}, nil
}
