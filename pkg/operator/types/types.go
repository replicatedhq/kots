package types

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	appstatetypes "github.com/replicatedhq/kots/pkg/appstate/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const (
	CreationPhaseAnnotation     = "kots.io/creation-phase"
	DeletionPhaseAnnotation     = "kots.io/deletion-phase"
	WaitForReadyAnnotation      = "kots.io/wait-for-ready"
	WaitForPropertiesAnnotation = "kots.io/wait-for-properties"
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

type Phases []Phase

type Phase struct {
	Name      string
	Resources Resources
}

type Resources []Resource

type Resource struct {
	Manifest     string
	GVR          schema.GroupVersionResource
	GVK          *schema.GroupVersionKind
	Unstructured *unstructured.Unstructured
	DecodeErrMsg string
}

type WaitForProperty struct {
	Path  string
	Value string
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

func (r Resource) ShouldWaitForReady() bool {
	if r.Unstructured != nil {
		annotations := r.Unstructured.GetAnnotations()
		if annotations == nil {
			return false
		}
		waitForReady, ok := annotations[WaitForReadyAnnotation]
		if !ok {
			return false
		}
		return waitForReady == "true"
	}
	return false
}

func (r Resource) ShouldWaitForProperties() bool {
	if r.Unstructured != nil {
		annotations := r.Unstructured.GetAnnotations()
		if annotations == nil {
			return false
		}
		_, ok := annotations[WaitForPropertiesAnnotation]
		return ok
	}
	return false
}

// GetWaitForProperties returns the key value pairs in the `kots.io/wait-for-properties` annotation
func (r Resource) GetWaitForProperties() []WaitForProperty {
	if r.Unstructured != nil {
		annotations := r.Unstructured.GetAnnotations()
		if annotations == nil {
			return nil
		}
		annotationValue, ok := annotations[WaitForPropertiesAnnotation]
		if !ok {
			return nil
		}

		waitForProperties := []WaitForProperty{}
		for _, property := range strings.Split(annotationValue, ",") {
			parts := strings.SplitN(property, "=", 2)
			if len(parts) != 2 {
				logger.Errorf("invalid wait for property %q", property)
				continue
			}
			waitForProperties = append(waitForProperties, WaitForProperty{
				Path:  parts[0],
				Value: parts[1],
			})
		}
		return waitForProperties
	}
	return nil
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

func (r Resources) GroupByCreationPhase() map[string]Resources {
	grouped := map[string]Resources{}

	for _, resource := range r {
		phase := "0" // default to 0
		if resource.Unstructured != nil {
			annotations := resource.Unstructured.GetAnnotations()
			if annotations != nil {
				if s, ok := annotations[CreationPhaseAnnotation]; ok {
					phase = s
				}
			}
		}

		parsed, err := strconv.ParseInt(phase, 10, 64)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to parse creation phase %q", phase))
			parsed = 0
		}

		key := fmt.Sprintf("%d", parsed)
		grouped[key] = append(grouped[key], resource)
	}

	return grouped
}

func (r Resources) GroupByDeletionPhase() map[string]Resources {
	grouped := map[string]Resources{}

	for _, resource := range r {
		phase := "0" // default to 0
		if resource.Unstructured != nil {
			annotations := resource.Unstructured.GetAnnotations()
			if annotations != nil {
				if s, ok := annotations[DeletionPhaseAnnotation]; ok {
					phase = s
				}
			}
		}

		parsed, err := strconv.ParseInt(phase, 10, 64)
		if err != nil {
			logger.Error(errors.Wrapf(err, "failed to parse deletion phase %q", phase))
			parsed = 0
		}

		key := fmt.Sprintf("%d", parsed)
		grouped[key] = append(grouped[key], resource)
	}

	return grouped
}
