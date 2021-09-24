package types

import (
	appstatetypes "github.com/replicatedhq/kots/pkg/appstate/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DeployAppArgs struct {
	AppID                string                `json:"app_id"`
	AppSlug              string                `json:"app_slug"`
	ClusterID            string                `json:"cluster_id"`
	Sequence             int64                 `json:"sequence"`
	KubectlVersion       string                `json:"kubectl_version"`
	AdditionalNamespaces []string              `json:"additional_namespaces"`
	ImagePullSecrets     []string              `json:"image_pull_secrets"`
	Namespace            string                `json:"namespace"`
	PreviousManifests    string                `json:"previous_manifests"`
	Manifests            string                `json:"manifests"`
	PreviousCharts       []byte                `json:"previous_charts"`
	Charts               []byte                `json:"charts"`
	Wait                 bool                  `json:"wait"`
	Action               string                `json:"action"`
	ClearNamespaces      []string              `json:"clear_namespaces"`
	ClearPVCs            bool                  `json:"clear_pvcs"`
	AnnotateSlug         bool                  `json:"annotate_slug"`
	IsRestore            bool                  `json:"is_restore"`
	RestoreLabelSelector *metav1.LabelSelector `json:"restore_label_selector"`
}

type AppInformersArgs struct {
	AppID     string                               `json:"app_id"`
	Sequence  int64                                `json:"sequence"`
	Informers []appstatetypes.StatusInformerString `json:"informers"`
}
