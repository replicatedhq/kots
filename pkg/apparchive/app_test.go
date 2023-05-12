package apparchive

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderingApp(t *testing.T) {
	tests := []struct {
		name                 string
		files                map[string]string
		wantRenderedFilesMap map[string][]byte
		wantAllContent       []byte
	}{
		{
			name: "app archive does not contain rendered manifests - should exclude charts",
			files: map[string]string{
				"overlays/downstreams/this-cluster/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../midstream
kind: Kustomization
`,
				"overlays/downstreams/this-cluster/charts/guestbook/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../../../midstream/charts/guestbook
kind: Kustomization
`,
				"overlays/midstream/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../base
commonAnnotations:
  kots.io/app-slug: my-app
kind: Kustomization
`,
				"overlays/midstream/charts/guestbook/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../../../base/charts/guestbook
commonAnnotations:
  kots.io/app-slug: my-app
kind: Kustomization
`,
				"base/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- configmap.yaml
- clusterrole.yaml
`,
				"base/configmap.yaml": `apiVersion: v1
kind: ConfigMap
metadata:
  name: myapp
data:
  key: value
`,
				"base/clusterrole.yaml": `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: myapp
rules:
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get
`,
				"base/charts/guestbook/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- templates/serviceaccount.yaml
- templates/service.yaml
`,
				"base/charts/guestbook/Chart.yaml": `apiVersion: v2
appVersion: 1.16.0
dependencies:
- alias: redis-alias
  name: redis
  repository: https://charts.bitnami.com/bitnami
  version: 17.0.6
description: A Helm chart for Kubernetes
name: guestbook
type: application
version: 0.1.0
`,
				"base/charts/guestbook/templates/serviceaccount.yaml": `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    helm.sh/chart: guestbook-0.1.0
  name: guestbook
  namespace: default
`,
				"base/charts/guestbook/templates/service.yaml": `apiVersion: v1
kind: Service
metadata:
  labels:
    helm.sh/chart: guestbook-0.1.0
  name: guestbook
  namespace: default
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: http
  selector:
    app.kubernetes.io/instance: guestbook
    app.kubernetes.io/name: guestbook
  type: ClusterIP
`,
			},
			wantRenderedFilesMap: map[string][]byte{
				"myapp-configmap.yaml": []byte(`apiVersion: v1
data:
  key: value
kind: ConfigMap
metadata:
  annotations:
    kots.io/app-slug: my-app
  name: myapp
`),
				"myapp-clusterrole.yaml": []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  annotations:
    kots.io/app-slug: my-app
  name: myapp
rules:
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get`),
			},
			wantAllContent: []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  annotations:
    kots.io/app-slug: my-app
  name: myapp
rules:
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get
---
apiVersion: v1
data:
  key: value
kind: ConfigMap
metadata:
  annotations:
    kots.io/app-slug: my-app
  name: myapp
`),
		},
		{
			name: "app archive contains the rendered manifests - should exclude charts",
			files: map[string]string{
				"overlays/downstreams/this-cluster/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../midstream
kind: Kustomization
`,
				"overlays/downstreams/this-cluster/charts/guestbook/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../../../midstream/charts/guestbook
kind: Kustomization
`,
				"overlays/midstream/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../base
commonAnnotations:
  kots.io/app-slug: my-app
kind: Kustomization
`,
				"overlays/midstream/charts/guestbook/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../../../base/charts/guestbook
commonAnnotations:
  kots.io/app-slug: my-app
kind: Kustomization
`,
				"base/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- configmap.yaml
- clusterrole.yaml
`,
				"base/configmap.yaml": `apiVersion: v1
kind: ConfigMap
metadata:
  name: myapp
data:
  key: value
`,
				"base/clusterrole.yaml": `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: myapp
rules:
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get
`,
				"base/charts/guestbook/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- templates/serviceaccount.yaml
- templates/service.yaml
`,
				"base/charts/guestbook/Chart.yaml": `apiVersion: v2
appVersion: 1.16.0
dependencies:
- alias: redis-alias
  name: redis
  repository: https://charts.bitnami.com/bitnami
  version: 17.0.6
description: A Helm chart for Kubernetes
name: guestbook
type: application
version: 0.1.0
`,
				"base/charts/guestbook/templates/serviceaccount.yaml": `apiVersion: v1
kind: ServiceAccount
metadata:
  labels:
    helm.sh/chart: guestbook-0.1.0
  name: guestbook
  namespace: default
`,
				"base/charts/guestbook/templates/service.yaml": `apiVersion: v1
kind: Service
metadata:
  labels:
    helm.sh/chart: guestbook-0.1.0
  name: guestbook
  namespace: default
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: http
  selector:
    app.kubernetes.io/instance: guestbook
    app.kubernetes.io/name: guestbook
  type: ClusterIP
`,
				"rendered/this-cluster/myapp-configmap.yaml": `apiVersion: v1
kind: ConfigMap
metadata:
  annotations:
    kots.io/app-slug: my-app
  name: myapp
data:
  key: value
`,
				"rendered/this-cluster/myapp-clusterrole.yaml": `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  annotations:
    kots.io/app-slug: my-app
  name: myapp
rules:
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get
`,
				"rendered/this-cluster/charts/guestbook/Chart.yaml": `apiVersion: v2
appVersion: 1.16.0
dependencies:
- alias: redis-alias
  name: redis
  repository: https://charts.bitnami.com/bitnami
  version: 17.0.6
description: A Helm chart for Kubernetes
name: guestbook
type: application
version: 0.1.0
`,
				"rendered/this-cluster/charts/guestbook/templates/guestbook-serviceaccount.yaml": `apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    kots.io/app-slug: my-app
  labels:
    helm.sh/chart: guestbook-0.1.0
  name: guestbook
  namespace: default`,
				"rendered/this-cluster/charts/guestbook/templates/guestbook-service.yaml": `apiVersion: v1
kind: Service
metadata:
  annotations:
    kots.io/app-slug: my-app
  labels:
    helm.sh/chart: guestbook-0.1.0
  name: guestbook
  namespace: default
spec:
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: http
  selector:
    app.kubernetes.io/instance: guestbook
    app.kubernetes.io/name: guestbook
  type: ClusterIP
`,
			},
			wantRenderedFilesMap: map[string][]byte{
				"myapp-configmap.yaml": []byte(`apiVersion: v1
kind: ConfigMap
metadata:
  annotations:
    kots.io/app-slug: my-app
  name: myapp
data:
  key: value
`),
				"myapp-clusterrole.yaml": []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  annotations:
    kots.io/app-slug: my-app
  name: myapp
rules:
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get
`),
			},
			wantAllContent: []byte(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  annotations:
    kots.io/app-slug: my-app
  name: myapp
rules:
- apiGroups:
  - ""
  resources:
  - namespaces
  verbs:
  - get

---
apiVersion: v1
kind: ConfigMap
metadata:
  annotations:
    kots.io/app-slug: my-app
  name: myapp
data:
  key: value
`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			req := require.New(t)

			for path, content := range tt.files {
				fullPath := filepath.Join(tmpDir, path)
				err := os.MkdirAll(filepath.Dir(fullPath), 0755)
				req.NoError(err)
				err = ioutil.WriteFile(fullPath, []byte(content), 0644)
				req.NoError(err)
			}

			gotAllContent, gotRenderedFilesMap, err := GetRenderedApp(tmpDir, "this-cluster", "kustomize")
			req.NoError(err)

			if !reflect.DeepEqual(gotAllContent, tt.wantAllContent) {
				t.Errorf("GetRenderedApp() allContent \n\n%s", fmtJSONDiff(string(gotAllContent), string(tt.wantAllContent)))
			}

			if !reflect.DeepEqual(gotRenderedFilesMap, tt.wantRenderedFilesMap) {
				t.Errorf("GetRenderedApp() renderedFilesMap \n\n%s", fmtJSONDiff(gotRenderedFilesMap, tt.wantRenderedFilesMap))
			}
		})
	}
}
