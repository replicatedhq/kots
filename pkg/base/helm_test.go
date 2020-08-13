package base

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
			name: "not a job",
			content: `apiVersion: batch/v1
kind: Deployment
metadata:
  name: pi
spec:
  template:
    spec:
      containers:
        - name: pi
          image: perl
          command: ["perl",  "-Mbignum=bpi", "-wle", "print bpi(2000)"]
      restartPolicy: Never`,
			expected: `apiVersion: batch/v1
kind: Deployment
metadata:
  name: pi
spec:
  template:
    spec:
      containers:
        - name: pi
          image: perl
          command: ["perl",  "-Mbignum=bpi", "-wle", "print bpi(2000)"]
      restartPolicy: Never`,
		},
		{
			name: "a job without a hook",
			content: `apiVersion: batch/v1
kind: Job
metadata:
  name: pi
spec:
  template:
    spec:
      containers:
        - name: pi
          image: perl
          command: ["perl",  "-Mbignum=bpi", "-wle", "print bpi(2000)"]
      restartPolicy: Never
  backoffLimit: 4`,
			expected: `apiVersion: batch/v1
kind: Job
metadata:
  name: pi
spec:
  template:
    spec:
      containers:
        - name: pi
          image: perl
          command: ["perl",  "-Mbignum=bpi", "-wle", "print bpi(2000)"]
      restartPolicy: Never
  backoffLimit: 4`,
		},
		{
			name: "a job with a helm hook",
			content: `apiVersion: batch/v1
kind: Job
metadata:
  name: pi
  annotations:
    "helm.sh/hook": "pre-install"
    "helm.sh/hook-weight": "-5"
    "helm.sh/hook-delete-policy": "hook-succeeded"
spec:
  template:
    spec:
      containers:
        - name: pi
          image: perl
          command: ["perl",  "-Mbignum=bpi", "-wle", "print bpi(2000)"]
      restartPolicy: Never
  backoffLimit: 4`,
			expected: `apiVersion: batch/v1
kind: Job
metadata:
  annotations:
    helm.sh/hook: pre-install
    helm.sh/hook-delete-policy: hook-succeeded
    helm.sh/hook-weight: "-5"
    kots.io/hook: pre-install
    kots.io/hook-delete-policy: hook-succeeded
    kots.io/hook-weight: "-5"
  creationTimestamp: null
  name: pi
spec:
  backoffLimit: 4
  template:
    metadata:
      creationTimestamp: null
    spec:
      containers:
      - command:
        - perl
        - -Mbignum=bpi
        - -wle
        - print bpi(2000)
        image: perl
        name: pi
        resources: {}
      restartPolicy: Never
status: {}
`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			scopetest := scopeagent.StartTest(t)
			defer scopetest.End()
			req := require.New(t)

			actual, err := transpileHelmHooksToKotsHooks([]byte(test.content))
			req.NoError(err)
			assert.Equal(t, test.expected, string(actual))
		})
	}
}
