package cli

import (
	"fmt"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"testing"
)

func Test_parseToleration(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *v1.Toleration
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name:  "Equal",
			input: "test.com/role:Equal:core:NoSchedule",
			want: &v1.Toleration{
				Key:      "test.com/role",
				Operator: v1.TolerationOpEqual,
				Value:    "core",
				Effect:   v1.TaintEffectNoSchedule,
			},
			wantErr: assert.NoError,
		},
		{
			name:  "Equal 30 seconds",
			input: "test.com/role:Equal:core:NoSchedule:30s",
			want: &v1.Toleration{
				Key:               "test.com/role",
				Operator:          v1.TolerationOpEqual,
				Value:             "core",
				Effect:            v1.TaintEffectNoSchedule,
				TolerationSeconds: util.IntPointer(30),
			},
			wantErr: assert.NoError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got, err := parseToleration(tt.input)
			if tt.wantErr == nil {
				req.NoError(err)
			} else {
				if !tt.wantErr(t, err, fmt.Sprintf("parseToleration(%v)", tt.input)) {
					return
				}
			}
			req.Equal(tt.want, got)
		})
	}
}
