package kotsutil

import (
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"
)

func Test_RemoveNilFieldsFromYAML(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name: "pod with no empty fields",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
  - image: nginx
`,
			want: `apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
  - image: nginx
`,
		},
		{
			name: "pod with empty labels and annotations",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
  annotations:
spec:
  containers:
  - image: nginx
`,
			want: `apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
  - image: nginx
`,
		},
		{
			name: "pod with empty initContainers",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  initContainers:
  containers:
  - image: nginx
`,
			want: `apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
  - image: nginx
`,
		},
		{
			name: "pod with empty imagePullSecrets",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  imagePullSecrets:
  containers:
  - image: nginx
`,
			want: `apiVersion: v1
kind: Pod
metadata:
  name: test
spec:
  containers:
  - image: nginx
`,
		},
		{
			name: "deployment with no empty fields",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      containers:
      - image: nginx
`,
			want: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      containers:
      - image: nginx
`,
		},
		{
			name: "deployment with empty pod spec initContainers",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      initContainers:
      containers:
      - image: nginx
`,
			want: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      containers:
      - image: nginx
`,
		},
		{
			name: "deployment with empty pod spec imagePullSecrets",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      imagePullSecrets:
      containers:
      - image: nginx
`,
			want: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      containers:
      - image: nginx
`,
		},
		{
			name: "deployment with empty pod spec volumes",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      volumes:
      containers:
      - image: nginx
`,
			want: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test
spec:
  template:
    spec:
      containers:
      - image: nginx
`,
		},
		{
			name: "statefulset with empty volumeClaimTemplates and volumes",
			input: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  volumeClaimTemplates:
  template:
    spec:
      volumes:
      containers:
      - name: mysql
`,
			want: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      containers:
      - name: mysql
`,
		},
		{
			name: "statefulset with null volumeClaimTemplates and volumes",
			input: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  volumeClaimTemplates: null
  template:
    spec:
      volumes: null
      containers:
      - name: mysql
`,
			want: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      containers:
      - name: mysql
`,
		},
		{
			name: "statefulset with empty volumeMounts",
			input: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      containers:
      - name: mysql
        volumeMounts:
`,
			want: `apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: mysql
spec:
  template:
    spec:
      containers:
      - name: mysql
`,
		},
	}

	for _, tt := range tests {
		got, err := RemoveNilFieldsFromYAML([]byte(tt.input))
		if (err != nil) != tt.wantErr {
			t.Errorf("%s - error = %v, wantErr %v", tt.name, err, tt.wantErr)
			return
		}
		if string(got) != tt.want {
			t.Errorf("%s - got = %v, want %v", tt.name, string(got), tt.want)
		}
	}
}

func TestFixUpYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		validate func(t *testing.T, got []byte, err error)
	}{
		{
			name:  "empty input",
			input: "",
			validate: func(t *testing.T, got []byte, err error) {
				if err != nil {
					t.Errorf("FixUpYAML() error = %v, want nil", err)
				}
				if len(got) != 0 {
					t.Errorf("FixUpYAML() = %v, want empty", string(got))
				}
			},
		},
		{
			name: "single document with long lines",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: test-pod-with-very-long-name-that-would-normally-wrap-at-80-characters
  labels:
    app: test-application-with-very-long-name-that-would-normally-wrap-at-80-characters
spec:
  containers:
  - name: nginx
    image: nginx:1.21
    env:
    - name: VERY_LONG_ENVIRONMENT_VARIABLE_NAME_THAT_WOULD_NORMALLY_WRAP_AT_80_CHARACTERS
      value: very-long-value-that-would-normally-wrap-at-80-characters
`,
			validate: func(t *testing.T, got []byte, err error) {
				if err != nil {
					t.Errorf("FixUpYAML() error = %v, want nil", err)
					return
				}

				// Check that the output is valid YAML
				var result map[string]interface{}
				if err := yaml.Unmarshal(got, &result); err != nil {
					t.Errorf("FixUpYAML() output is not valid YAML: %v", err)
					return
				}

				// Check that key fields are preserved
				if result["apiVersion"] != "v1" {
					t.Errorf("FixUpYAML() apiVersion = %v, want v1", result["apiVersion"])
				}
				if result["kind"] != "Pod" {
					t.Errorf("FixUpYAML() kind = %v, want Pod", result["kind"])
				}

				// Check that long lines are not wrapped (no line breaks in the middle of long values)
				gotStr := string(got)
				if strings.Contains(gotStr, "test-pod-with-very-long-name-that-would-normally-wrap-at-80-characters") &&
					strings.Contains(gotStr, "test-application-with-very-long-name-that-would-normally-wrap-at-80-characters") &&
					strings.Contains(gotStr, "VERY_LONG_ENVIRONMENT_VARIABLE_NAME_THAT_WOULD_NORMALLY_WRAP_AT_80_CHARACTERS") {
					// Good - long lines are preserved
				} else {
					t.Errorf("FixUpYAML() did not preserve long lines properly")
				}
			},
		},
		{
			name: "multiple documents",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: pod1
spec:
  containers:
  - name: nginx
    image: nginx:1.21
---
apiVersion: v1
kind: Service
metadata:
  name: service1
spec:
  selector:
    app: nginx
  ports:
  - port: 80
    targetPort: 80
`,
			validate: func(t *testing.T, got []byte, err error) {
				if err != nil {
					t.Errorf("FixUpYAML() error = %v, want nil", err)
					return
				}

				gotStr := string(got)
				// Check that documents are separated by ---
				if !strings.Contains(gotStr, "---") {
					t.Errorf("FixUpYAML() should separate multiple documents with ---")
				}

				// Check that both documents are present
				if !strings.Contains(gotStr, "kind: Pod") || !strings.Contains(gotStr, "kind: Service") {
					t.Errorf("FixUpYAML() should preserve both documents")
				}

				// Check that both document names are present
				if !strings.Contains(gotStr, "name: pod1") || !strings.Contains(gotStr, "name: service1") {
					t.Errorf("FixUpYAML() should preserve document names")
				}
			},
		},
		{
			name: "document with complex nested structure",
			input: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: complex-deployment
  labels:
    app: complex-app
    version: v1.0.0
spec:
  replicas: 3
  selector:
    matchLabels:
      app: complex-app
  template:
    metadata:
      labels:
        app: complex-app
        version: v1.0.0
    spec:
      containers:
      - name: web
        image: nginx:1.21
        ports:
        - containerPort: 80
        env:
        - name: NODE_ENV
          value: production
        - name: DATABASE_URL
          value: postgresql://user:password@localhost:5432/dbname
        resources:
          requests:
            memory: "64Mi"
            cpu: "250m"
          limits:
            memory: "128Mi"
            cpu: "500m"
        volumeMounts:
        - name: config
          mountPath: /etc/nginx
      volumes:
      - name: config
        configMap:
          name: nginx-config
`,
			validate: func(t *testing.T, got []byte, err error) {
				if err != nil {
					t.Errorf("FixUpYAML() error = %v, want nil", err)
					return
				}

				// Check that the output is valid YAML
				var result map[string]interface{}
				if err := yaml.Unmarshal(got, &result); err != nil {
					t.Errorf("FixUpYAML() output is not valid YAML: %v", err)
					return
				}

				// Check that key fields are preserved
				if result["apiVersion"] != "apps/v1" {
					t.Errorf("FixUpYAML() apiVersion = %v, want apps/v1", result["apiVersion"])
				}
				if result["kind"] != "Deployment" {
					t.Errorf("FixUpYAML() kind = %v, want Deployment", result["kind"])
				}

				// Check that nested structure is preserved
				gotStr := string(got)
				if !strings.Contains(gotStr, "complex-deployment") ||
					!strings.Contains(gotStr, "complex-app") ||
					!strings.Contains(gotStr, "nginx:1.21") ||
					!strings.Contains(gotStr, "NODE_ENV") ||
					!strings.Contains(gotStr, "DATABASE_URL") {
					t.Errorf("FixUpYAML() should preserve complex nested structure")
				}
			},
		},
		{
			name: "document with arrays and objects",
			input: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
data:
  config.yaml: |
    server:
      port: 8080
      hosts:
      - localhost
      - 127.0.0.1
      - example.com
    database:
      connection_string: postgresql://user:password@localhost:5432/dbname
      pool_size: 10
      timeout: 30s
  values.json: |
    {
      "feature_flags": {
        "enable_new_ui": true,
        "enable_beta_features": false,
        "max_connections": 1000
      },
      "endpoints": [
        "https://api.example.com/v1",
        "https://api.example.com/v2",
        "https://backup.example.com/v1"
      ]
    }
`,
			validate: func(t *testing.T, got []byte, err error) {
				if err != nil {
					t.Errorf("FixUpYAML() error = %v, want nil", err)
					return
				}

				// Check that the output is valid YAML
				var result map[string]interface{}
				if err := yaml.Unmarshal(got, &result); err != nil {
					t.Errorf("FixUpYAML() output is not valid YAML: %v", err)
					return
				}

				// Check that key fields are preserved
				if result["apiVersion"] != "v1" {
					t.Errorf("FixUpYAML() apiVersion = %v, want v1", result["apiVersion"])
				}
				if result["kind"] != "ConfigMap" {
					t.Errorf("FixUpYAML() kind = %v, want ConfigMap", result["kind"])
				}

				// Check that multiline strings are preserved
				gotStr := string(got)
				if !strings.Contains(gotStr, "config.yaml: |") ||
					!strings.Contains(gotStr, "values.json: |") ||
					!strings.Contains(gotStr, "server:") ||
					!strings.Contains(gotStr, "feature_flags") {
					t.Errorf("FixUpYAML() should preserve multiline strings and complex data")
				}
			},
		},
		{
			name: "document with empty values",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: empty-pod
spec:
  containers:
  - name: nginx
    image: nginx:1.21
    env: []
    ports: []
`,
			validate: func(t *testing.T, got []byte, err error) {
				if err != nil {
					t.Errorf("FixUpYAML() error = %v, want nil", err)
					return
				}

				// Check that the output is valid YAML
				var result map[string]interface{}
				if err := yaml.Unmarshal(got, &result); err != nil {
					t.Errorf("FixUpYAML() output is not valid YAML: %v", err)
					return
				}

				// Check that key fields are preserved
				if result["apiVersion"] != "v1" {
					t.Errorf("FixUpYAML() apiVersion = %v, want v1", result["apiVersion"])
				}
				if result["kind"] != "Pod" {
					t.Errorf("FixUpYAML() kind = %v, want Pod", result["kind"])
				}

				// Check that empty arrays are preserved
				gotStr := string(got)
				if !strings.Contains(gotStr, "env: []") ||
					!strings.Contains(gotStr, "ports: []") {
					t.Errorf("FixUpYAML() should preserve empty arrays")
				}
			},
		},
		{
			name: "invalid yaml",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: invalid-pod
spec:
  containers:
  - name: nginx
    image: nginx:1.21
    invalid: [unclosed array
`,
			validate: func(t *testing.T, got []byte, err error) {
				if err == nil {
					t.Errorf("FixUpYAML() error = nil, want error for invalid YAML")
				}
			},
		},
		{
			name:  "document with windows line endings",
			input: "apiVersion: v1\r\nkind: Pod\r\nmetadata:\r\n  name: windows-pod\r\nspec:\r\n  containers:\r\n  - name: nginx\r\n    image: nginx:1.21",
			validate: func(t *testing.T, got []byte, err error) {
				if err != nil {
					t.Errorf("FixUpYAML() error = %v, want nil", err)
					return
				}

				// Check that the output is valid YAML
				var result map[string]interface{}
				if err := yaml.Unmarshal(got, &result); err != nil {
					t.Errorf("FixUpYAML() output is not valid YAML: %v", err)
					return
				}

				// Check that key fields are preserved
				if result["apiVersion"] != "v1" {
					t.Errorf("FixUpYAML() apiVersion = %v, want v1", result["apiVersion"])
				}
				if result["kind"] != "Pod" {
					t.Errorf("FixUpYAML() kind = %v, want Pod", result["kind"])
				}

				// Check that windows line endings are converted to unix
				gotStr := string(got)
				if strings.Contains(gotStr, "\r\n") {
					t.Errorf("FixUpYAML() should convert windows line endings to unix")
				}
				if !strings.Contains(gotStr, "windows-pod") {
					t.Errorf("FixUpYAML() should preserve document content")
				}
			},
		},
		{
			name: "multiple documents with empty document",
			input: `apiVersion: v1
kind: Pod
metadata:
  name: pod1
spec:
  containers:
  - name: nginx
    image: nginx:1.21
---

---
apiVersion: v1
kind: Service
metadata:
  name: service1
spec:
  selector:
    app: nginx
  ports:
  - port: 80
    targetPort: 80
`,
			validate: func(t *testing.T, got []byte, err error) {
				if err != nil {
					t.Errorf("FixUpYAML() error = %v, want nil", err)
					return
				}

				gotStr := string(got)
				// Check that documents are separated by ---
				if !strings.Contains(gotStr, "---") {
					t.Errorf("FixUpYAML() should separate multiple documents with ---")
				}

				// Check that both documents are present (empty document should be filtered out)
				if !strings.Contains(gotStr, "kind: Pod") || !strings.Contains(gotStr, "kind: Service") {
					t.Errorf("FixUpYAML() should preserve both non-empty documents")
				}

				// Check that both document names are present
				if !strings.Contains(gotStr, "name: pod1") || !strings.Contains(gotStr, "name: service1") {
					t.Errorf("FixUpYAML() should preserve document names")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FixUpYAML([]byte(tt.input))
			tt.validate(t, got, err)
		})
	}
}
