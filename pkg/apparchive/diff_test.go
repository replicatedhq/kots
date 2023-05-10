package apparchive

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

func Test_diffAppFiles(t *testing.T) {
	tests := []struct {
		name         string
		archiveFiles map[string][]byte
		baseFiles    map[string][]byte
		want         *Diff
	}{
		{
			name: "no changes",
			archiveFiles: map[string][]byte{
				"my-file.txt": []byte(`this is my file
it has two lines`),
			},
			baseFiles: map[string][]byte{
				"my-file.txt": []byte(`this is my file
it has two lines`),
			},
			want: &Diff{
				FilesChanged: 0,
				LinesAdded:   0,
				LinesRemoved: 0,
			},
		},
		{
			name: "file added",
			archiveFiles: map[string][]byte{
				"my-file.txt": []byte(`this is my file
it has two lines`),
			},
			baseFiles: map[string][]byte{},
			want: &Diff{
				FilesChanged: 1,
				LinesAdded:   2,
				LinesRemoved: 0,
			},
		},
		{
			name:         "file removed",
			archiveFiles: map[string][]byte{},
			baseFiles: map[string][]byte{
				"my-file.txt": []byte(`this is my file
it has two lines`),
			},
			want: &Diff{
				FilesChanged: 1,
				LinesAdded:   0,
				LinesRemoved: 2,
			},
		},
		{
			name: "file changed",
			archiveFiles: map[string][]byte{
				"my-file.txt": []byte(`this is my file
it has two lines`),
			},
			baseFiles: map[string][]byte{
				"my-file.txt": []byte(`this is my file
it has
three lines`),
			},
			want: &Diff{
				FilesChanged: 1,
				LinesAdded:   1,
				LinesRemoved: 2,
			},
		},
		{
			name: "multiple files changed",
			archiveFiles: map[string][]byte{
				"my-file.txt": []byte(`this is my file
it has two lines`),
				"my-other-file.txt": []byte(`this is my other file
it has two lines`),
			},
			baseFiles: map[string][]byte{
				"my-file.txt": []byte(`this is my file
it has
three lines`),
				"my-other-file.txt": []byte(`this is my other file
it has also has
three lines`),
			},
			want: &Diff{
				FilesChanged: 2,
				LinesAdded:   2,
				LinesRemoved: 4,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			actual, err := diffAppFiles(test.archiveFiles, test.baseFiles)
			req.NoError(err)

			assert.Equal(t, *test.want, *actual)
		})
	}
}
