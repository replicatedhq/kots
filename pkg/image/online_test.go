package image

import (
	"context"
	"testing"

	"github.com/pkg/errors"
	dockerregistrytypes "github.com/replicatedhq/kots/pkg/docker/registry/types"
	containerstypes "go.podman.io/image/v5/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_IsPrivateImage(t *testing.T) {
	tests := []struct {
		name      string
		image     string
		openerErr error // nil = public, non-nil = private (or retry on EOF)
		wantPriv  bool
		wantErr   bool
	}{
		{
			name:      "public image",
			image:     "redis:latest",
			openerErr: nil,
			wantPriv:  false,
		},
		{
			name:      "private image auth required",
			image:     "quay.io/replicated/myimage:latest",
			openerErr: errors.New("unauthorized: authentication required"),
			wantPriv:  true,
		},
		{
			name:      "manifest list no matching arch is not private",
			image:     "redis:7@sha256:e96c03a6dda7d0f28e2de632048a3d34bb1636d0858b65ef9a554441c70f6633",
			openerErr: errors.New("no image found in manifest list for architecture amd64"),
			wantPriv:  false,
		},
		{
			name:      "unreachable registry treated as private",
			image:     "testing.registry.com:5000/ns/image:1",
			openerErr: errors.New("dial tcp: connection refused"),
			wantPriv:  true,
		},
		{
			name:     "invalid image ref returns error",
			image:    "not a valid image ref !!",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			openerErr := tt.openerErr
			prober := func(_ context.Context, _ containerstypes.ImageReference, _ *containerstypes.SystemContext) error {
				return openerErr
			}
			got, err := isPrivateImageWithProber(tt.image, dockerregistrytypes.RegistryOptions{}, prober)
			if tt.wantErr {
				req.Error(err)
				return
			}
			req.NoError(err)
			assert.Equal(t, tt.wantPriv, got)
		})
	}
}
