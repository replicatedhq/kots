package base

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldBeIncludedInBase(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		content  []byte
		expected bool
	}{
		{
			name:     "NOTES.txt",
			path:     "NOTES.txt",
			content:  []byte("this is a notes.txt\nfrom helm"),
			expected: false,
		},
		{
			name: "some-deployment.yaml",
			path: "some-deployment.yaml",
			content: []byte(`apiVersion: apps/v1
kind: Deployment
metadata:
    name: nginx-deployment
spec:
    replicas: 3
    selector:
        matchLabels:
            app: nginx
    template:
        metadata:
            labels:
                app: nginx
        spec:
          containers:
            - name: nginx
          image: nginx:1.7.9
          ports:
            - containerPort: 80`),
			expected: true,
		},
		{
			name: "a-custom-resource.yaml",
			path: "a-custom-resource.yaml",
			content: []byte(`apiVersion: databases.schemahero.io/v1alpha2
kind: Database
metadata:
  name: rds-postgres
  namespace: default
connection:
  postgres:
    uri:
      valueFrom:
        secretKeyRef:
            key: uri
            name: rds-postgres`),
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			b := BaseFile{
				Path:    test.path,
				Content: test.content,
			}

			actual, err := b.ShouldBeIncludedInBase()
			req.NoError(err)

			assert.Equal(t, test.expected, actual)
		})
	}
}
