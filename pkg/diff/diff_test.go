package diff

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/go-playground/assert.v1"
)

func Test_diffContent(t *testing.T) {
	tests := []struct {
		name                 string
		updatedContent       string
		baseContent          string
		expectedLinesAdded   int
		expectedLinesRemoved int
	}{
		{
			name: "identical",
			updatedContent: `env:
  - name: MINIO_ACCESS_KEY
    value: abc123
  - name: MINIO_SECRET_KEY
    value: abc234`,
			baseContent: `env:
  - name: MINIO_ACCESS_KEY
    value: abc123
  - name: MINIO_SECRET_KEY
    value: abc234`,
			expectedLinesAdded:   0,
			expectedLinesRemoved: 0,
		},
		{
			name: "single line edit",
			updatedContent: `env:
  - name: MINIO_ACCESS_KEY
    value: abc123
  - name: MINIO_SECRET_KEY
    value: abc234`,
			baseContent: `env:
  - name: MINIO_ACCESS_KEY
    value: abc123
  - name: MINIO_SECRET_KEY
    value: abc235`,
			expectedLinesAdded:   1,
			expectedLinesRemoved: 1,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			actualLinesAdded, actualLinesRemoved, err := diffContent(test.updatedContent, test.baseContent)
			req.NoError(err)

			assert.Equal(t, test.expectedLinesAdded, actualLinesAdded)
			assert.Equal(t, test.expectedLinesRemoved, actualLinesRemoved)
		})
	}
}
