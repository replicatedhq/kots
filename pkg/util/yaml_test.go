package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.undefinedlabs.com/scopeagent"
)

func Test_transpileHelmHooksToKotsHooks(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name: "deployment",
			content: `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: example
    component: nginx
    name: example-nginx
spec:
  replicas: 2
  selector:
    matchLabels:
      app: example
      component: nginx
    template:
      metadata:
        labels:
          app: example
          component: nginx
      spec:
        containers:
          - env:
            - name: ORIGINAL_CONFIG_STRING
              value: repl{{ ConfigOption "a_templated_text" }}
            - name: CHAINED_CONFIG_STRING
              value: repl{{ ConfigOption "a_templated_default_chain" }}
            - name: CHAINED_CONFIG_STRING_VALUE
              value: repl{{ ConfigOption "a_templated_readonly_value" }}
            - name: PASSWORD_VALUE
              value: repl{{ ConfigOption "a_password" }}
            - name: PASSWORD_VALUE_CHAIN
              value: repl{{ ConfigOption "a_templated_readonly_value_of_pass" }}
            - name: LICENSE_FIELD
              value: repl{{ LicenseFieldValue "laverya-kots-field" }}
            - name: IS_KURL
              value: '{{repl IsKurl }}'
            envFrom:
              - configMapRef:
                  name: example-config
            image: nginx
            name: nginx
            resources:
              limits:
                cpu: 50m
                memory: 64Mi
              requests:
                cpu: 5m
                memory: 10Mi`,
			expected: `apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: example
    component: nginx
    name: example-nginx
spec:
  replicas: 2
  selector:
    matchLabels:
      app: example
      component: nginx
    template:
      metadata:
        labels:
          app: example
          component: nginx
      spec:
        containers:
        - env:
          - name: ORIGINAL_CONFIG_STRING
            value: repl{{ ConfigOption "a_templated_text" }}
          - name: CHAINED_CONFIG_STRING
            value: repl{{ ConfigOption "a_templated_default_chain" }}
          - name: CHAINED_CONFIG_STRING_VALUE
            value: repl{{ ConfigOption "a_templated_readonly_value" }}
          - name: PASSWORD_VALUE
            value: repl{{ ConfigOption "a_password" }}
          - name: PASSWORD_VALUE_CHAIN
            value: repl{{ ConfigOption "a_templated_readonly_value_of_pass" }}
          - name: LICENSE_FIELD
            value: repl{{ LicenseFieldValue "laverya-kots-field" }}
          - name: IS_KURL
            value: '{{repl IsKurl }}'
          envFrom:
          - configMapRef:
              name: example-config
          image: nginx
          name: nginx
          resources:
            limits:
              cpu: 50m
              memory: 64Mi
            requests:
              cpu: 5m
              memory: 10Mi
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scopetest := scopeagent.StartTest(t)
			defer scopetest.End()
			req := require.New(t)

			actual, err := FixUpYAML([]byte(test.content))
			req.NoError(err)

			assert.Equal(t, test.expected, string(actual))
		})
	}
}
