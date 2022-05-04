package client

import (
	"reflect"
	"testing"
)

func TestGetGVKWithNameAndNs(t *testing.T) {
	type args struct {
		content string
		baseNS  string
	}
	tests := []struct {
		name    string
		args    args
		wantKey string
		wantGVK OverlySimpleGVKWithName
	}{
		{
			name: "native k8s object - ingress",
			args: args{
				content: `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: example-ingress
`,
				baseNS: "default",
			},
			wantKey: "Ingress-example-ingress-default",
			wantGVK: OverlySimpleGVKWithName{
				APIVersion: "networking.k8s.io/v1",
				Kind:       "Ingress",
				Metadata: OverlySimpleMetadata{
					Name: "example-ingress",
				},
			},
		},
		{
			name: "native k8s object - service",
			args: args{
				content: `apiVersion: v1
kind: Service
metadata:
  name: example-service	
`,
				baseNS: "example",
			},
			wantKey: "Service-example-service-example",
			wantGVK: OverlySimpleGVKWithName{
				APIVersion: "v1",
				Kind:       "Service",
				Metadata: OverlySimpleMetadata{
					Name: "example-service",
				},
			},
		},
		{
			name: "a crd",
			args: args{
				content: `apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: example-crd
`,
				baseNS: "example",
			},
			wantKey: "apiextensions.k8s.io/v1-CustomResourceDefinition-example-crd-example",
			wantGVK: OverlySimpleGVKWithName{
				APIVersion: "apiextensions.k8s.io/v1",
				Kind:       "CustomResourceDefinition",
				Metadata: OverlySimpleMetadata{
					Name: "example-crd",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key, gvk := GetGVKWithNameAndNs([]byte(tt.args.content), tt.args.baseNS)
			if key != tt.wantKey {
				t.Errorf("GetGVKWithNameAndNs() got key = %v, want %v", key, tt.wantKey)
			}
			if !reflect.DeepEqual(gvk, tt.wantGVK) {
				t.Errorf("GetGVKWithNameAndNs() got gvk = %v, want %v", gvk, tt.wantGVK)
			}
		})
	}
}
