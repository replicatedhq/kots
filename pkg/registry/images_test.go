package registry

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/registry/types"
)

func Test_shouldGarbageCollectImages(t *testing.T) {
	type args struct {
		isKurl           bool
		kurlRegistryHost string
		installParams    kotsutil.InstallationParams
		registrySettings types.RegistrySettings
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "return false if image garbage collection is disabled",
			args: args{
				installParams: kotsutil.InstallationParams{
					EnableImageDeletion: false,
				},
			},
			want: false,
		},
		{
			name: "return false if registry is read only",
			args: args{
				installParams: kotsutil.InstallationParams{
					EnableImageDeletion: true,
				},
				registrySettings: types.RegistrySettings{
					IsReadOnly: true,
				},
			},
			want: false,
		},
		{
			name: "return false if cluster is not kurl cluster",
			args: args{
				isKurl: false,
				installParams: kotsutil.InstallationParams{
					EnableImageDeletion: true,
				},
				registrySettings: types.RegistrySettings{
					IsReadOnly: false,
				},
			},
			want: false,
		},
		{
			name: "return false registry is not kurl registry/ when external registry is configured",
			args: args{
				isKurl:           true,
				kurlRegistryHost: "registry.kurl.sh",
				installParams: kotsutil.InstallationParams{
					EnableImageDeletion: true,
				},
				registrySettings: types.RegistrySettings{
					IsReadOnly: false,
					Hostname:   "registry.replicated.com",
				},
			},
			want: false,
		},
		{
			name: "return true when image garbage collection is enabled",
			args: args{
				isKurl:           true,
				kurlRegistryHost: "registry.kurl.sh",
				installParams: kotsutil.InstallationParams{
					EnableImageDeletion: true,
				},
				registrySettings: types.RegistrySettings{
					IsReadOnly: false,
					Hostname:   "registry.kurl.sh",
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldGarbageCollectImages(tt.args.isKurl, tt.args.kurlRegistryHost, tt.args.installParams, tt.args.registrySettings); got != tt.want {
				t.Errorf("shouldGarbageCollectImages() = %v, want %v", got, tt.want)
			}
		})
	}
}
