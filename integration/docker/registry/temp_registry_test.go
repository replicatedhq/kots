package replicated

import (
	"io/ioutil"
	"path"
	"reflect"
	"testing"
	"time"

	dockerregistry "github.com/replicatedhq/kots/pkg/docker/registry"
	dockertypes "github.com/replicatedhq/kots/pkg/docker/types"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestTempRegistry_GetImageLayers(t *testing.T) {
	type args struct {
		image string
	}
	tests := []struct {
		name    string
		args    args
		want    []dockertypes.Layer
		wantErr bool
	}{
		{
			name: "tag - single arch",
			args: args{
				image: "alpine:3.14-singlearch",
			},
			want: []dockertypes.Layer{
				{
					Digest: "sha256:8663204ce13b2961da55026a2034abb9e5afaaccf6a9cfb44ad71406dcd07c7b",
					Size:   2818370,
				},
			},
			wantErr: false,
		},
		{
			name: "tag - multi arch",
			args: args{
				image: "alpine:3.14-multiarch",
			},
			want: []dockertypes.Layer{
				{
					Digest: "sha256:8663204ce13b2961da55026a2034abb9e5afaaccf6a9cfb44ad71406dcd07c7b",
					Size:   2818370,
				},
				{
					Digest: "sha256:f9f2e4e531ad51ee917e8311e91a223a4893c1d754acb8246af87375ea60c6aa",
					Size:   2626056,
				},
				{
					Digest: "sha256:380010979fdd8a9a4b0bf397034a27ec6cabe61d36e9e6d460ea986f0ddaef38",
					Size:   2427969,
				},
				{
					Digest: "sha256:455c02918c4592a9beeeae47df541266f3ea53ed573feb767e5e8ab8dcee146e",
					Size:   2717389,
				},
				{
					Digest: "sha256:c11e5e1035714514f6e237dffd1836a4d03b48af64e55a8e08f9bd9e998e24a9",
					Size:   2821213,
				},
				{
					Digest: "sha256:ee5f6345565e7aeda814a5c097612cacb0a74186b1f01bf5199e1b812b5d3065",
					Size:   2814167,
				},
				{
					Digest: "sha256:6f6a6c77b1bd5dfb3e759efaa292f964f197ae4b96be74d80ef059f87317997a",
					Size:   2604075,
				},
			},
			wantErr: false,
		},
		{
			name: "digest - single arch",
			args: args{
				image: "alpine@sha256:54959ffc0f689960664029c5b2ee36ca06b029e506bc149eca18fdab8f7201a3",
			},
			want: []dockertypes.Layer{
				{
					Digest: "sha256:f9f2e4e531ad51ee917e8311e91a223a4893c1d754acb8246af87375ea60c6aa",
					Size:   2626056,
				},
			},
			wantErr: false,
		},
		{
			name: "digest - multi arch",
			args: args{
				image: "alpine@sha256:06b5d462c92fc39303e6363c65e074559f8d6b1363250027ed5053557e3398c5",
			},
			want: []dockertypes.Layer{
				{
					Digest: "sha256:8663204ce13b2961da55026a2034abb9e5afaaccf6a9cfb44ad71406dcd07c7b",
					Size:   2818370,
				},
				{
					Digest: "sha256:f9f2e4e531ad51ee917e8311e91a223a4893c1d754acb8246af87375ea60c6aa",
					Size:   2626056,
				},
				{
					Digest: "sha256:380010979fdd8a9a4b0bf397034a27ec6cabe61d36e9e6d460ea986f0ddaef38",
					Size:   2427969,
				},
				{
					Digest: "sha256:455c02918c4592a9beeeae47df541266f3ea53ed573feb767e5e8ab8dcee146e",
					Size:   2717389,
				},
				{
					Digest: "sha256:c11e5e1035714514f6e237dffd1836a4d03b48af64e55a8e08f9bd9e998e24a9",
					Size:   2821213,
				},
				{
					Digest: "sha256:ee5f6345565e7aeda814a5c097612cacb0a74186b1f01bf5199e1b812b5d3065",
					Size:   2814167,
				},
				{
					Digest: "sha256:6f6a6c77b1bd5dfb3e759efaa292f964f197ae4b96be74d80ef059f87317997a",
					Size:   2604075,
				},
			},
			wantErr: false,
		},
		{
			name: "digest and tag - single arch",
			args: args{
				image: "alpine:3.14@sha256:54959ffc0f689960664029c5b2ee36ca06b029e506bc149eca18fdab8f7201a3",
			},
			want: []dockertypes.Layer{
				{
					Digest: "sha256:f9f2e4e531ad51ee917e8311e91a223a4893c1d754acb8246af87375ea60c6aa",
					Size:   2626056,
				},
			},
			wantErr: false,
		},
	}

	req := require.New(t)

	manifestsContent, err := ioutil.ReadFile(path.Join("assets", "manifests.yaml"))
	req.NoError(err)

	var manifests map[string]string
	err = yaml.Unmarshal(manifestsContent, &manifests)
	req.NoError(err)

	serverOptions := MockServerOptions{
		Manifests: manifests,
	}
	server, err := StartMockServer(serverOptions)
	req.NoError(err)
	defer server.Close()

	r := &dockerregistry.TempRegistry{}
	r.OverridePort("3002")

	err = r.WaitForReady(time.Second * 30)
	req.NoError(err)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := r.GetImageLayers(tt.args.image)
			if (err != nil) != tt.wantErr {
				t.Errorf("TempRegistry.GetImageLayers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TempRegistry.GetImageLayers() = %v, want %v", got, tt.want)
			}
		})
	}
}
