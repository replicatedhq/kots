package appstate

import (
	"testing"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func Test_resourcePropertyMatchesValue(t *testing.T) {
	type args struct {
		r            *unstructured.Unstructured
		path         string
		desiredValue string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "property does not exist - no error",
			args: args{
				r:            &unstructured.Unstructured{},
				path:         ".status.availableReplicas",
				desiredValue: "1",
			},
			want: false,
		},
		{
			name: "property does not match - no error",
			args: args{
				r: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"availableReplicas": 0,
						},
					},
				},
				path:         ".status.availableReplicas",
				desiredValue: "1",
			},
			want: false,
		},
		{
			name: "property matches - no error",
			args: args{
				r: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"availableReplicas": 1,
						},
					},
				},
				path:         ".status.availableReplicas",
				desiredValue: "1",
			},
			want: true,
		},
		{
			name: "invalid path - error",
			args: args{
				r: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"availableReplicas": 1,
						},
					},
				},
				path:         "invalid][path",
				desiredValue: "1",
			},
			want:    false,
			wantErr: true,
		},
		{
			name: "empty key - error",
			args: args{
				r: &unstructured.Unstructured{
					Object: map[string]interface{}{
						"status": map[string]interface{}{
							"availableReplicas": 1,
						},
					},
				},
				path:         "",
				desiredValue: "1",
			},
			want:    false,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resourcePropertyMatchesValue(tt.args.r, tt.args.path, tt.args.desiredValue)
			if (err != nil) != tt.wantErr {
				t.Errorf("resourcePropertyMatchesValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("resourcePropertyMatchesValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
