package kotsutil

import (
	"testing"
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
