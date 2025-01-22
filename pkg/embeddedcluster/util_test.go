package embeddedcluster

import (
	"reflect"
	"testing"

	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster/kinds/apis/v1beta1"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func Test_GetArtifactsFromInstallation(t *testing.T) {
	type args struct {
		installation kotsv1beta1.Installation
	}
	tests := []struct {
		name string
		args args
		want *embeddedclusterv1beta1.ArtifactsLocation
	}{
		{
			name: "no artifacts",
			args: args{
				installation: kotsv1beta1.Installation{},
			},
			want: nil,
		},
		{
			name: "has all artifacts",
			args: args{
				installation: kotsv1beta1.Installation{
					Spec: kotsv1beta1.InstallationSpec{
						EmbeddedClusterArtifacts: &kotsv1beta1.EmbeddedClusterArtifacts{
							Charts:      "onprem.registry.com/my-app/embedded-cluster/charts.tar.gz:v1",
							ImagesAmd64: "onprem.registry.com/my-app/embedded-cluster/images-amd64.tar:v1",
							BinaryAmd64: "onprem.registry.com/my-app/embedded-cluster/embedded-cluster-amd64:v1",
							Metadata:    "onprem.registry.com/my-app/embedded-cluster/version-metadata.json:v1",
							AdditionalArtifacts: map[string]string{
								"kots":     "onprem.registry.com/my-app/embedded-cluster/kots.tar.gz:v1",
								"operator": "onprem.registry.com/my-app/embedded-cluster/operator.tar.gz:v1",
								"manager":  "onprem.registry.com/my-app/embedded-cluster/manager.tar.gz:v1",
							},
						},
					},
				},
			},
			want: &embeddedclusterv1beta1.ArtifactsLocation{
				Images:                  "onprem.registry.com/my-app/embedded-cluster/images-amd64.tar:v1",
				HelmCharts:              "onprem.registry.com/my-app/embedded-cluster/charts.tar.gz:v1",
				EmbeddedClusterBinary:   "onprem.registry.com/my-app/embedded-cluster/embedded-cluster-amd64:v1",
				EmbeddedClusterMetadata: "onprem.registry.com/my-app/embedded-cluster/version-metadata.json:v1",
				AdditionalArtifacts: map[string]string{
					"kots":     "onprem.registry.com/my-app/embedded-cluster/kots.tar.gz:v1",
					"operator": "onprem.registry.com/my-app/embedded-cluster/operator.tar.gz:v1",
					"manager":  "onprem.registry.com/my-app/embedded-cluster/manager.tar.gz:v1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetArtifactsFromInstallation(tt.args.installation)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetArtifactsFromInstallation() = %v, want %v", got, tt.want)
			}
		})
	}
}
