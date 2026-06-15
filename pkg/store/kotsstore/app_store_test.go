package kotsstore

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseInstanceIDLineage(t *testing.T) {
	tests := []struct {
		name        string
		lineageJSON string
		want        []string
	}{
		{
			name:        "empty value means no lineage",
			lineageJSON: "",
			want:        nil,
		},
		{
			name:        "valid lineage",
			lineageJSON: `["instance-a","instance-b"]`,
			want:        []string{"instance-a", "instance-b"},
		},
		{
			// unparseable lineage must not error: failing here would abort restore
			// detection on every boot (the fingerprint never persists) while reports
			// silently pin to the app ID
			name:        "unparseable lineage self-heals to empty",
			lineageJSON: `{not json`,
			want:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseInstanceIDLineage("app-1", tt.lineageJSON))
		})
	}
}
