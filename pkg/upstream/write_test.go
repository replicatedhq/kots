package upstream

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pmezard/go-difflib/difflib"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/store"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
)

func Test_renderValuesYAMLForLicense(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mock_store.NewMockStore(ctrl)

	mockClientset := fake.NewSimpleClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: k8sutil.KotsadmIDConfigMapName},
		Data:       map[string]string{"id": "cluster-id"},
	})

	releasedAt, _ := time.Parse(time.RFC3339, "2020-01-01T00:00:00Z")

	type args struct {
		clientset              kubernetes.Interface
		kotsStore              store.Store
		unrenderedContents     []byte
		u                      *types.Upstream
		replicatedSDKChartName string
		isReplicatedSDK        bool
	}
	tests := []struct {
		name                  string
		args                  args
		mockStoreExpectations func()
		want                  []byte
		wantErr               bool
	}{
		{
			name: "sdk as a subchart - no existing replicated or global values",
			args: args{
				clientset: mockClientset,
				kotsStore: mockStore,
				unrenderedContents: []byte(`# Comment in values
existing: value`),
				u: &types.Upstream{
					License: &kotsv1beta1.License{
						Spec: kotsv1beta1.LicenseSpec{
							LicenseID: "license-id",
							AppSlug:   "app-slug",
							Endpoint:  "https://replicated.app",
							Entitlements: map[string]kotsv1beta1.EntitlementField{
								"license-field": {
									Title:       "License Field",
									Description: "This is a license field",
									ValueType:   "string",
									Value: kotsv1beta1.EntitlementValue{
										Type:   kotsv1beta1.String,
										StrVal: "license-field-value",
									},
								},
							},
							CustomerEmail: "customer@example.com",
							CustomerName:  "Customer Name",
							LicenseType:   "trial",
							Signature:     []byte{},
						},
					},
					Application: &kotsv1beta1.Application{
						Spec: kotsv1beta1.ApplicationSpec{
							Title:           "App Title",
							StatusInformers: []string{"deployment/my-deployment"},
						},
					},
					ReplicatedRegistryDomain: "registry.replicated.com",
					ReplicatedProxyDomain:    "proxy.replicated.com",
					ChannelID:                "channel-id",
					ChannelName:              "channel-name",
					UpdateCursor:             "1",
					ReleaseSequence:          1,
					ReleasedAt:               &releasedAt,
					ReleaseNotes:             "Release Notes",
					VersionLabel:             "1.0.0",
				},
				replicatedSDKChartName: "replicated",
				isReplicatedSDK:        false,
			},
			mockStoreExpectations: func() {
				mockStore.EXPECT().GetAppIDFromSlug("app-slug").Return("app-id", nil)
			},
			want: []byte(`# Comment in values
existing: value
replicated:
  appID: app-id
  appName: App Title
  channelID: channel-id
  channelName: channel-name
  channelSequence: 1
  license: |
    metadata:
      creationTimestamp: null
    spec:
      appSlug: app-slug
      customerEmail: customer@example.com
      customerName: Customer Name
      endpoint: https://replicated.app
      entitlements:
        license-field:
          description: This is a license field
          title: License Field
          value: license-field-value
          valueType: string
      licenseID: license-id
      licenseType: trial
      signature: ""
    status: {}
  releaseCreatedAt: "2020-01-01T00:00:00Z"
  releaseNotes: Release Notes
  releaseSequence: 1
  replicatedAppEndpoint: https://replicated.app
  replicatedID: cluster-id
  statusInformers:
    - deployment/my-deployment
  userAgent: KOTS/v0.0.0-unknown
  versionLabel: 1.0.0
global:
  replicated:
    channelName: channel-name
    customerEmail: customer@example.com
    customerName: Customer Name
    dockerconfigjson: eyJhdXRocyI6eyJwcm94eS5yZXBsaWNhdGVkLmNvbSI6eyJhdXRoIjoiYkdsalpXNXpaUzFwWkRwc2FXTmxibk5sTFdsayJ9LCJyZWdpc3RyeS5yZXBsaWNhdGVkLmNvbSI6eyJhdXRoIjoiYkdsalpXNXpaUzFwWkRwc2FXTmxibk5sTFdsayJ9fX0=
    licenseFields:
      license-field:
        name: license-field
        title: License Field
        description: This is a license field
        value: license-field-value
        valueType: string
    licenseID: license-id
    licenseType: trial
`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockStoreExpectations()
			got, err := renderValuesYAMLForLicense(tt.args.clientset, tt.args.kotsStore, tt.args.unrenderedContents, tt.args.u, tt.args.replicatedSDKChartName, tt.args.isReplicatedSDK)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderValuesYAMLForLicense() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("renderValuesYAMLForLicense() \n\n%s", fmtYAMLDiff(string(got), string(tt.want)))
			}
		})
	}
}

func fmtYAMLDiff(got, want string) string {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(got),
		B:        difflib.SplitLines(want),
		FromFile: "Got",
		ToFile:   "Want",
		Context:  1,
	}
	diffStr, _ := difflib.GetUnifiedDiffString(diff)
	return fmt.Sprintf("got:\n%s \n\nwant:\n%s \n\ndiff:\n%s", got, want, diffStr)
}
