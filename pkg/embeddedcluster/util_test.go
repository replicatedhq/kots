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
		name string
		args args
		want *embeddedclusterv1beta1.ArtifactsLocation
	}{
		{
			name: "no artifacts",
			args: args{
				installation: kotsv1beta1.Installation{},
				appSlug:      "my-app",
			},
			want: nil,
		},
		{
			name: "has all artifacts",
			args: args{
				installation: kotsv1beta1.Installation{
					Spec: kotsv1beta1.InstallationSpec{
						EmbeddedClusterArtifacts: []string{
							"onprem.registry.com/my-app/embedded-cluster/charts.tar.gz:v1",
							"onprem.registry.com/my-app/embedded-cluster/images-amd64.tar:v1",
							"onprem.registry.com/my-app/embedded-cluster/embedded-cluster-amd64:v1",
							"onprem.registry.com/my-app/embedded-cluster/version-metadata.json:v1",
						},
					},
				},
				appSlug: "my-app",
			},
			want: &embeddedclusterv1beta1.ArtifactsLocation{
				Images:                  "onprem.registry.com/my-app/embedded-cluster/images-amd64.tar:v1",
				HelmCharts:              "onprem.registry.com/my-app/embedded-cluster/charts.tar.gz:v1",
				EmbeddedClusterBinary:   "onprem.registry.com/my-app/embedded-cluster/embedded-cluster-amd64:v1",
				EmbeddedClusterMetadata: "onprem.registry.com/my-app/embedded-cluster/version-metadata.json:v1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getArtifactsFromInstallation(tt.args.installation, tt.args.appSlug)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getArtifactsFromInstallation() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEmbeddedClusterArtifactOCIPath(t *testing.T) {
	type args struct {
		filename string
		opts     EmbeddedClusterArtifactOCIPathOptions
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "happy path for binary",
			args: args{
				filename: "embedded-cluster/embedded-cluster-amd64",
				opts: EmbeddedClusterArtifactOCIPathOptions{
					RegistryHost:      "registry.example.com",
					RegistryNamespace: "my-app",
					ChannelID:         "test-channel-id",
					UpdateCursor:      "1",
					VersionLabel:      "1.0.0",
				},
			},
			want: "registry.example.com/my-app/embedded-cluster/embedded-cluster-amd64:test-channel-id-1-1.0.0",
		},
		{
			name: "happy path for charts bundle",
			args: args{
				filename: "embedded-cluster/charts.tar.gz",
				opts: EmbeddedClusterArtifactOCIPathOptions{
					RegistryHost:      "registry.example.com",
					RegistryNamespace: "my-app",
					ChannelID:         "test-channel-id",
					UpdateCursor:      "1",
					VersionLabel:      "1.0.0",
				},
			},
			want: "registry.example.com/my-app/embedded-cluster/charts.tar.gz:test-channel-id-1-1.0.0",
		},
		{
			name: "happy path for image bundle",
			args: args{
				filename: "embedded-cluster/images-amd64.tar",
				opts: EmbeddedClusterArtifactOCIPathOptions{
					RegistryHost:      "registry.example.com",
					RegistryNamespace: "my-app",
					ChannelID:         "test-channel-id",
					UpdateCursor:      "1",
					VersionLabel:      "1.0.0",
				},
			},
			want: "registry.example.com/my-app/embedded-cluster/images-amd64.tar:test-channel-id-1-1.0.0",
		},
		{
			name: "happy path for version metadata",
			args: args{
				filename: "embedded-cluster/version-metadata.json",
				opts: EmbeddedClusterArtifactOCIPathOptions{
					RegistryHost:      "registry.example.com",
					RegistryNamespace: "my-app",
					ChannelID:         "test-channel-id",
					UpdateCursor:      "1",
					VersionLabel:      "1.0.0",
				},
			},
			want: "registry.example.com/my-app/embedded-cluster/version-metadata.json:test-channel-id-1-1.0.0",
		},
		{
			name: "file with name that needs to be sanitized",
			args: args{
				filename: "A file with spaces.tar.gz",
				opts: EmbeddedClusterArtifactOCIPathOptions{
					RegistryHost:      "registry.example.com",
					RegistryNamespace: "my-app",
					ChannelID:         "test-channel-id",
					UpdateCursor:      "1",
					VersionLabel:      "1.0.0",
				},
			},
			want: "registry.example.com/my-app/embedded-cluster/afilewithspaces.tar.gz:test-channel-id-1-1.0.0",
		},
		{
			name: "version label name that needs to be sanitized",
			args: args{
				filename: "test.txt",
				opts: EmbeddedClusterArtifactOCIPathOptions{
					RegistryHost:      "registry.example.com",
					RegistryNamespace: "my-app",
					ChannelID:         "test-channel-id",
					UpdateCursor:      "1",
					VersionLabel:      "A version with spaces",
				},
			},
			want: "registry.example.com/my-app/embedded-cluster/test.txt:test-channel-id-1-Aversionwithspaces",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EmbeddedClusterArtifactOCIPath(tt.args.filename, tt.args.opts); got != tt.want {
				t.Errorf("EmbeddedClusterArtifactOCIPath() = %v, want %v", got, tt.want)
			}
		})
	}
}
