package apparchive

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mholt/archiver/v3"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/require"
)

func TestRenderingCharts(t *testing.T) {
	tests := []struct {
		name                 string
		files                map[string]string
		wantRenderedFilesMap map[string][]byte
	}{
		{
			name: "app archive does not contain rendered charts",
			files: map[string]string{
				// this postgresql chart does not exist in base, function should not error
				"overlays/downstreams/this-cluster/charts/postgresql/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../../../midstream/charts/postgresql
kind: Kustomization
`,
				"overlays/downstreams/this-cluster/charts/guestbook/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../../../midstream/charts/guestbook
kind: Kustomization
`,
				"overlays/downstreams/this-cluster/charts/guestbook/crds/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../../../../midstream/charts/guestbook/crds
kind: Kustomization
`,
				"overlays/midstream/charts/postgresql/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../../../base/charts/postgresql
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
				"overlays/midstream/charts/guestbook/crds/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../../../../base/charts/guestbook/crds
commonAnnotations:
  kots.io/app-slug: my-app
kind: Kustomization
`,
				"base/charts/guestbook/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- templates/serviceaccount.yaml
- templates/service.yaml
`,
				"base/charts/guestbook/crds/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- test.yaml
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
				"base/charts/guestbook/crds/test.yaml": `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: tests.k8s.io
spec:
  group: tests.k8s.io
  names:
    kind: Test
    plural: tests
    shortNames:
    - te
    singular: test
  scope: Namespaced
  version: v1beta1
  versions:
  - name: v1beta1
    served: false
    storage: false
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
				"guestbook/Chart.yaml": []byte(`apiVersion: v2
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
`),
				"guestbook/crds/tests.k8s.io-customresourcedefinition.yaml": []byte(`apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  annotations:
    kots.io/app-slug: my-app
  name: tests.k8s.io
spec:
  group: tests.k8s.io
  names:
    kind: Test
    plural: tests
    shortNames:
    - te
    singular: test
  scope: Namespaced
  version: v1beta1
  versions:
  - name: v1beta1
    served: false
    storage: false
`),
				"guestbook/templates/guestbook-serviceaccount.yaml": []byte(`apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    kots.io/app-slug: my-app
  labels:
    helm.sh/chart: guestbook-0.1.0
  name: guestbook
  namespace: default`),
				"guestbook/templates/guestbook-service.yaml": []byte(`apiVersion: v1
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
`),
			},
		},
		{
			name: "app archive contains the rendered charts",
			files: map[string]string{
				"overlays/downstreams/this-cluster/charts/guestbook/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../../../midstream/charts/guestbook
kind: Kustomization
`,
				"overlays/downstreams/this-cluster/charts/guestbook/crds/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../../../../midstream/charts/guestbook/crds
kind: Kustomization
`,
				"overlays/midstream/charts/guestbook/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../../../base/charts/guestbook
commonAnnotations:
  kots.io/app-slug: my-app
kind: Kustomization
`,
				"overlays/midstream/charts/guestbook/crds/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../../../../base/charts/guestbook/crds
commonAnnotations:
  kots.io/app-slug: my-app
kind: Kustomization
`,
				"base/charts/guestbook/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- templates/serviceaccount.yaml
- templates/service.yaml
`,
				"base/charts/guestbook/crds/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- test.yaml
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
				"base/charts/guestbook/crds/test.yaml": `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: tests.k8s.io
spec:
  group: tests.k8s.io
  names:
    kind: Test
    plural: tests
    shortNames:
    - te
    singular: test
  scope: Namespaced
  version: v1beta1
  versions:
  - name: v1beta1
    served: false
    storage: false
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
				"rendered/this-cluster/charts/guestbook/crds/tests.k8s.io-customresourcedefinition.yaml": `apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  annotations:
    kots.io/app-slug: my-app
  name: tests.k8s.io
spec:
  group: tests.k8s.io
  names:
    kind: Test
    plural: tests
    shortNames:
    - te
    singular: test
  scope: Namespaced
  version: v1beta1
  versions:
  - name: v1beta1
    served: false
    storage: false
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
				"guestbook/Chart.yaml": []byte(`apiVersion: v2
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
`),
				"guestbook/crds/tests.k8s.io-customresourcedefinition.yaml": []byte(`apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  annotations:
    kots.io/app-slug: my-app
  name: tests.k8s.io
spec:
  group: tests.k8s.io
  names:
    kind: Test
    plural: tests
    shortNames:
    - te
    singular: test
  scope: Namespaced
  version: v1beta1
  versions:
  - name: v1beta1
    served: false
    storage: false
`),
				"guestbook/templates/guestbook-serviceaccount.yaml": []byte(`apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    kots.io/app-slug: my-app
  labels:
    helm.sh/chart: guestbook-0.1.0
  name: guestbook
  namespace: default`),
				"guestbook/templates/guestbook-service.yaml": []byte(`apiVersion: v1
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
`),
			},
		},
		{
			name: "app archive does not contain rendered charts and has an empty base kustomization",
			files: map[string]string{
				"overlays/downstreams/this-cluster/charts/guestbook/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../../../midstream/charts/guestbook
kind: Kustomization
`,
				"overlays/midstream/charts/guestbook/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
bases:
- ../../../../base/charts/guestbook
commonAnnotations:
  kots.io/app-slug: my-app
kind: Kustomization
`,
				"base/charts/guestbook/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
`,
			},
			wantRenderedFilesMap: map[string][]byte{},
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

			baseDir := filepath.Join(tmpDir, "base")
			overlaysDir := filepath.Join(tmpDir, "overlays")

			gotArchive, gotRenderedFilesMap, err := RenderChartsArchive(baseDir, overlaysDir, "this-cluster", "kustomize")
			req.NoError(err)

			if !reflect.DeepEqual(gotRenderedFilesMap, tt.wantRenderedFilesMap) {
				t.Errorf("RenderChartsArchive() renderedFilesMap \n\n%s", fmtJSONDiff(gotRenderedFilesMap, tt.wantRenderedFilesMap))
			}

			// validate archive files
			renderedTmp := t.TempDir()

			err = ioutil.WriteFile(filepath.Join(renderedTmp, "archive.tar.gz"), gotArchive, 0644)
			req.NoError(err)

			extracted := filepath.Join(renderedTmp, "extracted")
			err = os.MkdirAll(extracted, 0755)
			req.NoError(err)

			tarGz := archiver.TarGz{
				Tar: &archiver.Tar{
					ImplicitTopLevelFolder: false,
				},
			}
			err = tarGz.Unarchive(filepath.Join(renderedTmp, "archive.tar.gz"), extracted)
			req.NoError(err)

			for wantPath, wantContent := range tt.wantRenderedFilesMap {
				gotContent, err := ioutil.ReadFile(filepath.Join(extracted, "charts", wantPath))
				require.Nil(t, err)

				if !reflect.DeepEqual(gotContent, wantContent) {
					t.Errorf("RenderChartsArchive() %s \n\n%s", wantPath, fmtJSONDiff(string(gotContent), string(wantContent)))
				}
			}

			// this should return the same output
			gotArchive2, gotRenderedFilesMap2, err := GetRenderedV1Beta1ChartsArchive(tmpDir, "this-cluster", "kustomize")
			req.NoError(err)

			if !reflect.DeepEqual(gotRenderedFilesMap2, tt.wantRenderedFilesMap) {
				t.Errorf("GetRenderedV1Beta1ChartsArchive() renderedFilesMap \n\n%s", fmtJSONDiff(gotRenderedFilesMap2, tt.wantRenderedFilesMap))
			}

			// validate archive files
			os.RemoveAll(renderedTmp)
			renderedTmp = t.TempDir()

			err = ioutil.WriteFile(filepath.Join(renderedTmp, "archive.tar.gz"), gotArchive2, 0644)
			req.NoError(err)

			extracted = filepath.Join(renderedTmp, "extracted")
			err = os.MkdirAll(extracted, 0755)
			req.NoError(err)

			tarGz = archiver.TarGz{
				Tar: &archiver.Tar{
					ImplicitTopLevelFolder: false,
				},
			}
			err = tarGz.Unarchive(filepath.Join(renderedTmp, "archive.tar.gz"), extracted)
			req.NoError(err)

			for wantPath, wantContent := range tt.wantRenderedFilesMap {
				gotContent, err := ioutil.ReadFile(filepath.Join(extracted, "charts", wantPath))
				require.Nil(t, err)

				if !reflect.DeepEqual(gotContent, wantContent) {
					t.Errorf("GetRenderedV1Beta1ChartsArchive() %s \n\n%s", wantPath, fmtJSONDiff(string(gotContent), string(wantContent)))
				}
			}
		})
	}
}

func fmtJSONDiff(got, want interface{}) string {
	if _, ok := got.(map[string][]byte); ok {
		tmp := map[string]string{}
		for k, v := range got.(map[string][]byte) {
			tmp[k] = string(v)
		}
		got = tmp
	}

	if _, ok := want.(map[string][]byte); ok {
		tmp := map[string]string{}
		for k, v := range want.(map[string][]byte) {
			tmp[k] = string(v)
		}
		want = tmp
	}

	a, _ := json.MarshalIndent(got, "", "  ")
	b, _ := json.MarshalIndent(want, "", "  ")
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(string(a)),
		B:        difflib.SplitLines(string(b)),
		FromFile: "Got",
		ToFile:   "Want",
		Context:  1,
	}
	diffStr, _ := difflib.GetUnifiedDiffString(diff)
	return fmt.Sprintf("got:\n%s \n\nwant:\n%s \n\ndiff:\n%s", got, want, diffStr)
}
