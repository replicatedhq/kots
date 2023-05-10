package types

import (
	appstatetypes "github.com/replicatedhq/kots/pkg/appstate/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type DeployAppArgs struct {
	AppID                        string                `json:"app_id"`
	AppSlug                      string                `json:"app_slug"`
	ClusterID                    string                `json:"cluster_id"`
	Sequence                     int64                 `json:"sequence"`
	KubectlVersion               string                `json:"kubectl_version"`
	KustomizeVersion             string                `json:"kustomize_version"`
	AdditionalNamespaces         []string              `json:"additional_namespaces"`
	ImagePullSecrets             []string              `json:"image_pull_secrets"`
	PreviousManifests            string                `json:"previous_manifests"`
	Manifests                    string                `json:"manifests"`
	PreviousV1Beta1ChartsArchive []byte                `json:"previous_charts"`
	V1Beta1ChartsArchive         []byte                `json:"charts"`
	PreviousV1Beta2ChartsArchive []byte                `json:"previous_v1beta2_charts"`
	V1Beta2ChartsArchive         []byte                `json:"v1beta2_charts"`
	Wait                         bool                  `json:"wait"`
	Action                       string                `json:"action"`
	ClearNamespaces              []string              `json:"clear_namespaces"`
	ClearPVCs                    bool                  `json:"clear_pvcs"`
	AnnotateSlug                 bool                  `json:"annotate_slug"`
	IsRestore                    bool                  `json:"is_restore"`
	RestoreLabelSelector         *metav1.LabelSelector `json:"restore_label_selector"`
	PreviousKotsKinds            *kotsutil.KotsKinds
	KotsKinds                    *kotsutil.KotsKinds
}

type UndeployAppArgs struct {
	AppID                string                `json:"app_id"`
	AppSlug              string                `json:"app_slug"`
	ClusterID            string                `json:"cluster_id"`
	KubectlVersion       string                `json:"kubectl_version"`
	KustomizeVersion     string                `json:"kustomize_version"`
	AdditionalNamespaces []string              `json:"additional_namespaces"`
	Manifests            string                `json:"manifests"`
	V1Beta1ChartsArchive []byte                `json:"v1Beta1ChartsArchive"`
	V1Beta2ChartsArchive []byte                `json:"v1Beta2ChartsArchive"`
	Wait                 bool                  `json:"wait"`
	ClearNamespaces      []string              `json:"clear_namespaces"`
	ClearPVCs            bool                  `json:"clear_pvcs"`
	IsRestore            bool                  `json:"is_restore"`
	RestoreLabelSelector *metav1.LabelSelector `json:"restore_label_selector"`
	KotsKinds            *kotsutil.KotsKinds
}

type AppInformersArgs struct {
	AppID     string                               `json:"app_id"`
	Sequence  int64                                `json:"sequence"`
	Informers []appstatetypes.StatusInformerString `json:"informers"`
}

type Resources []Resource

type Resource struct {
	Manifest     string
	GVR          schema.GroupVersionResource
	GVK          *schema.GroupVersionKind
	Unstructured *unstructured.Unstructured
	DecodeErrMsg string
}

func (r Resource) GetGroup() string {
	if r.GVK != nil {
		return r.GVK.Group
	}
	return ""
}

func (r Resource) GetVersion() string {
	if r.GVK != nil {
		return r.GVK.Version
	}
	return ""
}

func (r Resource) GetKind() string {
	if r.GVK != nil {
		return r.GVK.Kind
	}
	return ""
}

func (r Resource) GetName() string {
	if r.Unstructured != nil {
		return r.Unstructured.GetName()
	}
	return ""
}

func (r Resource) GetNamespace() string {
	if r.Unstructured != nil {
		return r.Unstructured.GetNamespace()
	}
	return ""
}

func (r Resources) HasCRDs() bool {
	for _, resource := range r {
		if resource.GVK != nil && resource.GVK.Kind == "CustomResourceDefinition" && resource.GVK.Group == "apiextensions.k8s.io" {
			return true
		}
	}
	return false
}

func (r Resources) HasNamespaces() bool {
	for _, resource := range r {
		if resource.GVK != nil && resource.GVK.Kind == "Namespace" && resource.GVK.Group == "" && resource.GVK.Version == "v1" {
			return true
		}
	}
	return false
}

func (r Resources) GroupByKind() map[string]Resources {
	grouped := map[string]Resources{}

	for _, resource := range r {
		kind := ""
		if resource.GVK != nil {
			kind = resource.GVK.Kind
		}
		grouped[kind] = append(grouped[kind], resource)
	}

	return grouped
}
