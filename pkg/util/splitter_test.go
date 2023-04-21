package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSplitYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]string
	}{
		{
			name: "single doc",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec: ~`,
			expected: map[string]string{
				"test-deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec: ~`,
			},
		},
		{
			name: "two docs",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec: ~
---
apiVersion: v1
kind: Service
metadata:
  name: test
spec: ~`,

			expected: map[string]string{
				"test-deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec: ~`,
				"test-service.yaml": `apiVersion: v1
kind: Service
metadata:
  name: test
spec: ~`,
			},
		},
		{
			name: "with an empty doc",
			input: `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec: ~
---
apiVersion: v1
kind: Service
metadata:
  name: test
spec: ~`,

			expected: map[string]string{
				"test-deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec: ~`,
				"test-service.yaml": `apiVersion: v1
kind: Service
metadata:
  name: test
spec: ~`,
			},
		},
		{
			name: "same kind and name but different namespace",
			input: `---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
  namespace: ns1
spec: ~
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
  namespace: ns2
spec: ~
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
  namespace: ns3
spec: ~`,
			expected: map[string]string{
				"test-deployment.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
  namespace: ns1
spec: ~`,
				"test-deployment-1.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
  namespace: ns2
spec: ~`,
				"test-deployment-2.yaml": `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
  namespace: ns3
spec: ~`,
			},
		},
		{
			name: "same kind, name, and namespace but different apiVersion",
			input: `---
apiVersion: networking.k8s.io/v1beta2
kind: Ingress
metadata:
  name: test
  namespace: ns1
---
apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: test
  namespace: ns1
---
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test
  namespace: ns1`,
			expected: map[string]string{
				"test-ingress.yaml": `apiVersion: networking.k8s.io/v1beta2
kind: Ingress
metadata:
  name: test
  namespace: ns1`,
				"test-ingress-1.yaml": `apiVersion: networking.k8s.io/v1beta1
kind: Ingress
metadata:
  name: test
  namespace: ns1`,
				"test-ingress-2.yaml": `apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: test
  namespace: ns1`,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			actual, err := SplitYAML([]byte(test.input))
			req.NoError(err)

			// convert to strings to make it easier to view the failures
			actualConverted := map[string]string{}
			for f, v := range actual {
				actualConverted[f] = string(v)
			}
			assert.Equal(t, test.expected, actualConverted)
		})
	}
}
