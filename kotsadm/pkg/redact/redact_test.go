package redact

import (
	"testing"

	"github.com/stretchr/testify/require"
	_ "go.undefinedlabs.com/scopeagent/autoinstrument"
)

func TestCleanupSpec(t *testing.T) {
	tests := []struct {
		name    string
		spec    string
		want    string
		wantErr bool
	}{
		{
			name: "basic valid spec",
			spec: `
apiVersion: troubleshoot.replicated.com/v1beta1
kind: Redactor
metadata:
  name: my-redactor-name
spec:
  redactors:
  - name: replace password
    file: data/my-password-dump
    values:
    - abc123
  - name: all files
    regex:
    - (another)(?P<mask>.*)(here)
    multiLine:
    - selector: 'S3_ENDPOINT'
      redactor: '("value": ").*(")'
    yaml:
    - "abc.xyz.*"
`,
			want: `kind: Redactor
apiVersion: troubleshoot.replicated.com/v1beta1
metadata:
  name: my-redactor-name
spec:
  redactors:
  - name: replace password
    file: data/my-password-dump
    values:
    - abc123
  - name: all files
    regex:
    - (another)(?P<mask>.*)(here)
    multiLine:
    - selector: S3_ENDPOINT
      redactor: '("value": ").*(")'
    yaml:
    - abc.xyz.*
`,
		},
		{
			name: "no items spec",
			spec: `
apiVersion: troubleshoot.replicated.com/v1beta1
kind: Redactor
metadata:
  name: my-redactor-name
spec:
  redactors:
`,
			want: `kind: Redactor
apiVersion: troubleshoot.replicated.com/v1beta1
metadata:
  name: my-redactor-name
`,
		},
		{
			name:    "empty spec",
			wantErr: true,
			spec:    ``,
		},
		{
			name:    "missing group/version",
			wantErr: true,
			spec: `
kind: Redactor
metadata:
  name: my-redactor-name
spec:
  redactors:
  - name: replace password
    file: data/my-password-dump
    values:
    - abc123
`,
		},
		{
			name:    "missing kind",
			wantErr: true,
			spec: `
apiVersion: troubleshoot.replicated.com/v1beta1
metadata:
  name: my-redactor-name
spec:
  redactors:
  - name: replace password
    file: data/my-password-dump
    values:
    - abc123
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			got, err := CleanupSpec(tt.spec)
			if tt.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
				req.Equal(tt.want, got)
			}
		})
	}
}
