package kotsutil

import (
	"testing"

	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_LoadTSKindsFromPath(t *testing.T) {
	tests := []struct {
		name              string
		archiveDir        string
		wantedPreflights  int
		wantNilPreflights bool
	}{
		{
			name:             "load single preflight from path",
			archiveDir:       "cases/preflights-in-helm-chart/rendered",
			wantedPreflights: 1,
		},
		{
			name:             "load multiple preflights from path",
			archiveDir:       "cases/multiple-preflights-in-helm-chart/rendered",
			wantedPreflights: 2,
		},
		{
			name:             "load zero preflights from path",
			archiveDir:       "cases/no-preflights-in-helm-chart/rendered",
			wantedPreflights: 0,
		},
		{
			name:              "don't fail if path doesn't exist",
			archiveDir:        "cases/not-a-real-dir/rendered",
			wantedPreflights:  0,
			wantNilPreflights: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := kotsutil.LoadTSKindsFromPath(tt.archiveDir)
			require.NoError(t, err)

			assert.Equal(t, tt.wantedPreflights, len(got.PreflightsV1Beta2))

			if tt.wantNilPreflights {
				assert.Nil(t, got.PreflightsV1Beta2)
			}
		})
	}
}
