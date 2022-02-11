package image

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_latestMinioImage(t *testing.T) {
	tests := []struct {
		name string
		a    string
		b    string
		want string
	}{
		{
			name: "simple case",
			a:    "minio/minio:RELEASE.2021-10-02T16-31-05Z",
			b:    "minio/minio:RELEASE.2022-01-08T03-11-54Z",
			want: "minio/minio:RELEASE.2022-01-08T03-11-54Z",
		},
		{
			name: "a is better case",
			a:    "minio/minio:RELEASE.2022-01-08T03-11-54Z",
			b:    "minio/minio:RELEASE.2021-10-02T16-31-05Z",
			want: "minio/minio:RELEASE.2022-01-08T03-11-54Z",
		},
		{
			name: "registry with port",
			a:    "registry.somebigbank.com:5000/minio/minio:RELEASE.2022-01-08T03-11-54Z",
			b:    "minio/minio:RELEASE.2021-10-02T16-31-05Z",
			want: "registry.somebigbank.com:5000/minio/minio:RELEASE.2022-01-08T03-11-54Z",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			got := latestMinioImage(tt.a, tt.b)
			req.Equal(tt.want, got)
		})
	}
}
