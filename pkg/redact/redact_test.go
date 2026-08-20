package redact

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/replicatedhq/kots/pkg/redact/types"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_getSlug(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "all alphanumeric",
			input: "aBC123",
			want:  "abc123",
		},
		{
			name:  "dashes",
			input: "abc-123",
			want:  "abc-123",
		},
		{
			name:  "spaces",
			input: "abc 123",
			want:  "abc-123",
		},
		{
			name:  "symbols",
			input: "abc%^123!@#",
			want:  "abc-123-at",
		},
		{
			name:  "kotsadm-redact",
			input: "kotsadm-redact",
			want:  "kotsadm-redact-metadata",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)
			req.Equal(tt.want, getSlug(tt.input))
		})
	}
}

func Test_getRedactSpec(t *testing.T) {
	tests := []struct {
		name      string
		secret    corev1.Secret
		want      string
		errstring string
	}{
		{
			name: "old spec only", // this test shows the code path when the redact specs haven't been updated
			secret: corev1.Secret{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Data: map[string][]byte{
					"kotsadm-redact": []byte(`
kind: Redactor
apiVersion: troubleshoot.sh/v1beta2
metadata:
  name: kotsadm-redact
spec:
  redactors: #there are comments here
  - name: replace password
    fileSelector:
      file: data/my-password-dump
    removals:
      values:
      - abc123
  - name: all files
    removals:
      regex:
      - redactor: (another)(?P<mask>.*)(here)
      - selector: S3_ENDPOINT
        redactor: '("value": ").*(")'
      yamlPath:
      - abc.xyz.*`),
				},
			},
			want: `apiVersion: troubleshoot.sh/v1beta2
kind: Redactor
metadata:
  name: kotsadm-redact
spec:
  redactors:
    - fileSelector:
        file: data/my-password-dump
      name: replace password
      removals:
        values:
          - abc123
    - fileSelector: {}
      name: all files
      removals:
        regex:
          - redactor: (another)(?P<mask>.*)(here)
          - redactor: '("value": ").*(")'
            selector: S3_ENDPOINT
        yamlPath:
          - abc.xyz.*
status: {}
`,
			errstring: "",
		},
		{
			name: "multiple new specs only",
			secret: corev1.Secret{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Data: map[string][]byte{
					"replace-password": []byte(`{"metadata":{"name":"replace password","slug":"replace-password","createdAt":"2020-06-15T14:26:10.721619-04:00","updatedAt":"2020-06-15T14:26:10.721619-04:00","enabled":true,"description":""},"redact":"kind: Redactor\napiVersion: troubleshoot.sh/v1beta2\nmetadata:\n  name: replace password\nspec:\n  redactors:\n  - name: replace password\n    fileSelector:\n      file: data/my-password-dump\n    removals:\n      values:\n      - abc123\n"}`),
					"all-files":        []byte(`{"metadata":{"name":"all files","slug":"all-files","createdAt":"2020-06-15T14:26:10.721733-04:00","updatedAt":"2020-06-15T14:26:10.721734-04:00","enabled":true,"description":""},"redact":"kind: Redactor\napiVersion: troubleshoot.sh/v1beta2\nmetadata:\n  name: all files\nspec:\n  redactors:\n  - name: all files\n    removals:\n      regex:\n      - redactor: (another)(?P\u003cmask\u003e.*)(here)\n      - selector: S3_ENDPOINT\n        redactor: '(\"value\": \").*(\")'\n      yamlPath:\n      - abc.xyz.*\n"}`),
				},
			},
			want: `apiVersion: troubleshoot.sh/v1beta2
kind: Redactor
metadata:
  name: kotsadm-redact
spec:
  redactors:
    - fileSelector: {}
      name: all files
      removals:
        regex:
          - redactor: (another)(?P<mask>.*)(here)
          - redactor: '("value": ").*(")'
            selector: S3_ENDPOINT
        yamlPath:
          - abc.xyz.*
    - fileSelector:
        file: data/my-password-dump
      name: replace password
      removals:
        values:
          - abc123
status: {}
`,
			errstring: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			got, errstring, err := getRedactSpec(&tt.secret)
			req.NoError(err)
			req.Equal(tt.want, got)
			req.Equal(tt.errstring, errstring)
		})
	}
}

