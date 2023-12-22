package replicated

import (
	"os"
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
		{
			name: "oci image",
			args: args{
				image: "some-oci-image:some-tag",
			},
			want: []dockertypes.Layer{
				{
					Size:   55045922,
					Digest: "sha256:32fb02163b6bb519a30f909008e852354dae10bdfd6b34190dbdfe8f15403ea0",
				},
				{
					Size:   5166586,
					Digest: "sha256:167c7feebee855d117e192389484ea8367be1ba84e7ee35f4e5e5663195facbf",
				},
				{
					Size:   10876729,
					Digest: "sha256:d6dfff1f6f3ddd2194ea0775f199572e8b2d75c38713eef0444d6b1fd0ac7604",
				},
				{
					Size:   54585443,
					Digest: "sha256:e9cdcd4942ebc7445d8a70117a83ecbc77dcc5ffc72c4b6f8e24c0c76cfee15d",
				},
				{
					Size:   196811476,
					Digest: "sha256:ca3bce705f6c47c25b6e7896b4da514bf271c5827b1d19f51611c4a149dd713c",
				},
				{
					Size:   4205,
					Digest: "sha256:4f4cf292bc62eeea8a34b4160f3ef1f335b6b7b2bb9d28c605dc4002c8a24bc2",
				},
				{
					Size:   45579859,
					Digest: "sha256:054111693acb70b8d7d75dc8a846bb235bc0bfbf996b48867e23231975d0145b",
				},
				{
					Size:   2279423,
					Digest: "sha256:6ebdc2485ae0ea24eee4a04ebdf693e9a021aa45ab49b4c220e3db53c01d8f79",
				},
				{
					Size:   450,
					Digest: "sha256:63aedd7b9a0f5940555dca680f079ed3962526dd7fe03bfcce5fa5fe373d2424",
				},
				{
					Size:   14145524,
					Digest: "sha256:c1c68f450b6ee7ca31d1995a57f1063fc1061ed941661acbb143fec5792ad47e",
				},
				{
					Size:   141,
					Digest: "sha256:29f8913d8e982041f8366254ffa32dcfc82eca673710074d81756e5c3030aa86",
				},
				{
					Size:   960,
					Digest: "sha256:15164a7c6a2f31ac9ec373a1aa3423f6be8d892aab62b6c45c2e1bf23dc7adb3",
				},
				{
					Size:   282465,
					Digest: "sha256:f09029b4a9ddaa1a57f03a3dbe28ae4d661582074b1700c13663b321660af3c0",
				},
				{
					Size:   393,
					Digest: "sha256:e1300e83b67f39a8081c8234e00d586dfa179bafc693fa360a4c1291c76e4807",
				},
				{
					Size:   8144122,
					Digest: "sha256:4d7f456c67ea953671ade8da31d8e860b12ad6a2ecbc4c0692f6528eb46c0b7c",
				},
				{
					Size:   32,
					Digest: "sha256:4f4fb700ef54461cfa02571ae0db9a0dc1e0cdb5577484a6d75e68dc38e8acc1",
				},
				{
					Size:   670652217,
					Digest: "sha256:1cbe0f76d449b826a46fb582b4d4a29772103f830a261fe690bc0012612a83f9",
				},
				{
					Size:   101201807,
					Digest: "sha256:0d650e561c9d9f5f8b013053329c4ad767ff265cd99fbdefb18cecce2e5f13ce",
				},
			},
			wantErr: false,
		},
	}

	req := require.New(t)

	manifestsContent, err := os.ReadFile(path.Join("assets", "manifests.yaml"))
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
