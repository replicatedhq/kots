package kustomize

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mholt/archiver"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/stretchr/testify/require"
)

func TestRenderChartsArchive(t *testing.T) {
	tests := []struct {
		name                string
		files               map[string]string
		wantAllYAML         string
		wantKustomizedFiles map[string]string
		wantErr             bool
	}{
		{
			name: "handles charts that do not exist in base",
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
				"base/charts/guestbook/kustomization.yaml": `apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
- templates/serviceaccount.yaml
- templates/service.yaml
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
			wantAllYAML: `apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    kots.io/app-slug: my-app
  labels:
    helm.sh/chart: guestbook-0.1.0
  name: guestbook
  namespace: default
---
apiVersion: v1
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
			wantKustomizedFiles: map[string]string{
				"guestbook-serviceaccount.yaml": `apiVersion: v1
kind: ServiceAccount
metadata:
  annotations:
    kots.io/app-slug: my-app
  labels:
    helm.sh/chart: guestbook-0.1.0
  name: guestbook
  namespace: default`,
				"guestbook-service.yaml": `apiVersion: v1
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
			wantErr: false,
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

			gotArchive, gotKustomizedFiles, err := RenderChartsArchive(tmpDir, "this-cluster", "kustomize")
			if (err != nil) != tt.wantErr {
				t.Errorf("RenderChartsArchive() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotKustomizedFiles, tt.wantKustomizedFiles) {
				t.Errorf("RenderChartsArchive() kustomizedFiles \n\n%s", fmtJSONDiff(gotKustomizedFiles, tt.wantKustomizedFiles))
			}

			// validate archive contents
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

			gotAllYAML, err := ioutil.ReadFile(filepath.Join(extracted, "charts", "guestbook", "templates", "all.yaml"))
			require.Nil(t, err)

			if !reflect.DeepEqual(string(gotAllYAML), tt.wantAllYAML) {
				t.Errorf("RenderChartsArchive() gotAllYAML \n\n%s", fmtJSONDiff(string(gotAllYAML), tt.wantAllYAML))
			}
		})
	}
}

func fmtJSONDiff(got, want interface{}) string {
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