func Test_MigrateRedactConfigMap(t *testing.T) {
	legacyData := map[string]string{
		"replace-password": `{"metadata":{"name":"replace password","slug":"replace-password","enabled":true,"description":""},"redact":"kind: Redactor\n"}`,
	}

	clientset := fake.NewSimpleClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      redactSecretName,
			Namespace: util.PodNamespace,
			Labels:    map[string]string{"kots.io/kotsadm": "true"},
		},
		Data: legacyData,
	})

	err := MigrateRedactConfigMap(clientset)
	require.NoError(t, err)

	secret, err := clientset.CoreV1().Secrets(util.PodNamespace).Get(context.TODO(), redactSecretName, metav1.GetOptions{})
	require.NoError(t, err)

	assert.Equal(t, legacyData["replace-password"], string(secret.Data["replace-password"]))

	_, err = clientset.CoreV1().ConfigMaps(util.PodNamespace).Get(context.TODO(), redactSecretName, metav1.GetOptions{})
	assert.True(t, kuberneteserrors.IsNotFound(err))
}

func Test_splitRedactors(t *testing.T) {
	tests := []struct {
		name string
		spec string
		want map[string]RedactorMetadata
	}{
		{
			name: "basic",
			spec: `kind: Redactor
apiVersion: troubleshoot.sh/v1beta2
metadata:
  name: kotsadm-redact
spec:
  redactors:
  - name: replace password
    fileSelector:
      file: data/my-password-dump
    removals:
      values:
      - abc123
  - name: all files
    removals:
      regex:
      - redactor: (another)(?P<mask>.*)(here)
      - selector: S3_ENDPOINT
        redactor: '("value": ").*(")'
      yamlPath:
      - abc.xyz.*
`,
			want: map[string]RedactorMetadata{
				"all-files": {
					Metadata: types.RedactorList{
						Name:    "all files",
						Slug:    "all-files",
						Enabled: true,
					},
					Redact: `apiVersion: troubleshoot.sh/v1beta2
kind: Redactor
metadata:
  name: all files
spec:
  redactors:
    - fileSelector: {}
      name: all files
      removals:
        regex:
          - redactor: (another)(?P<mask>.*)(here)
          - redactor: '("value": ").*(")'
            selector: S3_ENDPOINT
        yamlPath:
          - abc.xyz.*
status: {}
`,
				},
				"replace-password": {
					Metadata: types.RedactorList{
						Name:    "replace password",
						Slug:    "replace-password",
						Enabled: true,
					},
					Redact: `apiVersion: troubleshoot.sh/v1beta2
kind: Redactor
metadata:
  name: replace password
spec:
  redactors:
    - fileSelector:
        file: data/my-password-dump
      name: replace password
      removals:
        values:
          - abc123
status: {}
`,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			got, err := splitRedactors(tt.spec)
			req.NoError(err)
			req.Len(got, len(tt.want))
			for idx, val := range tt.want {
				gotIdx, ok := got[idx]
				fmt.Printf("\n\n%s\n\n", gotIdx)
				req.True(ok)
				gotIdxParsed := RedactorMetadata{}
				req.NoError(json.Unmarshal([]byte(gotIdx), &gotIdxParsed))
				req.Equal(val.Redact, gotIdxParsed.Redact)

				// set timestamps equal
				gotIdxParsed.Metadata.Updated = val.Metadata.Updated
				gotIdxParsed.Metadata.Created = val.Metadata.Created
				req.Equal(val.Metadata, gotIdxParsed.Metadata)
			}
		})
	}
}

func Test_setRedactYaml(t *testing.T) {
	previousTime, err := time.Parse(time.RFC3339, "2010-06-15T14:26:10.721619-04:00")
	if err != nil {
		panic(err)
	}

	testTime, err := time.Parse(time.RFC3339, "2020-06-15T14:26:10.721619-04:00")
	if err != nil {
		panic(err)
	}

	type args struct {
		slug        string
		description string
		enabled     bool
		newRedact   bool
		yamlBytes   []byte
		data        map[string]string
	}
	tests := []struct {
		name         string
		args         args
		newMap       map[string]string
		newMetadata  *RedactorMetadata
		expectedSlug string
	}{
		{
			name: "create new redact",
			args: args{
				slug:        "new-redact",
				description: "new description",
				enabled:     true,
				newRedact:   true,
				yamlBytes: []byte(`kind: Redactor
apiVersion: troubleshoot.sh/v1beta2
metadata:
  name: new redact`),
				data: nil,
			},
			newMap: map[string]string{
				"new-redact": `{"metadata":{"name":"new redact","slug":"new-redact","createdAt":"2020-06-15T14:26:10.721619-04:00","updatedAt":"2020-06-15T14:26:10.721619-04:00","enabled":true,"description":"new description"},"redact":"kind: Redactor\napiVersion: troubleshoot.sh/v1beta2\nmetadata:\n  name: new redact"}`,
			},
			newMetadata: &RedactorMetadata{
				Redact: `kind: Redactor
apiVersion: troubleshoot.sh/v1beta2
metadata:
  name: new redact`,
				Metadata: types.RedactorList{
					Name:        "new redact",
					Slug:        "new-redact",
					Enabled:     true,
					Description: "new description",
					Created:     testTime,
					Updated:     testTime,
				},
			},
		},
		{
			name: "update existing redact, leave other untouched",
			args: args{
				slug:        "new-redact",
				description: "a description",
				enabled:     true,
				yamlBytes: []byte(`kind: Redactor
apiVersion: troubleshoot.sh/v1beta2
metadata:
  name: update redact`),
				data: map[string]string{
					"new-redact":      `{"metadata":{"name":"new redact","slug":"new-redact","createdAt":"2010-06-15T14:26:10.721619-04:00","updatedAt":"2010-06-15T14:26:10.721619-04:00","enabled":true,"description":"new description"},"redact":"kind: Redactor\napiVersion: troubleshoot.sh/v1beta2\nmetadata:\n  name: new redact"}`,
					"leave-untouched": `other keys should not be modified`,
				},
			},
			newMap: map[string]string{
				"update-redact":   `{"metadata":{"name":"update redact","slug":"update-redact","createdAt":"2010-06-15T14:26:10.721619-04:00","updatedAt":"2020-06-15T14:26:10.721619-04:00","enabled":true,"description":"a description"},"redact":"kind: Redactor\napiVersion: troubleshoot.sh/v1beta2\nmetadata:\n  name: update redact"}`,
				"leave-untouched": `other keys should not be modified`,
			},
			newMetadata: &RedactorMetadata{
				Redact: `kind: Redactor
apiVersion: troubleshoot.sh/v1beta2
metadata:
  name: update redact`,
				Metadata: types.RedactorList{
					Name:        "update redact",
					Slug:        "update-redact",
					Enabled:     true,
					Description: "a description",
					Created:     previousTime,
					Updated:     testTime,
				},
			},
		},
		{
			name: "update existing redact without changing slug",
			args: args{
				slug:        "update-redact",
				description: "updated description",
				enabled:     true,
				yamlBytes: []byte(`kind: Redactor
apiVersion: troubleshoot.sh/v1beta2
metadata:
  name: update redact
spec: {}`),
				data: map[string]string{
					"update-redact":   `{"metadata":{"name":"update redact","slug":"update-redact","createdAt":"2010-06-15T14:26:10.721619-04:00","updatedAt":"2010-06-15T14:26:10.721619-04:00","enabled":true,"description":"a description"},"redact":"kind: Redactor\napiVersion: troubleshoot.sh/v1beta2\nmetadata:\n  name: update redact"}`,
					"leave-untouched": `other keys should not be modified`,
				},
			},
			newMap: map[string]string{
				"update-redact":   `{"metadata":{"name":"update redact","slug":"update-redact","createdAt":"2010-06-15T14:26:10.721619-04:00","updatedAt":"2020-06-15T14:26:10.721619-04:00","enabled":true,"description":"updated description"},"redact":"kind: Redactor\napiVersion: troubleshoot.sh/v1beta2\nmetadata:\n  name: update redact\nspec: {}"}`,
				"leave-untouched": `other keys should not be modified`,
			},
			newMetadata: &RedactorMetadata{
				Redact: `kind: Redactor
apiVersion: troubleshoot.sh/v1beta2
metadata:
  name: update redact
spec: {}`,
				Metadata: types.RedactorList{
					Name:        "update redact",
					Slug:        "update-redact",
					Enabled:     true,
					Description: "updated description",
					Created:     previousTime,
					Updated:     testTime,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			newMap, newMetadata, err := setRedactYaml(tt.args.slug, tt.args.description, tt.args.enabled, tt.args.newRedact, testTime, tt.args.yamlBytes, tt.args.data)
			req.NoError(err)

			req.Equal(tt.newMap, newMap)
			req.Equal(tt.newMetadata, newMetadata)
		})
	}
}

func Test_setRedactEnabled(t *testing.T) {
	previousTime, err := time.Parse(time.RFC3339, "2010-06-15T14:26:10.721619-04:00")
	if err != nil {
		panic(err)
	}

	testTime, err := time.Parse(time.RFC3339, "2020-06-15T14:26:10.721619-04:00")
	if err != nil {
		panic(err)
	}

	type args struct {
		slug    string
		enabled bool
		data    map[string]string
	}
	tests := []struct {
		name         string
		args         args
		newMap       map[string]string
		newMetadata  *RedactorMetadata
		expectedSlug string
	}{
		{
			name: "update existing redact",
			args: args{
				slug:    "update-redact",
				enabled: false,
				data: map[string]string{
					"update-redact":   `{"metadata":{"name":"update redact","slug":"update-redact","createdAt":"2010-06-15T14:26:10.721619-04:00","updatedAt":"2010-06-15T14:26:10.721619-04:00","enabled":true,"description":"a description"},"redact":"kind: Redactor\napiVersion: troubleshoot.sh/v1beta2\nmetadata:\n  name: update redact"}`,
					"leave-untouched": `other keys should not be modified`,
				},
			},
			newMap: map[string]string{
				"update-redact":   `{"metadata":{"name":"update redact","slug":"update-redact","createdAt":"2010-06-15T14:26:10.721619-04:00","updatedAt":"2020-06-15T14:26:10.721619-04:00","enabled":false,"description":"a description"},"redact":"kind: Redactor\napiVersion: troubleshoot.sh/v1beta2\nmetadata:\n  name: update redact"}`,
				"leave-untouched": `other keys should not be modified`,
			},
			newMetadata: &RedactorMetadata{
				Redact: `kind: Redactor
apiVersion: troubleshoot.sh/v1beta2
metadata:
  name: update redact`,
				Metadata: types.RedactorList{
					Name:        "update redact",
					Slug:        "update-redact",
					Enabled:     false,
					Description: "a description",
					Created:     previousTime,
					Updated:     testTime,
				},
			},
		},
		{
			name: "updated time changes even if enabled does not",
			args: args{
				slug:    "update-redact",
				enabled: true,
				data: map[string]string{
					"update-redact":   `{"metadata":{"name":"update redact","slug":"update-redact","createdAt":"2010-06-15T14:26:10.721619-04:00","updatedAt":"2010-06-15T14:26:10.721619-04:00","enabled":true,"description":"a description"},"redact":"kind: Redactor\napiVersion: troubleshoot.sh/v1beta2\nmetadata:\n  name: update redact"}`,
					"leave-untouched": `other keys should not be modified`,
				},
			},
			newMap: map[string]string{
				"update-redact":   `{"metadata":{"name":"update redact","slug":"update-redact","createdAt":"2010-06-15T14:26:10.721619-04:00","updatedAt":"2020-06-15T14:26:10.721619-04:00","enabled":true,"description":"a description"},"redact":"kind: Redactor\napiVersion: troubleshoot.sh/v1beta2\nmetadata:\n  name: update redact"}`,
				"leave-untouched": `other keys should not be modified`,
			},
			newMetadata: &RedactorMetadata{
				Redact: `kind: Redactor
apiVersion: troubleshoot.sh/v1beta2
metadata:
  name: update redact`,
				Metadata: types.RedactorList{
					Name:        "update redact",
					Slug:        "update-redact",
					Enabled:     true,
					Description: "a description",
					Created:     previousTime,
					Updated:     testTime,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			newMap, newMetadata, err := setRedactEnabled(tt.args.slug, tt.args.enabled, testTime, tt.args.data)
			req.NoError(err)

			req.Equal(tt.newMap, newMap)
			req.Equal(tt.newMetadata, newMetadata)
		})
	}
}
