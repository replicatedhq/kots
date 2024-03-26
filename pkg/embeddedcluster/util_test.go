package embeddedcluster

import (
	"reflect"
	"testing"

	embeddedclusterv1beta1 "github.com/replicatedhq/embedded-cluster-kinds/apis/v1beta1"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func Test_getArtifactsFromInstallation(t *testing.T) {
	type args struct {
		installation kotsv1beta1.Installation
		appSlug      string
	}
	tests := []struct {
		name    string
		args    args
		want    *embeddedclusterv1beta1.ArtifactsLocation
		wantErr bool
	}{
		{
			name: "no artifacts",
			args: args{
				installation: kotsv1beta1.Installation{},
				appSlug:      "my-app",
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "has all artifacts",
			args: args{
				installation: kotsv1beta1.Installation{
					Spec: kotsv1beta1.InstallationSpec{
						AirgapArtifacts: []string{
							"onprem.registry.com/my-app/embedded-cluster/charts.tar.gz:v1",
							"onprem.registry.com/my-app/embedded-cluster/images-amd64.tar:v1",
							"onprem.registry.com/my-app/embedded-cluster/embedded-cluster-amd64:v1",
						},
					},
				},
				appSlug: "my-app",
			},
			want: &embeddedclusterv1beta1.ArtifactsLocation{
				Images:                "onprem.registry.com/my-app/embedded-cluster/images-amd64.tar:v1",
				HelmCharts:            "onprem.registry.com/my-app/embedded-cluster/charts.tar.gz:v1",
				EmbeddedClusterBinary: "onprem.registry.com/my-app/embedded-cluster/embedded-cluster-amd64:v1",
			},
			wantErr: false,
		},
		{
			name: "missing an artifact",
			args: args{
				installation: kotsv1beta1.Installation{
					Spec: kotsv1beta1.InstallationSpec{
						AirgapArtifacts: []string{
							"onprem.registry.com/my-app/embedded-cluster/charts.tar.gz:v1",
							"onprem.registry.com/my-app/embedded-cluster/embedded-cluster-amd64:v1",
						},
					},
				},
				appSlug: "my-app",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getArtifactsFromInstallation(tt.args.installation, tt.args.appSlug)
			if (err != nil) != tt.wantErr {
				t.Errorf("getArtifactsFromInstallation() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getArtifactsFromInstallation() = %v, want %v", got, tt.want)
			}
		})
	}
}
