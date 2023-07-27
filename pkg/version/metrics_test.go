package version

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/replicatedhq/kots/pkg/app/types"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"github.com/stretchr/testify/require"
)

func Test_GetGraphs(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	req := require.New(t)

	tests := []struct {
		name        string
		app         *types.App
		sequence    int64
		files       map[string]string
		mockStoreFn func(app *types.App, sequence int64, archiveDir string, files map[string]string) *mock_store.MockStore
		want        []kotsv1beta1.MetricGraph
		wantErr     bool
	}{
		{
			name: "no graphs - return default graphs",
			app: &types.App{
				ID:       "app-id",
				Slug:     "app-slug",
				IsAirgap: false,
			},
			sequence: 1,
			files: map[string]string{
				"upstream/app.yaml": `
apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: my-application
spec:
  title: My Application`,
			},
			mockStoreFn: func(app *types.App, sequence int64, archiveDir string, files map[string]string) *mock_store.MockStore {
				mockStore := mock_store.NewMockStore(ctrl)
				mockStore.EXPECT().GetAppVersionArchive(app.ID, sequence, gomock.Any()).Times(1).DoAndReturn(func(id string, seq int64, archDir string) error {
					err := setupDirectoriesAndFiles(archDir, files)
					req.NoError(err)
					return nil
				})
				mockStore.EXPECT().GetRegistryDetailsForApp(app.ID).Times(1).Return(registrytypes.RegistrySettings{}, nil)
				return mockStore
			},
			want:    DefaultMetricGraphs,
			wantErr: false,
		},
		{
			name: "has graphs - return those graphs",
			app: &types.App{
				ID:       "app-id",
				Slug:     "app-slug",
				IsAirgap: false,
			},
			sequence: 1,
			files: map[string]string{
				"upstream/app.yaml": `
apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: my-application
spec:
  title: My Application
  graphs:
    - title: test-graph`,
			},
			mockStoreFn: func(app *types.App, sequence int64, archiveDir string, files map[string]string) *mock_store.MockStore {
				mockStore := mock_store.NewMockStore(ctrl)
				mockStore.EXPECT().GetAppVersionArchive(app.ID, sequence, gomock.Any()).Times(1).DoAndReturn(func(id string, seq int64, archDir string) error {
					err := setupDirectoriesAndFiles(archDir, files)
					req.NoError(err)
					return nil
				})
				mockStore.EXPECT().GetRegistryDetailsForApp(app.ID).Times(1).Return(registrytypes.RegistrySettings{}, nil)
				return mockStore
			},
			want: []kotsv1beta1.MetricGraph{
				{
					Title: "test-graph",
				},
			},
			wantErr: false,
		},
		{
			name: "has graphs with templated queries - return those graphs with queries rendered",
			app: &types.App{
				ID:       "app-id",
				Slug:     "app-slug",
				IsAirgap: false,
			},
			sequence: 1,
			files: map[string]string{
				"upstream/app.yaml": `
apiVersion: kots.io/v1beta1
kind: Application
metadata:
    name: my-application
spec:
  title: My Application
  graphs:
    - title: test-graph
      query: '{{repl ConfigOption "my_query"}}'`,
				"upstream/userdata/config.yaml": `
apiVersion: kots.io/v1beta1
kind: ConfigValues
metadata:
  name: my-app
spec:
  values:
    my_query:
      value: this-is-a-test-query`,
			},
			mockStoreFn: func(app *types.App, sequence int64, archiveDir string, files map[string]string) *mock_store.MockStore {
				mockStore := mock_store.NewMockStore(ctrl)
				mockStore.EXPECT().GetAppVersionArchive(app.ID, sequence, gomock.Any()).Times(1).DoAndReturn(func(id string, seq int64, archDir string) error {
					err := setupDirectoriesAndFiles(archDir, files)
					req.NoError(err)
					return nil
				})
				mockStore.EXPECT().GetRegistryDetailsForApp(app.ID).Times(1).Return(registrytypes.RegistrySettings{}, nil)
				return mockStore
			},
			want: []kotsv1beta1.MetricGraph{
				{
					Title: "test-graph",
					Query: "this-is-a-test-query",
				},
			},
			wantErr: false,
		},
		{
			name: "has graphs with multiple templated queries - return those graphs with all queries rendered",
			app: &types.App{
				ID:       "app-id",
				Slug:     "app-slug",
				IsAirgap: false,
			},
			sequence: 1,
			files: map[string]string{
				"upstream/app.yaml": `
apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: my-application
spec:
  title: My Application
  graphs:
    - title: test-graph
      queries:
        - query: '{{repl ConfigOption "my_query_one"}}'
        - query: '{{repl ConfigOption "my_query_two"}}'`,
				"upstream/userdata/config.yaml": `
apiVersion: kots.io/v1beta1
kind: ConfigValues
metadata:
  name: my-app
spec:
  values:
    my_query_one:
      value: this-is-test-query-one
    my_query_two:
      value: this-is-test-query-two`,
			},
			mockStoreFn: func(app *types.App, sequence int64, archiveDir string, files map[string]string) *mock_store.MockStore {
				mockStore := mock_store.NewMockStore(ctrl)
				mockStore.EXPECT().GetAppVersionArchive(app.ID, sequence, gomock.Any()).Times(1).DoAndReturn(func(id string, seq int64, archDir string) error {
					err := setupDirectoriesAndFiles(archDir, files)
					req.NoError(err)
					return nil
				})
				mockStore.EXPECT().GetRegistryDetailsForApp(app.ID).Times(1).Return(registrytypes.RegistrySettings{}, nil)
				return mockStore
			},
			want: []kotsv1beta1.MetricGraph{
				{
					Title: "test-graph",
					Queries: []kotsv1beta1.MetricQuery{
						{
							Query: "this-is-test-query-one",
						},
						{
							Query: "this-is-test-query-two",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "has graphs with templated queries and prometheus returned elements - return those graphs with queries rendered",
			app: &types.App{
				ID:       "app-id",
				Slug:     "app-slug",
				IsAirgap: false,
			},
			sequence: 1,
			files: map[string]string{
				"upstream/app.yaml": `
apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: my-application
spec:
  title: My Application
  graphs:
    - title: test-graph
      query: '{{repl ConfigOption "my_query"}}-{{ some_prom_value }}'`,
				"upstream/userdata/config.yaml": `
apiVersion: kots.io/v1beta1
kind: ConfigValues
metadata:
  name: my-app
spec:
  values:
    my_query:
      value: this-is-a-test-query`,
			},
			mockStoreFn: func(app *types.App, sequence int64, archiveDir string, files map[string]string) *mock_store.MockStore {
				mockStore := mock_store.NewMockStore(ctrl)
				mockStore.EXPECT().GetAppVersionArchive(app.ID, sequence, gomock.Any()).Times(1).DoAndReturn(func(id string, seq int64, archDir string) error {
					err := setupDirectoriesAndFiles(archDir, files)
					req.NoError(err)
					return nil
				})
				mockStore.EXPECT().GetRegistryDetailsForApp(app.ID).Times(1).Return(registrytypes.RegistrySettings{}, nil)
				return mockStore
			},
			want: []kotsv1beta1.MetricGraph{
				{
					Title: "test-graph",
					Query: "this-is-a-test-query-{{ some_prom_value }}",
				},
			},
			wantErr: false,
		},
		{
			name: "has graphs with templated queries containing invalid templates - return the default graphs",
			app: &types.App{
				ID:       "app-id",
				Slug:     "app-slug",
				IsAirgap: false,
			},
			sequence: 1,
			files: map[string]string{
				"upstream/app.yaml": `
apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: my-application
spec:
  title: My Application
  graphs:
    - title: test-graph
      query: '{{repl NotARealTemplate}}'`,
			},
			mockStoreFn: func(app *types.App, sequence int64, archiveDir string, files map[string]string) *mock_store.MockStore {
				mockStore := mock_store.NewMockStore(ctrl)
				mockStore.EXPECT().GetAppVersionArchive(app.ID, sequence, gomock.Any()).Times(1).DoAndReturn(func(id string, seq int64, archDir string) error {
					err := setupDirectoriesAndFiles(archDir, files)
					req.NoError(err)
					return nil
				})
				mockStore.EXPECT().GetRegistryDetailsForApp(app.ID).Times(1).Return(registrytypes.RegistrySettings{}, nil)
				return mockStore
			},
			want:    DefaultMetricGraphs,
			wantErr: true,
		},
		{
			name: "has graphs with templated title, legend, and axis format/template - return those graphs with these fields rendered",
			app: &types.App{
				ID:       "app-id",
				Slug:     "app-slug",
				IsAirgap: false,
			},
			sequence: 1,
			files: map[string]string{
				"upstream/app.yaml": `
apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: my-application
spec:
  title: My Application
  graphs:
    - title: '{{repl ConfigOption "my_title"}}'
      queries:
        - query: 'non-templated-query'
          legend: '{{repl ConfigOption "my_legend"}}'
      legend: '{{repl ConfigOption "my_legend"}}'
      yAxisFormat: '{{repl ConfigOption "my_x_axis_format"}}'
      yAxisTemplate: '{{repl ConfigOption "my_x_axis_template"}}'`,
				"upstream/userdata/config.yaml": `
apiVersion: kots.io/v1beta1
kind: ConfigValues
metadata:
  name: my-app
spec:
  values:
    my_title:
      value: this-is-a-test-title
    my_legend:
      value: this-is-a-test-legend
    my_x_axis_format:
      value: this-is-a-test-x-axis-format
    my_x_axis_template:
      value: this-is-a-test-x-axis-template`,
			},
			mockStoreFn: func(app *types.App, sequence int64, archiveDir string, files map[string]string) *mock_store.MockStore {
				mockStore := mock_store.NewMockStore(ctrl)
				mockStore.EXPECT().GetAppVersionArchive(app.ID, sequence, gomock.Any()).Times(1).DoAndReturn(func(id string, seq int64, archDir string) error {
					err := setupDirectoriesAndFiles(archDir, files)
					req.NoError(err)
					return nil
				})
				mockStore.EXPECT().GetRegistryDetailsForApp(app.ID).Times(1).Return(registrytypes.RegistrySettings{}, nil)
				return mockStore
			},
			want: []kotsv1beta1.MetricGraph{
				{
					Title: "this-is-a-test-title",
					Queries: []kotsv1beta1.MetricQuery{
						{
							Query:  "non-templated-query",
							Legend: "this-is-a-test-legend",
						},
					},
					Legend:        "this-is-a-test-legend",
					YAxisFormat:   "this-is-a-test-x-axis-format",
					YAxisTemplate: "this-is-a-test-x-axis-template",
				},
			},
			wantErr: false,
		},
		{
			name: "fails to get app version archive - return default graphs and error",
			app: &types.App{
				ID:       "app-id",
				Slug:     "app-slug",
				IsAirgap: false,
			},
			sequence: 1,
			mockStoreFn: func(app *types.App, sequence int64, archiveDir string, files map[string]string) *mock_store.MockStore {
				mockStore := mock_store.NewMockStore(ctrl)
				mockStore.EXPECT().GetAppVersionArchive(app.ID, sequence, gomock.Any()).Times(1).Return(errors.New("failed to get app version archive"))
				return mockStore
			},
			want:    DefaultMetricGraphs,
			wantErr: true,
		},
		{
			name: "fails to get registry details - return default graphs and error",
			app: &types.App{
				ID:       "app-id",
				Slug:     "app-slug",
				IsAirgap: false,
			},
			sequence: 1,
			files: map[string]string{
				"upstream/app.yaml": `
apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: my-application
spec:
  title: My Application`,
			},
			mockStoreFn: func(app *types.App, sequence int64, archiveDir string, files map[string]string) *mock_store.MockStore {
				mockStore := mock_store.NewMockStore(ctrl)
				mockStore.EXPECT().GetAppVersionArchive(app.ID, sequence, gomock.Any()).Times(1).DoAndReturn(func(id string, seq int64, archDir string) error {
					err := setupDirectoriesAndFiles(archDir, files)
					req.NoError(err)
					return nil
				})
				mockStore.EXPECT().GetRegistryDetailsForApp(app.ID).Times(1).Return(registrytypes.RegistrySettings{}, errors.New("failed to get registry details"))
				return mockStore
			},
			want:    DefaultMetricGraphs,
			wantErr: true,
		},
		{
			name: "fails to render file because of an invalid field in kots kinds - return default graphs and error",
			app: &types.App{
				ID:       "app-id",
				Slug:     "app-slug",
				IsAirgap: false,
			},
			sequence: 1,
			files: map[string]string{
				"upstream/app.yaml": `
apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: my-application
spec:
  title: My Application`,
				"upstream/userdata/installation.yaml": `
apiVersion: kots.io/v1beta1
kind: Installation
metadata:
  name: my-application
spec:
  encryptionKey: this-is-not-an-encryption-key`,
			},
			mockStoreFn: func(app *types.App, sequence int64, archiveDir string, files map[string]string) *mock_store.MockStore {
				mockStore := mock_store.NewMockStore(ctrl)
				mockStore.EXPECT().GetAppVersionArchive(app.ID, sequence, gomock.Any()).Times(1).DoAndReturn(func(id string, seq int64, archDir string) error {
					err := setupDirectoriesAndFiles(archDir, files)
					req.NoError(err)
					return nil
				})
				mockStore.EXPECT().GetRegistryDetailsForApp(app.ID).Times(1).Return(registrytypes.RegistrySettings{}, nil)
				return mockStore
			},
			want:    DefaultMetricGraphs,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			archiveDir, err := os.MkdirTemp("", fmt.Sprintf("kotsadm"))
			req.NoError(err)
			defer os.RemoveAll(archiveDir)
			mockStore := tt.mockStoreFn(tt.app, tt.sequence, archiveDir, tt.files)
			got, err := GetGraphs(tt.app, tt.sequence, mockStore)
			if tt.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
			}
			req.Equal(tt.want, got)
		})
	}
}

func setupDirectoriesAndFiles(archiveDir string, files map[string]string) error {
	for path, content := range files {
		dir := filepath.Dir(path)
		if err := os.MkdirAll(filepath.Join(archiveDir, dir), 0744); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(archiveDir, path), []byte(content), 0644); err != nil {
			return err
		}
	}
	return nil
}
