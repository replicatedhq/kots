package client

import (
	"reflect"
	"testing"

	"github.com/replicatedhq/kots/pkg/operator/types"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func Test_decodeManifests(t *testing.T) {
	type args struct {
		manifests []string
	}
	tests := []struct {
		name string
		args args
		want types.Resources
	}{
		{
			name: "expect no error for valid pod manifest",
			args: args{
				manifests: []string{podManifest},
			},
			want: types.Resources{
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
			want: types.Resources{
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
