package base

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldBeIncludedInBaseKustomization(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		content          []byte
		excludeKotsKinds bool
		expected         bool
	}{
		{
			name:             "NOTES.txt",
			path:             "NOTES.txt",
			content:          []byte("this is a notes.txt\nfrom helm"),
			excludeKotsKinds: false,
			expected:         false,
		},
		{
			name: "config.yaml excluded",
			path: "config.yaml",
			content: []byte(`apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: sentry-enterprise
spec:
  groups:
    - name: database
      title: Database
      description: Database Options
      items:
        - name: postgres_type
          type: select_one
          title: Postgres
          default: embedded_postgres
          items:
            - name: embedded_postgres
              title: Embedded Postgres
            - name: external_postgres
              title: External Postgres
        - name: embedded_postgres_password
          type: text
          read_only: true
          value: ""`),
			excludeKotsKinds: true,
			expected:         false,
		},
		{
			name: "config.yaml not excluded",
			path: "config.yaml",
			content: []byte(`apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: sentry-enterprise
spec:
  groups:
    - name: database
      title: Database
      description: Database Options
      items:
        - name: postgres_type
          type: select_one
          title: Postgres
          default: embedded_postgres
          items:
            - name: embedded_postgres
              title: Embedded Postgres
            - name: external_postgres
              title: External Postgres
        - name: embedded_postgres_password
          type: text
          read_only: true
          value: ""`),
			excludeKotsKinds: false,
			expected:         true,
		},
		{
			name:             "some-deployment.yaml",
			path:             "some-deployment.yaml",
			excludeKotsKinds: false,
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
			name:             "a-custom-resource.yaml",
			path:             "a-custom-resource.yaml",
			excludeKotsKinds: false,
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
			b := BaseFile{
				Path:    test.path,
				Content: test.content,
			}

			actual := b.ShouldBeIncludedInBaseKustomization(test.excludeKotsKinds)
			assert.Equal(t, test.expected, actual)
		})
	}
}
