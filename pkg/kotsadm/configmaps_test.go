package kotsadm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func Test_updateConfigMap(t *testing.T) {
	tests := []struct {
		name     string
		existing map[string]string
		desired  map[string]string
		want     map[string]string
	}{
		{
			name: "operator-set keys survive and desired keys are overwritten",
			existing: map[string]string{
				"skip-preflights":            "false",
				"prune-enabled":              "true",
				"prune-support-bundle-count": "5",
			},
			desired: map[string]string{
				"skip-preflights": "true",
				"ensure-rbac":     "true",
			},
			want: map[string]string{
				"skip-preflights":            "true", // desired key overwritten
				"ensure-rbac":                "true", // new desired key added
				"prune-enabled":              "true", // operator key preserved
				"prune-support-bundle-count": "5",    // operator key preserved
			},
		},
		{
			name:     "existing keys not in desired are left untouched",
			existing: map[string]string{"existing-key": "keep"},
			desired: map[string]string{
				"script": "true",
			},
			want: map[string]string{
				"existing-key": "keep",
				"script":       "true",
			},
		},
		{
			name:     "nil existing data is initialized",
			existing: nil,
			desired: map[string]string{
				"skip-preflights": "true",
			},
			want: map[string]string{
				"skip-preflights": "true",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := &corev1.ConfigMap{Data: tt.existing}
			desired := &corev1.ConfigMap{Data: tt.desired}

			got := updateConfigMap(existing, desired)

			assert.Equal(t, tt.want, got.Data)
		})
	}
}
