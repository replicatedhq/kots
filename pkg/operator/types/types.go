package types

import (
	"sort"

	appstatetypes "github.com/replicatedhq/kots/pkg/appstate/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

type DeployAppArgs struct {
	AppID                string                `json:"app_id"`
	AppSlug              string                `json:"app_slug"`
	ClusterID            string                `json:"cluster_id"`
	Sequence             int64                 `json:"sequence"`
	KubectlVersion       string                `json:"kubectl_version"`
	KustomizeVersion     string                `json:"kustomize_version"`
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
	PreviousKotsKinds    *kotsutil.KotsKinds
	KotsKinds            *kotsutil.KotsKinds
}

type AppInformersArgs struct {
	AppID     string                               `json:"app_id"`
	Sequence  int64                                `json:"sequence"`
	Informers []appstatetypes.StatusInformerString `json:"informers"`
}

type Plan struct {
	BeforeAll []string
	Other     []string
	AfterAll  []string
}

func (p *Plan) Has(key string) bool {
	for _, k := range p.BeforeAll {
		if k == key {
			return true
		}
	}
	for _, k := range p.Other {
		if k == key {
			return true
		}
	}
	for _, k := range p.AfterAll {
		if k == key {
			return true
		}
	}
	return false
}

func (p *Plan) AddBeforeAll(key string) {
	if p.Has(key) {
		return
	}
	p.BeforeAll = append(p.BeforeAll, key)
}

func (p *Plan) AddOther(key string) {
	if p.Has(key) {
		return
	}
	p.Other = append(p.Other, key)
}

func (p *Plan) AddAfterAll(key string) {
	if p.Has(key) {
		return
	}
	p.AfterAll = append(p.AfterAll, key)
}

func (p *Plan) Flatten() []string {
	sort.Strings(p.Other) // sort alphabetically
	f := []string{}
	f = append(f, p.BeforeAll...)
	f = append(f, p.Other...)
	f = append(f, p.AfterAll...)
	return f
}

type Resources []Resource

type Resource struct {
	Manifest     string
	GVR          schema.GroupVersionResource
	GVK          *schema.GroupVersionKind
	Unstructured *unstructured.Unstructured
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

func (r Resources) SortWithPlan(plan Plan) Resources {
	resourcesMap := r.GroupByKind()

	for kind := range resourcesMap {
		if !plan.Has(kind) {
			plan.AddOther(kind)
		}
	}

	sortedResources := Resources{}
	for _, kind := range plan.Flatten() {
		sortedResources = append(sortedResources, resourcesMap[kind]...)
	}

	return sortedResources
}
