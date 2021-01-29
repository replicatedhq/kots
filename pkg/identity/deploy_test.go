package identity

import (
	"encoding/json"
	"reflect"
	"testing"

	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	kotsadmversion "github.com/replicatedhq/kots/pkg/kotsadm/version"
	corev1 "k8s.io/api/core/v1"
)

func Test_imageRewriteKotsadmRegistry(t *testing.T) {
	type args struct {
		namespace       string
		registryOptions kotsadmtypes.KotsadmOptions
		upstreamImage   string
		alwaysRewrite   bool
	}
	tests := []struct {
		name                 string
		args                 args
		wantImage            string
		wantImagePullSecrets []corev1.LocalObjectReference
		wantErr              bool
	}{
		{
			name: "dex",
			args: args{
				upstreamImage: "quay.io/dexidp/dex:v2.26.0",
			},
			wantImage:            "quay.io/dexidp/dex:v2.26.0",
			wantImagePullSecrets: nil,
		},
		{
			name: "dex no rewrite",
			args: args{
				registryOptions: kotsadmtypes.KotsadmOptions{
					OverrideNamespace: "testnamespace",
				},
				upstreamImage: "quay.io/dexidp/dex:v2.26.0",
			},
			wantImage:            "quay.io/dexidp/dex:v2.26.0",
			wantImagePullSecrets: nil,
		},
		{
			name: "kotsadm",
			args: args{
				upstreamImage: "kotsadm/kotsadm:v1.25.0",
			},
			wantImage:            "kotsadm/kotsadm:v1.25.0",
			wantImagePullSecrets: nil,
		},
		{
			name: "kotsadm always rewrite",
			args: args{
				registryOptions: kotsadmtypes.KotsadmOptions{
					OverrideNamespace: "testnamespace",
				},
				upstreamImage: "kotsadm/kotsadm:v1.25.0",
				alwaysRewrite: true,
			},
			wantImage:            "testnamespace/kotsadm:v0.0.0-unknown",
			wantImagePullSecrets: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fn := kotsadmversion.ImageRewriteKotsadmRegistry(tt.args.namespace, &tt.args.registryOptions)
			gotImage, gotImagePullSecrets, err := fn(tt.args.upstreamImage, tt.args.alwaysRewrite)
			if (err != nil) != tt.wantErr {
				t.Errorf("ImageRewriteFunc() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotImage != tt.wantImage {
				t.Errorf("ImageRewriteFunc() = %v, want %v", gotImage, tt.wantImage)
			}
			if !reflect.DeepEqual(gotImagePullSecrets, tt.wantImagePullSecrets) {
				bGot, _ := json.MarshalIndent(gotImagePullSecrets, "", "  ")
				bWant, _ := json.MarshalIndent(tt.wantImagePullSecrets, "", "  ")
				t.Errorf("ImageRewriteFunc() = %v, want %v", string(bGot), string(bWant))
			}
		})
	}
}
