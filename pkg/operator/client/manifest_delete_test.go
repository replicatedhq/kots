package client

import (
	"reflect"
	"testing"
	"time"

	"github.com/replicatedhq/kots/pkg/operator/applier"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func Test_initResourceKindOrderMap(t *testing.T) {
	type args struct {
		kindOrder Plan
	}
	tests := []struct {
		name string
		args args
		want map[string][]Resource
	}{
		{
			name: "expect empty map",
			args: args{
				kindOrder: Plan{},
			},
			want: map[string][]Resource{},
		}, {
			name: "expect map with PostOrder entry",
			args: args{
				kindOrder: Plan{
					AfterAll: []string{"group1", "group2"},
				},
			},
			want: map[string][]Resource{
				"group1": {},
				"group2": {},
			},
		}, {
			name: "expect map with PreOrder entry",
			args: args{
				kindOrder: Plan{
					BeforeAll: []string{"group1", "group2"},
				},
			},
			want: map[string][]Resource{
				"group1": {},
				"group2": {},
			},
		}, {
			name: "expect map with PreOrder and PostOrder entry",
			args: args{
				kindOrder: Plan{
					BeforeAll: []string{"group1", "group2"},
					AfterAll:  []string{"group3", "group4"},
				},
			},
			want: map[string][]Resource{
				"group1": {},
				"group2": {},
				"group3": {},
				"group4": {},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := initResourcesMap(tt.args.kindOrder); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("initResourceKindOrderMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getOrderedKinds(t *testing.T) {
	type args struct {
		kindOrder    Plan
		defaultKinds KindSortOrder
	}
	tests := []struct {
		name string
		args args
		want KindSortOrder
	}{
		{
			name: "expect empty KindSortOrder",
			args: args{
				kindOrder:    Plan{},
				defaultKinds: KindSortOrder{},
			},
			want: KindSortOrder{},
		}, {
			name: "expect KindSortOrder with PreOrder",
			args: args{
				kindOrder: Plan{
					BeforeAll: []string{"group1", "group2"},
				},
				defaultKinds: KindSortOrder{},
			},
			want: KindSortOrder{
				"group1", "group2",
			},
		}, {
			name: "expect KindSortOrder with PostOrder",
			args: args{
				kindOrder: Plan{
					AfterAll: []string{"group1", "group2"},
				},
				defaultKinds: KindSortOrder{},
			},
			want: KindSortOrder{
				"group1", "group2",
			},
		}, {
			name: "expect KindSortOrder with PreOrder and PostOrder",
			args: args{
				kindOrder: Plan{
					BeforeAll: []string{"group1", "group2"},
					AfterAll:  []string{"group3", "group4"},
				},
				defaultKinds: KindSortOrder{},
			},
			want: KindSortOrder{
				"group1", "group2", "group3", "group4",
			},
		}, {
			name: "expect KindSortOrder with PreOrder and PostOrder and defaultKinds",
			args: args{
				kindOrder: Plan{
					BeforeAll: []string{"group1", "group2"},
					AfterAll:  []string{"group3", "group4"},
				},
				defaultKinds: KindSortOrder{
					"group5", "group6",
				},
			},
			want: KindSortOrder{
				"group1", "group2", "group5", "group6", "group3", "group4",
			},
		}, {
			name: "expect KindSortOrder with defaultKinds",
			args: args{
				kindOrder: Plan{},
				defaultKinds: KindSortOrder{
					"group5", "group6",
				},
			},
			want: KindSortOrder{
				"group5", "group6",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getOrderedKinds(tt.args.kindOrder, tt.args.defaultKinds); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getOrderedKinds() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_deleteManifestResource(t *testing.T) {
	gvk := schema.GroupVersionKind{
		Group:   "group",
		Version: "version",
		Kind:    "kind",
	}
	type args struct {
		resource          Resource
		targetNS          string
		waitFlag          bool
		kubernetesApplier applier.KubectlInterface
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "expect no error for resource with GVKN",
			args: args{
				resource: Resource{
					GVK:          &gvk,
					Unstructured: unstructuredPodWithLabels,
				},
				targetNS:          "default",
				kubernetesApplier: &kubectlApplierMock,
			},
		}, {
			name: "expect no error for resource without GVKN",
			args: args{
				resource: Resource{
					Unstructured: unstructuredPodWithLabels,
				},
				targetNS:          "default",
				kubernetesApplier: &kubectlApplierMock,
			},
		}, {
			name: "expect no error for resource without Unstructured",
			args: args{
				resource: Resource{
					GVK: &gvk,
				},
				targetNS:          "default",
				kubernetesApplier: &kubectlApplierMock,
			},
		}, {
			name: "expect no error for resource with Unstructured without namespace",
			args: args{
				resource: Resource{
					GVK: &gvk,
					Unstructured: &unstructured.Unstructured{
						Object: map[string]interface{}{},
					},
				},
				targetNS:          "default",
				kubernetesApplier: &kubectlApplierMock,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteResource(tt.args.resource, tt.args.targetNS, tt.args.waitFlag, tt.args.kubernetesApplier)
		})
	}
}

func Test_decodeToUnstructured(t *testing.T) {
	type args struct {
		manifest string
	}
	tests := []struct {
		name             string
		args             args
		wantUnstructured *unstructured.Unstructured
		wantGVK          *schema.GroupVersionKind
		wantErr          bool
	}{
		{
			name: "expect no error for valid pod manifest",
			args: args{
				manifest: podManifest,
			},
			wantUnstructured: unstructuredPod,
			wantGVK: &schema.GroupVersionKind{
				Group:   "",
				Version: "v1",
				Kind:    "Pod",
			},
			wantErr: false,
		}, {
			name: "expect error for invalid manifest",
			args: args{
				manifest: `test: false123`,
			},
			wantUnstructured: nil,
			wantGVK:          nil,
			wantErr:          true,
		}, {
			name: "expect no for rabbitmq CR manifest",
			args: args{
				manifest: rabbitmqCRManifest,
			},
			wantUnstructured: unstructuredRabbitMQCR,
			wantGVK: &schema.GroupVersionKind{
				Group:   "rabbitmq.com",
				Version: "v1beta1",
				Kind:    "RabbitmqCluster",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1, err := decodeToUnstructured(tt.args.manifest)
			if (err != nil) != tt.wantErr {
				t.Errorf("decodeToUnstructured() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.wantUnstructured) {
				t.Errorf("decodeToUnstructured() got = %v, want %v", got, tt.wantUnstructured)
			}
			if !reflect.DeepEqual(got1, tt.wantGVK) {
				t.Errorf("decodeToUnstructured() got1 = %v, want %v", got1, tt.wantGVK)
			}
		})
	}
}

func Test_decodeManifests(t *testing.T) {
	type args struct {
		manifests []string
	}
	tests := []struct {
		name string
		args args
		want []Resource
	}{
		{
			name: "expect no error for valid pod manifest",
			args: args{
				manifests: []string{podManifest},
			},
			want: []Resource{
				{
					GVK: &schema.GroupVersionKind{
						Group:   "",
						Version: "v1",
						Kind:    "Pod",
					},
					GVR:          schema.GroupVersionResource{},
					Unstructured: unstructuredPod,
				},
			},
		},
		{
			name: "expect no error for invalid pod manifest",
			args: args{
				manifests: []string{`test: false123`},
			},
			want: []Resource{
				{
					GVK:          nil,
					GVR:          schema.GroupVersionResource{},
					Unstructured: nil,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decodeManifests(tt.args.manifests)
			if len(got) != len(tt.want) {
				t.Errorf("decodeManifests() got = %v, want %v", len(got), len(tt.want))
			}
			for i := range got {
				if !reflect.DeepEqual(got[i].GVK, tt.want[i].GVK) {
					t.Errorf("decodeManifests() got = %v, want %v", got[i].GVK, tt.want[i].GVK)
				}
				if !reflect.DeepEqual(got[i].GVR, tt.want[i].GVR) {
					t.Errorf("decodeManifests() got = %v, want %v", got[i].GVR, tt.want[i].GVR)
				}
				if !reflect.DeepEqual(got[i].Unstructured, tt.want[i].Unstructured) {
					t.Errorf("decodeManifests() got = %v, want %v", got[i].Unstructured, tt.want[i].Unstructured)
				}
			}
		})
	}
}

func Test_buildCrdGVKMap(t *testing.T) {
	type args struct {
		resources []Resource
	}
	tests := []struct {
		name string
		args args
		want map[string]bool
	}{
		{
			name: "expect map with rabbitmq crd key for valid crd manifest",
			args: args{
				resources: []Resource{
					{
						GVK: &schema.GroupVersionKind{
							Group:   "",
							Version: "v1",
							Kind:    "Pod",
						},
						GVR:          schema.GroupVersionResource{},
						Unstructured: unstructuredPodWithLabels,
					},
					{
						GVK: &schema.GroupVersionKind{
							Group:   "apiextensions.k8s.io",
							Version: "v1",
							Kind:    "CustomResourceDefinition",
						},
						GVR:          schema.GroupVersionResource{},
						Unstructured: unstructuredRabbitMQCRD,
					},
				},
			},
			want: map[string]bool{
				"rabbitmq.com/RabbitmqCluster/v1beta1": true,
			},
		},
		{
			name: "expect empty map for invalid crd manifest",
			args: args{
				resources: []Resource{
					{
						GVK: &schema.GroupVersionKind{
							Group:   "apiextensions.k8s.io",
							Version: "v1",
							Kind:    "CustomResourceDefinition",
						},
						GVR: schema.GroupVersionResource{},
						Unstructured: &unstructured.Unstructured{
							Object: map[string]interface{}{
								"apiVersion": "apiextensions.k8s.io",
								"kind":       "CustomResourceDefinition",
								"metadata": map[string]interface{}{
									"name":      "rabbitmq-crd",
									"namespace": "default",
								},
								"spec": map[string]interface{}{
									"group": 123,
								},
							},
						},
					},
				},
			},
			want: map[string]bool{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildCrdGVKMap(tt.args.resources); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildCrdGVKMap() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildDeleteKindOrderedResources(t *testing.T) {
	podResource := Resource{
		GVK:          &podGVK,
		GVR:          schema.GroupVersionResource{},
		Unstructured: unstructuredPodWithLabels,
	}
	crdResource := Resource{
		GVK:          &crdGVK,
		GVR:          schema.GroupVersionResource{},
		Unstructured: unstructuredRabbitMQCRD,
	}
	crResource := Resource{
		GVK:          &crGVK,
		GVR:          schema.GroupVersionResource{},
		Unstructured: unstructuredRabbitMQCR,
	}
	nilGVKResource := Resource{
		GVK:          nil,
		GVR:          schema.GroupVersionResource{},
		Unstructured: unstructuredRabbitMQCRD,
	}
	type args struct {
		deleteKindOrder Plan
		resources       []Resource
		crdGVKMap       map[string]bool
	}
	tests := []struct {
		name             string
		args             args
		want             map[string][]Resource
		WantOrderedKinds KindSortOrder
	}{
		{
			name: "expect empty map with empty kind order for empty resources",
			args: args{
				deleteKindOrder: Plan{},
				resources:       []Resource{},
				crdGVKMap:       map[string]bool{},
			},
			want:             map[string][]Resource{},
			WantOrderedKinds: KindSortOrder{},
		}, {
			name: "expect pod map with pod kind default order for pod resources",
			args: args{
				deleteKindOrder: Plan{},
				resources:       []Resource{podResource},
				crdGVKMap:       map[string]bool{},
			},
			want: map[string][]Resource{
				"Pod": {podResource},
			},
			WantOrderedKinds: KindSortOrder{"Pod"},
		}, {
			name: "expect CRD map with CRD kind default order for pod resources",
			args: args{
				deleteKindOrder: Plan{
					BeforeAll: []string{"CustomResourceDefinition"},
				},
				resources: []Resource{crdResource},
				crdGVKMap: map[string]bool{},
			},
			want: map[string][]Resource{
				"CustomResourceDefinition": {crdResource},
			},
			WantOrderedKinds: KindSortOrder{"CustomResourceDefinition"},
		}, {
			name: "expect CRD map with empty string kind default order for empty GVK",
			args: args{
				deleteKindOrder: Plan{},
				resources:       []Resource{nilGVKResource},
				crdGVKMap:       map[string]bool{},
			},
			want: map[string][]Resource{
				"": {nilGVKResource},
			},
			WantOrderedKinds: KindSortOrder{""},
		}, {
			name: "expect CRD map with CRD kind default order for pod and crd/cr resources and crdKeyMap",
			args: args{
				deleteKindOrder: Plan{
					BeforeAll: []string{"CustomResourceDefinition"},
					AfterAll:  []string{"CustomResource"},
				},
				resources: []Resource{crResource, crdResource},
				crdGVKMap: map[string]bool{
					"rabbitmq.com/RabbitmqCluster/v1beta1": true,
				},
			},
			want: map[string][]Resource{
				"CustomResourceDefinition": {crdResource},
				"CustomResource":           {crResource},
			},
			WantOrderedKinds: KindSortOrder{"CustomResourceDefinition", "CustomResource"},
		}, {
			name: "an super edge case where the crd kind is in the crdKeyMap",
			args: args{
				deleteKindOrder: Plan{
					BeforeAll: []string{"RabbitmqCluster"},
				},
				resources: []Resource{crResource},
				crdGVKMap: map[string]bool{
					"rabbitmq.com/RabbitmqCluster/v1beta1": true,
				},
			},
			want: map[string][]Resource{
				"RabbitmqCluster": {crResource},
			},
			WantOrderedKinds: KindSortOrder{"RabbitmqCluster"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := sortResources(tt.args.deleteKindOrder, tt.args.resources, tt.args.crdGVKMap)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildDeleteKindOrderedResources() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.WantOrderedKinds) {
				t.Errorf("buildDeleteKindOrderedResources() got1 = %v, want %v", got1, tt.WantOrderedKinds)
			}
		})
	}
}

func Test_deleteManifestResources(t *testing.T) {
	type args struct {
		manifests         []string
		targetNS          string
		kubernetesApplier applier.KubectlInterface
		kindDeleteOrder   Plan
		waitFlag          bool
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "expect no error when deleting empty manifests",
			args: args{
				manifests:         []string{},
				targetNS:          "",
				kubernetesApplier: nil,
				kindDeleteOrder:   Plan{},
				waitFlag:          false,
			},
		}, {
			name: "expect no error when deleting manifests",
			args: args{
				manifests:         []string{podManifest},
				targetNS:          "test",
				kubernetesApplier: &kubectlApplierMock,
				kindDeleteOrder:   Plan{},
				waitFlag:          false,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deleteManifests(tt.args.manifests, tt.args.targetNS, tt.args.kubernetesApplier, tt.args.kindDeleteOrder, tt.args.waitFlag)
		})
	}
}

func Test_shouldWaitForResourceDeletion(t *testing.T) {
	type args struct {
		kind     string
		waitFlag bool
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "expect true when wait flag is true",
			args: args{
				kind:     "Pod",
				waitFlag: true,
			},
			want: true,
		}, {
			name: "expect false when wait flag is false",
			args: args{
				kind:     "Pod",
				waitFlag: false,
			},
			want: false,
		}, {
			name: "expect false when kind is PersistentVolumeClaim",
			args: args{
				kind:     "PersistentVolumeClaim",
				waitFlag: true,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldWaitForResourceDeletion(tt.args.kind, tt.args.waitFlag); got != tt.want {
				t.Errorf("shouldWaitForResourceDeletion() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildDeleteKindOrderedNamespaceResources(t *testing.T) {
	namespacedPodResource := Resource{
		GVR:          podGVR,
		GVK:          &podGVK,
		Unstructured: unstructuredPodWithLabels,
	}
	namespacedPodResourceMarkedForDeletion := Resource{
		GVR:          podGVR,
		GVK:          &podGVK,
		Unstructured: unstructuredPodMarkedDeletion,
	}

	type args struct {
		dyn                  dynamic.Interface
		gvrs                 []schema.GroupVersionResource
		appSlug              string
		namespace            string
		isRestore            bool
		restoreLabelSelector labels.Selector
		deleteKindOrder      Plan
	}
	tests := []struct {
		name                   string
		args                   args
		want                   map[string][]Resource
		wantdeleteOrderedKinds KindSortOrder
		wantErr                bool
	}{
		{
			name: "expect empty map and empty kind order with nil gvrs",
			args: args{
				gvrs: nil,
			},
			want:                   map[string][]Resource{},
			wantdeleteOrderedKinds: KindSortOrder{},
			wantErr:                false,
		}, {
			name: "expect empty map and empty kind order with empty gvrs",
			args: args{
				gvrs: []schema.GroupVersionResource{},
			},
			want:                   map[string][]Resource{},
			wantdeleteOrderedKinds: KindSortOrder{},
			wantErr:                false,
		}, {
			name: "expect empty map and empty kind order with gvr items empty",
			args: args{
				gvrs: []schema.GroupVersionResource{podGVR},
				dyn:  ReturnEmtyListDynamicClientMock(unstructuredPodWithLabels),
			},
			want:                   map[string][]Resource{},
			wantdeleteOrderedKinds: KindSortOrder{},
			wantErr:                false,
		}, {
			name: "expect empty map and empty kind order with gvr items empty",
			args: args{
				gvrs: []schema.GroupVersionResource{podGVR},
				dyn:  ReturnErrorDynamicClientListMock(unstructuredPodWithLabels),
			},
			want:                   map[string][]Resource{},
			wantdeleteOrderedKinds: KindSortOrder{},
			wantErr:                false,
		},
		{
			name: "expect pod map and pod kind order with valid gvr items",
			args: args{
				gvrs:      []schema.GroupVersionResource{podGVR},
				dyn:       ReturnDynamicClientMock(unstructuredPodWithLabels),
				isRestore: false,
				appSlug:   "test",
				namespace: "test",
			},
			want:                   map[string][]Resource{"Pod": {namespacedPodResource}},
			wantdeleteOrderedKinds: KindSortOrder{"Pod"},
			wantErr:                false,
		}, {
			name: "expect pod map and pod kind order with valid gvr items and restore true",
			args: args{
				gvrs:      []schema.GroupVersionResource{podGVR},
				dyn:       ReturnDynamicClientMock(unstructuredPodWithLabels),
				isRestore: true,
				appSlug:   "test",
				namespace: "test",
			},
			want:                   map[string][]Resource{"Pod": {namespacedPodResource}},
			wantdeleteOrderedKinds: KindSortOrder{"Pod"},
			wantErr:                false,
		}, {
			name: "expect pod map and pod kind order with a pod marked for deletion",
			args: args{
				gvrs:      []schema.GroupVersionResource{podGVR},
				dyn:       ReturnDynamicClientMock(unstructuredPodWithLabels, unstructuredPodMarkedDeletion),
				isRestore: true,
				appSlug:   "test",
				namespace: "test",
			},
			want:                   map[string][]Resource{"Pod": {namespacedPodResource, namespacedPodResourceMarkedForDeletion}},
			wantdeleteOrderedKinds: KindSortOrder{"Pod"},
			wantErr:                false,
		}, {
			name: "expect pod map and pod kind order with a pod excluded from backup",
			args: args{
				gvrs:      []schema.GroupVersionResource{podGVR},
				dyn:       ReturnDynamicClientMock(unstructuredPodWithLabels, unstructuredPodExcludeFromBackup),
				isRestore: true,
				appSlug:   "test",
				namespace: "test",
			},
			want:                   map[string][]Resource{"Pod": {namespacedPodResource}},
			wantdeleteOrderedKinds: KindSortOrder{"Pod"},
			wantErr:                false,
		},
		{
			name: "expect pod map and pod kind order with a pod restore label not match",
			args: args{
				gvrs:      []schema.GroupVersionResource{podGVR},
				dyn:       ReturnDynamicClientMock(unstructuredPodWithLabels, unstructuredPodWithRestoreLabelNotMatch),
				isRestore: true,
				appSlug:   "test",
				namespace: "test",
				restoreLabelSelector: labels.SelectorFromSet(map[string]string{
					"label/restore": "true",
				}),
			},
			want:                   map[string][]Resource{"Pod": {namespacedPodResource}},
			wantdeleteOrderedKinds: KindSortOrder{"Pod"},
			wantErr:                false,
		},
		{
			name: "expect pod map and pod kind order with a pod restore label match",
			args: args{
				gvrs:      []schema.GroupVersionResource{podGVR},
				dyn:       ReturnDynamicClientMock(unstructuredPodWithLabels, unstructuredPodWithRestoreLabel),
				isRestore: true,
				appSlug:   "test",
				namespace: "test",
				restoreLabelSelector: labels.SelectorFromSet(map[string]string{
					"label/restore": "true",
				}),
			},
			want: map[string][]Resource{"Pod": {
				namespacedPodResource,
				Resource{
					GVR:          podGVR,
					GVK:          &podGVK,
					Unstructured: unstructuredPodWithRestoreLabel,
				},
			}},
			wantdeleteOrderedKinds: KindSortOrder{"Pod"},
			wantErr:                false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := getResourcesInNamespace(tt.args.dyn, tt.args.gvrs, tt.args.appSlug, tt.args.namespace, tt.args.isRestore, tt.args.restoreLabelSelector, tt.args.deleteKindOrder)
			if !reflect.DeepEqual(got1, tt.wantdeleteOrderedKinds) {
				t.Errorf("buildDeleteKindOrderedNamespaceResources() got1 = %v, want %v", got1, tt.wantdeleteOrderedKinds)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildDeleteKindOrderedNamespaceResources() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_clearNamespacedResources(t *testing.T) {
	namespacedPodResource := Resource{
		GVR:          podGVR,
		GVK:          &podGVK,
		Unstructured: unstructuredPodWithLabels,
	}
	namespacedPodResourceMarkedForDeletion := Resource{
		GVR:          podGVR,
		GVK:          &podGVK,
		Unstructured: unstructuredPodMarkedDeletion,
	}
	type args struct {
		dyn              dynamic.Interface
		namespace        string
		resourcesMap     map[string][]Resource
		deleteKindOrders KindSortOrder
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "expect no error when no resources to clear",
			args:    args{},
			wantErr: false,
		}, {
			name: "expect no error when no resources to clear with kind order",
			args: args{
				deleteKindOrders: KindSortOrder{"Pod"},
			},
			wantErr: false,
		}, {
			name: "expect no error when pod resources to clear with kind order",
			args: args{
				resourcesMap:     map[string][]Resource{"Pod": {namespacedPodResource}},
				deleteKindOrders: KindSortOrder{"Pod"},
				dyn:              ReturnDynamicClientDeleteMock(unstructuredPodWithLabels),
				namespace:        "default",
			},
			wantErr: false,
		}, {
			name: "expect error when pod resources to clear with kind order",
			args: args{
				resourcesMap:     map[string][]Resource{"Pod": {namespacedPodResource}},
				deleteKindOrders: KindSortOrder{"Pod"},
				dyn:              ReturnErrDynamicClientDeleteMock(unstructuredPodWithLabels),
			},
			wantErr: true,
		}, {
			name: "expect no error when pod resources to clear with kind order and with pod marked for deletion",
			args: args{
				resourcesMap:     map[string][]Resource{"Pod": {namespacedPodResource, namespacedPodResourceMarkedForDeletion}},
				deleteKindOrders: KindSortOrder{"Pod"},
				dyn:              ReturnDynamicClientDeleteMock(unstructuredPodWithLabels, unstructuredPodMarkedDeletion),
				namespace:        "default",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := clearNamespacedResources(tt.args.dyn, tt.args.namespace, tt.args.resourcesMap, tt.args.deleteKindOrders); (err != nil) != tt.wantErr {
				t.Errorf("clearNamespacedResources() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_clearNamespaces(t *testing.T) {
	type args struct {
		appSlug              string
		namespacesToClear    []string
		isRestore            bool
		restoreLabelSelector labels.Selector
		kindDeleteOrder      Plan
		k8sDynamicClient     dynamic.Interface
		gvrs                 map[schema.GroupVersionResource]struct{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "expect no error when no namespaces to clear",
			args:    args{},
			wantErr: false,
		}, {
			name: "expect no error when no namespaces to clear with gvr in skip list",
			args: args{
				gvrs: map[schema.GroupVersionResource]struct{}{
					{
						Group:    "",
						Version:  "v1",
						Resource: "events",
					}: {},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := clearNamespaces(tt.args.appSlug, tt.args.namespacesToClear, tt.args.isRestore, tt.args.restoreLabelSelector, tt.args.kindDeleteOrder, tt.args.k8sDynamicClient, tt.args.gvrs); (err != nil) != tt.wantErr {
				t.Errorf("clearNamespaces() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_getLabelSelector(t *testing.T) {
	tests := []struct {
		name             string
		appLabelSelector metav1.LabelSelector
		want             string
	}{
		{
			name: "no requirements",
			appLabelSelector: metav1.LabelSelector{
				MatchLabels:      nil,
				MatchExpressions: nil,
			},
			want: "",
		},
		{
			name: "one requirement",
			appLabelSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kots.io/label": "abc",
				},
				MatchExpressions: nil,
			},
			want: "kots.io/label=abc",
		},
		{
			name: "two requirements",
			appLabelSelector: metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kots.io/label": "abc",
					"otherlabel":    "xyz",
				},
				MatchExpressions: nil,
			},
			want: "kots.io/label=abc,otherlabel=xyz",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, getLabelSelector(&tt.appLabelSelector))
		})
	}
}

func Test_clearNamespacesWithWait(t *testing.T) {
	type args struct {
		appSlug              string
		namespacesToClear    []string
		isRestore            bool
		restoreLabelSelector labels.Selector
		kindDeleteOrder      Plan
		k8sDynamicClient     dynamic.Interface
		deletionGVRs         []schema.GroupVersionResource
		waitTimeOut          int
		waitSleep            time.Duration
		waitExtra            time.Duration
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "expect no error when no namespaces to clear",
			args:    args{},
			wantErr: false,
		}, {
			name: "expect no error when resourcesToDeleteMap is empty",
			args: args{
				appSlug:           "not-test",
				namespacesToClear: []string{"default"},
				k8sDynamicClient:  ReturnDynamicClientMock(unstructuredPodWithLabels),
				deletionGVRs:      []schema.GroupVersionResource{podGVR},
			},
			wantErr: false,
		}, {
			name: "expect no error when resourcesToDeleteMap has a pod to delete",
			args: args{
				appSlug:           "test",
				namespacesToClear: []string{"default"},
				k8sDynamicClient:  NewSimpleDynamicClient(unstructuredPodWithLabels),
				deletionGVRs:      []schema.GroupVersionResource{podGVR},
				waitTimeOut:       1,
			},
			wantErr: false,
		}, {
			name: "expect no error when resourcesToDeleteMap has a pod to delete",
			args: args{
				appSlug:           "test",
				namespacesToClear: []string{"default"},
				k8sDynamicClient:  NewSimpleDynamicClient(unstructuredPodWithLabels),
				deletionGVRs:      []schema.GroupVersionResource{podGVR},
				waitTimeOut:       1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := clearNamespacesWithWait(tt.args.appSlug, tt.args.namespacesToClear, tt.args.isRestore, tt.args.restoreLabelSelector, tt.args.kindDeleteOrder, tt.args.k8sDynamicClient, tt.args.deletionGVRs, tt.args.waitTimeOut, tt.args.waitSleep, tt.args.waitExtra); (err != nil) != tt.wantErr {
				t.Errorf("clearNamespacesWithWait() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
