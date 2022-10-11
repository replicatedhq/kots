package kotsstore

import (
	"testing"
	"time"

	"github.com/replicatedhq/kots/pkg/supportbundle/types"
	"github.com/stretchr/testify/assert"
)

func Test_getSupportBundleStatus(t *testing.T) {
	tests := []struct {
		name       string
		updatedAt  time.Time
		status     types.SupportBundleStatus
		wantStatus types.SupportBundleStatus
	}{
		{
			name:       "bundle still running",
			updatedAt:  time.Now(),
			status:     types.BUNDLE_RUNNING,
			wantStatus: types.BUNDLE_RUNNING,
		},
		{
			name:       "bundle failed",
			updatedAt:  time.Now(),
			status:     types.BUNDLE_FAILED,
			wantStatus: types.BUNDLE_FAILED,
		},
		{
			name:       "bundle timed out",
			updatedAt:  time.Now().Add(-30 * time.Second),
			status:     types.BUNDLE_RUNNING,
			wantStatus: types.BUNDLE_FAILED,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := getSupportBundleStatus(test.status, &test.updatedAt)
			assert.Equal(t, test.wantStatus, got)
		})
	}
}
