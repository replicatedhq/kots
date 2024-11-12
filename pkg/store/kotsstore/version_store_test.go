package kotsstore

import (
	"testing"

	"github.com/golang/mock/gomock"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	"github.com/replicatedhq/kots/pkg/store/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	troubleshootv1beta2 "github.com/replicatedhq/troubleshoot/pkg/apis/troubleshoot/v1beta2"
	"github.com/replicatedhq/troubleshoot/pkg/multitype"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_determineDownstreamVersionStatus(t *testing.T) {
	tests := []struct {
		name           string
		app            *apptypes.App
		sequence       int64
		kotsKinds      *kotsutil.KotsKinds
		isInstall      bool
		isAutomated    bool
		configFile     string
		skipPreflights bool
		setup          func(t *testing.T, mockStore *mock_store.MockStore)
		expected       types.DownstreamVersionStatus
	}{
		{
			name: "embedded cluster installation without config file",
			app: &apptypes.App{
				ID: "test-app",
			},
			isInstall: true,
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "1234")
			},
			expected: types.VersionPendingClusterManagement,
		},
		{
			name: "embedded cluster installation with config file",
			app: &apptypes.App{
				ID: "test-app",
			},
			isInstall:  true,
			configFile: "config.yaml",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "1234")
			},
			expected: types.VersionPending,
		},
		{
			name: "embedded cluster update without config file",
			app: &apptypes.App{
				ID: "test-app",
			},
			isInstall: false,
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "1234")
			},
			expected: types.VersionPending,
		},
		{
			name: "embedded cluster update with config file",
			app: &apptypes.App{
				ID: "test-app",
			},
			isInstall:  false,
			configFile: "config.yaml",
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "1234")
			},
			expected: types.VersionPending,
		},
		{
			name: "app needs configuration in automated installs",
			app: &apptypes.App{
				ID:   "test-app",
				Slug: "test-app",
			},
			sequence:    1,
			isInstall:   true,
			isAutomated: true,
			kotsKinds: &kotsutil.KotsKinds{
				Config: &kotsv1beta1.Config{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "Config",
					},
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Items: []kotsv1beta1.ConfigItem{
									{
										Name:     "required_item",
										Required: true,
									},
								},
							},
						},
					},
				},
			},
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				mockStore.EXPECT().GetRegistryDetailsForApp("test-app").Return(registrytypes.RegistrySettings{}, nil)
			},
			expected: types.VersionPendingConfig,
		},
		{
			name: "app needs configuration in non-automated installs",
			app: &apptypes.App{
				ID:   "test-app",
				Slug: "test-app",
			},
			sequence:    1,
			isInstall:   true,
			isAutomated: false,
			kotsKinds: &kotsutil.KotsKinds{
				Config: &kotsv1beta1.Config{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "Config",
					},
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Items: []kotsv1beta1.ConfigItem{
									{
										Name:     "required_item",
										Required: true,
									},
								},
							},
						},
					},
				},
			},
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				mockStore.EXPECT().GetRegistryDetailsForApp("test-app").Return(registrytypes.RegistrySettings{}, nil)
			},
			expected: types.VersionPendingConfig,
		},
		{
			name: "app needs configuration in non-automated installs even if all items are optional",
			app: &apptypes.App{
				ID:   "test-app",
				Slug: "test-app",
			},
			sequence:    1,
			isInstall:   true,
			isAutomated: false,
			kotsKinds: &kotsutil.KotsKinds{
				Config: &kotsv1beta1.Config{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "Config",
					},
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Items: []kotsv1beta1.ConfigItem{
									{
										Name:     "optional_item_1",
										Required: false,
									},
									{
										Name:     "optional_item_2",
										Required: false,
									},
								},
							},
						},
					},
				},
			},
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				mockStore.EXPECT().GetRegistryDetailsForApp("test-app").Return(registrytypes.RegistrySettings{}, nil)
			},
			expected: types.VersionPendingConfig,
		},
		{
			name: "app needs configuration in app updates",
			app: &apptypes.App{
				ID:   "test-app",
				Slug: "test-app",
			},
			sequence:    1,
			isInstall:   false,
			isAutomated: false,
			kotsKinds: &kotsutil.KotsKinds{
				Config: &kotsv1beta1.Config{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "Config",
					},
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Items: []kotsv1beta1.ConfigItem{
									{
										Name:     "required_item",
										Required: true,
									},
								},
							},
						},
					},
				},
			},
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				mockStore.EXPECT().GetRegistryDetailsForApp("test-app").Return(registrytypes.RegistrySettings{}, nil)
			},
			expected: types.VersionPendingConfig,
		},
		{
			name: "optional configurations in automated installs should skip config page",
			app: &apptypes.App{
				ID:   "test-app",
				Slug: "test-app",
			},
			sequence:    1,
			isInstall:   true,
			isAutomated: true,
			kotsKinds: &kotsutil.KotsKinds{
				Config: &kotsv1beta1.Config{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "Config",
					},
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Items: []kotsv1beta1.ConfigItem{
									{
										Name:     "optional_item_1",
										Required: false,
									},
									{
										Name:     "optional_item_2",
										Required: false,
									},
								},
							},
						},
					},
				},
			},
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				mockStore.EXPECT().GetRegistryDetailsForApp("test-app").Return(registrytypes.RegistrySettings{}, nil)
			},
			expected: types.VersionPending,
		},
		{
			name: "optional configurations in app update should not require config",
			app: &apptypes.App{
				ID:   "test-app",
				Slug: "test-app",
			},
			sequence:    1,
			isInstall:   false,
			isAutomated: false,
			kotsKinds: &kotsutil.KotsKinds{
				Config: &kotsv1beta1.Config{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "Config",
					},
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Items: []kotsv1beta1.ConfigItem{
									{
										Name:     "optional_item_1",
										Required: false,
									},
									{
										Name:     "optional_item_2",
										Required: false,
									},
								},
							},
						},
					},
				},
			},
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				mockStore.EXPECT().GetRegistryDetailsForApp("test-app").Return(registrytypes.RegistrySettings{}, nil)
			},
			expected: types.VersionPending,
		},
		{
			name: "has optional preflights and not skipped",
			app: &apptypes.App{
				ID:   "test-app",
				Slug: "test-app",
			},
			kotsKinds: &kotsutil.KotsKinds{
				Preflight: &troubleshootv1beta2.Preflight{
					Spec: troubleshootv1beta2.PreflightSpec{
						Analyzers: []*troubleshootv1beta2.Analyze{
							{},
						},
					},
				},
			},
			skipPreflights: false,
			expected:       types.VersionPendingPreflight,
		},
		{
			name: "has optional preflights and is skipped",
			app: &apptypes.App{
				ID:   "test-app",
				Slug: "test-app",
			},
			kotsKinds: &kotsutil.KotsKinds{
				Preflight: &troubleshootv1beta2.Preflight{
					Spec: troubleshootv1beta2.PreflightSpec{
						Analyzers: []*troubleshootv1beta2.Analyze{
							{},
						},
					},
				},
			},
			skipPreflights: true,
			expected:       types.VersionPending,
		},
		{
			name: "cannot skip strict preflights",
			app: &apptypes.App{
				ID:   "test-app",
				Slug: "test-app",
			},
			kotsKinds: &kotsutil.KotsKinds{
				Preflight: &troubleshootv1beta2.Preflight{
					Spec: troubleshootv1beta2.PreflightSpec{
						Analyzers: []*troubleshootv1beta2.Analyze{
							{
								ClusterVersion: &troubleshootv1beta2.ClusterVersion{
									AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
										Strict: multitype.FromBool(true),
									},
								},
							},
						},
					},
				},
			},
			skipPreflights: true,
			expected:       types.VersionPendingPreflight,
		},
		{
			name: "can skip excluded strict preflights",
			app: &apptypes.App{
				ID:   "test-app",
				Slug: "test-app",
			},
			kotsKinds: &kotsutil.KotsKinds{
				Preflight: &troubleshootv1beta2.Preflight{
					Spec: troubleshootv1beta2.PreflightSpec{
						Analyzers: []*troubleshootv1beta2.Analyze{
							{
								ClusterVersion: &troubleshootv1beta2.ClusterVersion{
									AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
										Strict:  multitype.FromBool(true),
										Exclude: multitype.FromBool(true),
									},
								},
							},
						},
					},
				},
			},
			skipPreflights: true,
			expected:       types.VersionPending,
		},
		{
			name: "should not run preflights if all preflights are excluded",
			app: &apptypes.App{
				ID:   "test-app",
				Slug: "test-app",
			},
			kotsKinds: &kotsutil.KotsKinds{
				Preflight: &troubleshootv1beta2.Preflight{
					Spec: troubleshootv1beta2.PreflightSpec{
						Analyzers: []*troubleshootv1beta2.Analyze{
							{
								ClusterVersion: &troubleshootv1beta2.ClusterVersion{
									AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
										Exclude: multitype.FromBool(true),
									},
								},
							},
							{
								ClusterVersion: &troubleshootv1beta2.ClusterVersion{
									AnalyzeMeta: troubleshootv1beta2.AnalyzeMeta{
										Strict:  multitype.FromBool(true),
										Exclude: multitype.FromBool(true),
									},
								},
							},
						},
					},
				},
			},
			skipPreflights: false,
			expected:       types.VersionPending,
		},
		{
			name: "embedded cluster shows cluster management first",
			app: &apptypes.App{
				ID:   "test-app",
				Slug: "test-app",
			},
			isInstall: true,
			kotsKinds: &kotsutil.KotsKinds{
				Config: &kotsv1beta1.Config{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "Config",
					},
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Items: []kotsv1beta1.ConfigItem{
									{
										Name: "item_1",
									},
								},
							},
						},
					},
				},
				Preflight: &troubleshootv1beta2.Preflight{
					Spec: troubleshootv1beta2.PreflightSpec{
						Analyzers: []*troubleshootv1beta2.Analyze{
							{},
						},
					},
				},
			},
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				t.Setenv("EMBEDDED_CLUSTER_ID", "1234")
			},
			expected: types.VersionPendingClusterManagement,
		},
		{
			name: "config comes before preflights",
			app: &apptypes.App{
				ID:   "test-app",
				Slug: "test-app",
			},
			isInstall: true,
			kotsKinds: &kotsutil.KotsKinds{
				Config: &kotsv1beta1.Config{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "kots.io/v1beta1",
						Kind:       "Config",
					},
					Spec: kotsv1beta1.ConfigSpec{
						Groups: []kotsv1beta1.ConfigGroup{
							{
								Items: []kotsv1beta1.ConfigItem{
									{
										Name: "item_1",
									},
								},
							},
						},
					},
				},
				Preflight: &troubleshootv1beta2.Preflight{
					Spec: troubleshootv1beta2.PreflightSpec{
						Analyzers: []*troubleshootv1beta2.Analyze{
							{},
						},
					},
				},
			},
			setup: func(t *testing.T, mockStore *mock_store.MockStore) {
				mockStore.EXPECT().GetRegistryDetailsForApp("test-app").Return(registrytypes.RegistrySettings{}, nil)
			},
			expected: types.VersionPendingConfig,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := require.New(t)

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStore := mock_store.NewMockStore(ctrl)
			if test.setup != nil {
				test.setup(t, mockStore)
			}

			status, err := determineDownstreamVersionStatus(
				mockStore,
				test.app,
				test.sequence,
				test.kotsKinds,
				test.isInstall,
				test.isAutomated,
				test.configFile,
				test.skipPreflights,
			)
			req.NoError(err)
			req.Equal(test.expected, status)
		})
	}
}
