package operator_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/operator"
	mock_client "github.com/replicatedhq/kots/pkg/operator/client/mock"
	operatortypes "github.com/replicatedhq/kots/pkg/operator/types"
	registrytypes "github.com/replicatedhq/kots/pkg/registry/types"
	mock_store "github.com/replicatedhq/kots/pkg/store/mock"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"k8s.io/client-go/kubernetes/fake"
)

var _ = Describe("Operator", func() {
	Describe("Start()", func() {
		When("there is a currently deployed app sequence", func() {
			var (
				mockStore    *mock_store.MockStore
				mockClient   *mock_client.MockClientInterface
				testOperator *operator.Operator
				mockCtrl     *gomock.Controller
				clusterToken       = "cluster-token"
				appID              = "some-app-id"
				sequence     int64 = 0
				archiveDir   string

				archiveFiles = map[string]string{
					"upstream/app.yaml": `
apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: my-application
spec:
  statusInformers:
    - deployment/some-deployment`,
				}
			)

			BeforeEach(func() {
				mockCtrl = gomock.NewController(GinkgoT())
				mockStore = mock_store.NewMockStore(mockCtrl)

				mockClient = mock_client.NewMockClientInterface(mockCtrl)
				mockK8sClientset := fake.NewSimpleClientset()

				testOperator = operator.Init(mockClient, mockStore, clusterToken, mockK8sClientset)
			})

			AfterEach(func() {
				mockCtrl.Finish()

				err := os.RemoveAll(archiveDir)
				Expect(err).ToNot(HaveOccurred())
			})

			It("starts the status informers", func() {
				mockClient.EXPECT().Init().Return(nil)

				mockStore.EXPECT().GetClusterIDFromDeployToken(clusterToken).Return("", nil)

				apps := []*apptypes.App{
					{
						ID:                    appID,
						Slug:                  "some-app-slug",
						IsAirgap:              false,
						RestoreInProgressName: "",
					},
				}
				mockStore.EXPECT().ListAppsForDownstream("").AnyTimes().Return(apps, nil)

				deployedVersion := &downstreamtypes.DownstreamVersion{
					ParentSequence: sequence,
					Status:         storetypes.VersionDeployed,
				}
				mockStore.EXPECT().GetCurrentDownstreamVersion(appID, "").AnyTimes().Return(deployedVersion, nil)

				mockStore.EXPECT().GetAppVersionArchive(appID, sequence, gomock.Any()).DoAndReturn(func(id string, seq int64, archDir string) error {
					archiveDir = archDir
					err := writeArchiveFiles(archiveDir, archiveFiles)
					Expect(err).ToNot(HaveOccurred())
					return nil
				})

				registrySettings := registrytypes.RegistrySettings{
					Hostname:   "hostname",
					Username:   "user",
					Password:   "pass",
					Namespace:  "namespace",
					IsReadOnly: false,
				}
				mockStore.EXPECT().GetRegistryDetailsForApp(appID).Return(registrySettings, nil)

				wg := sync.WaitGroup{}
				wg.Add(1)
				mockClient.EXPECT().ApplyAppInformers(gomock.Any()).Times(1).Do(func(args operatortypes.AppInformersArgs) {
					wg.Done()
				})

				err := testOperator.Start()
				Expect(err).ToNot(HaveOccurred())

				done := make(chan struct{})
				go func() {
					wg.Wait()
					close(done)
				}()

				// wait for the informers to start or timeout
				select {
				case <-done:
				case <-time.After(2 * time.Second):
					Fail("timed out waiting for informers to start")
				}
			})
		})
		When("there is not a currently deployed app sequence", func() {
			var (
				mockStore    *mock_store.MockStore
				mockClient   *mock_client.MockClientInterface
				testOperator *operator.Operator
				mockCtrl     *gomock.Controller
				clusterToken = "cluster-token"
				appID        = "some-app-id"
			)

			BeforeEach(func() {
				mockCtrl = gomock.NewController(GinkgoT())
				mockStore = mock_store.NewMockStore(mockCtrl)

				mockClient = mock_client.NewMockClientInterface(mockCtrl)
				mockK8sClientset := fake.NewSimpleClientset()
				testOperator = operator.Init(mockClient, mockStore, clusterToken, mockK8sClientset)
			})

			AfterEach(func() {
				mockCtrl.Finish()
			})

			It("should not start the status informers", func() {
				mockClient.EXPECT().Init().Return(nil)

				mockStore.EXPECT().GetClusterIDFromDeployToken(clusterToken).Return("", nil)

				apps := []*apptypes.App{
					{
						ID:                    appID,
						Slug:                  "some-app-slug",
						IsAirgap:              false,
						RestoreInProgressName: "",
					},
				}
				mockStore.EXPECT().ListAppsForDownstream("").AnyTimes().Return(apps, nil)

				mockStore.EXPECT().GetCurrentDownstreamVersion(appID, "").AnyTimes().Return(nil, nil)

				wg := sync.WaitGroup{}
				wg.Add(1)
				mockClient.EXPECT().ApplyAppInformers(gomock.Any()).Times(0).Do(func(args operatortypes.AppInformersArgs) {
					wg.Done()
				})

				err := testOperator.Start()
				Expect(err).ToNot(HaveOccurred())

				done := make(chan struct{})
				go func() {
					wg.Wait()
					close(done)
				}()

				// wait for the informers to start or timeout
				select {
				case <-done:
					Fail("informers should not have started")
				case <-time.After(2 * time.Second):
				}
			})
		})
	})

	Describe("DeployApp()", func() {
		When("there is a deployment and app file with a status informer", func() {
			var (
				mockStore    *mock_store.MockStore
				mockClient   *mock_client.MockClientInterface
				testOperator *operator.Operator
				mockCtrl     *gomock.Controller
				clusterToken       = "cluster-token"
				appID              = "some-app-id"
				sequence     int64 = 0

				archiveDir                 string
				previouslyDeployedSequence int64

				archiveFiles = map[string]string{
					"base/kustomization.yaml": `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - deployment.yaml`,
					"overlays/midstream/kustomization.yaml": `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../base`,
					"overlays/downstreams/kustomization.yaml": `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../midstream`,
					"base/deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: some-deployment
  labels:
    app: example
spec:
  selector:
    matchLabels:
      app: example
  template:
    metadata:
      labels:
        app: example
    spec:
      containers:
        - name: nginx
          image: nginx`,
					"upstream/app.yaml": `
apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: my-application
spec:
  statusInformers:
    - deployment/some-deployment`,
				}
			)

			BeforeEach(func() {
				os.Setenv("KOTSADM_ENV", "test")
				previouslyDeployedSequence = -1
				mockCtrl = gomock.NewController(GinkgoT())
				mockStore = mock_store.NewMockStore(mockCtrl)

				mockClient = mock_client.NewMockClientInterface(mockCtrl)
				mockK8sClientset := fake.NewSimpleClientset()
				testOperator = operator.Init(mockClient, mockStore, clusterToken, mockK8sClientset)
			})

			AfterEach(func() {
				os.Setenv("KOTSADM_ENV", "")
				mockCtrl.Finish()

				err := os.RemoveAll(archiveDir)
				Expect(err).ToNot(HaveOccurred())
			})

			It("successfully deploys the app and does not return an error ", func() {
				mockStore.EXPECT().SetDownstreamVersionStatus(appID, sequence, gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

				app := &apptypes.App{
					ID:                    appID,
					Slug:                  "some-app-slug",
					IsAirgap:              false,
					RestoreInProgressName: "",
				}
				mockStore.EXPECT().GetApp(appID).Return(app, nil)

				downstreams := &downstreamtypes.Downstream{}
				mockStore.EXPECT().GetDownstream("").Return(downstreams, nil)

				mockStore.EXPECT().GetAppVersionArchive(appID, sequence, gomock.Any()).DoAndReturn(func(id string, seq int64, archDir string) error {
					archiveDir = archDir
					err := writeArchiveFiles(archiveDir, archiveFiles)
					Expect(err).ToNot(HaveOccurred())
					return nil
				})

				registrySettings := registrytypes.RegistrySettings{
					Hostname:   "hostname",
					Username:   "user",
					Password:   "pass",
					Namespace:  "namespace",
					IsReadOnly: false,
				}
				mockStore.EXPECT().GetRegistryDetailsForApp(appID).Return(registrySettings, nil)

				mockStore.EXPECT().GetPreviouslyDeployedSequence(appID, "").Return(previouslyDeployedSequence, nil)

				mockClient.EXPECT().DeployApp(gomock.Any()).Return(true, nil)

				mockClient.EXPECT().ApplyAppInformers(gomock.Any())

				deployed, err := testOperator.DeployApp(appID, sequence)
				Expect(err).ToNot(HaveOccurred())
				Expect(deployed).To(BeTrue())
			})

			When("a previously deployed application has an error", func() {
				var (
					previousArchiveFiles = map[string]string{
						"base/kustomization.yaml": `
	apiVersion: kustomize.config.k8s.io/v1beta1
	kind: Kustomization
	resources:
	  - deployment.yaml`,
						"overlays/midstream/kustomization.yaml": `
	apiVersion: kustomize.config.k8s.io/v1beta1
	kind: Kustomization
	resources:
	  - ../../base`,
						"overlays/downstreams/kustomization.yaml": `
	apiVersion: kustomize.config.k8s.io/v1beta1
	kind: Kustomization
	resources:
	  - ../midstream`,
						"base/deployment.yaml": `
	apiVersion: apps/v1
	kind: Deployment
	metadata:
	  this is an invalid deployment`,
						"upstream/app.yaml": `
	apiVersion: kots.io/v1beta1
	kind: Application
	metadata:
	  name: my-application
	spec:
	  statusInformers:
		- deployment/some-deployment`,
					}
				)

				BeforeEach(func() {
					previouslyDeployedSequence = 1
				})

				It("deployed the app and does not error if the errors no longer exist", func() {
					mockStore.EXPECT().SetDownstreamVersionStatus(appID, sequence, gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

					app := &apptypes.App{
						ID:                    appID,
						Slug:                  "some-app-slug",
						IsAirgap:              false,
						RestoreInProgressName: "",
					}
					mockStore.EXPECT().GetApp(appID).Return(app, nil)

					downstreams := &downstreamtypes.Downstream{}
					mockStore.EXPECT().GetDownstream("").Return(downstreams, nil)

					validCurrentDeployment := mockStore.EXPECT().GetAppVersionArchive(appID, sequence, gomock.Any()).DoAndReturn(func(id string, seq int64, archDir string) error {
						archiveDir = archDir
						err := writeArchiveFiles(archiveDir, archiveFiles)
						Expect(err).ToNot(HaveOccurred())
						return nil
					})
					invalidPreviousDeployment := mockStore.EXPECT().GetAppVersionArchive(appID, sequence, gomock.Any()).DoAndReturn(func(id string, seq int64, archDir string) error {
						archiveDir = archDir
						err := writeArchiveFiles(archiveDir, previousArchiveFiles)
						Expect(err).ToNot(HaveOccurred())
						return nil
					})
					gomock.InOrder(
						validCurrentDeployment,
						invalidPreviousDeployment,
					)

					registrySettings := registrytypes.RegistrySettings{
						Hostname:   "hostname",
						Username:   "user",
						Password:   "pass",
						Namespace:  "namespace",
						IsReadOnly: false,
					}
					mockStore.EXPECT().GetRegistryDetailsForApp(appID).Return(registrySettings, nil)

					mockStore.EXPECT().GetPreviouslyDeployedSequence(appID, "").Return(previouslyDeployedSequence, nil)

					mockStore.EXPECT().GetParentSequenceForSequence(appID, "", previouslyDeployedSequence).Return(int64(0), nil)

					mockClient.EXPECT().DeployApp(gomock.Any()).DoAndReturn(func(deployArgs operatortypes.DeployAppArgs) (bool, error) {
						Expect(deployArgs.PreviousManifests).To(BeEmpty())
						return true, nil
					})

					mockClient.EXPECT().ApplyAppInformers(gomock.Any())

					deployed, err := testOperator.DeployApp(appID, sequence)
					Expect(err).ToNot(HaveOccurred())
					Expect(deployed).To(BeTrue())
				})
			})
		})

		When("there is a helm chart with template functions", func() {

			var (
				mockStore    *mock_store.MockStore
				mockClient   *mock_client.MockClientInterface
				testOperator *operator.Operator
				mockCtrl     *gomock.Controller
				clusterToken       = "cluster-token"
				appID              = "some-app-id"
				sequence     int64 = 0

				expectedNamespace        = "my-namespace"
				expectedHelmUpgradeFlags = []string{"--set", "extraValue=my-extra-value"}

				archiveDir                 string
				previouslyDeployedSequence int64
				archiveFiles               = map[string]string{
					"base/kustomization.yaml": `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources: []`,
					"overlays/midstream/kustomization.yaml": `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../base`,
					"overlays/downstreams/kustomization.yaml": `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../midstream`,
					"base/charts/my-chart/kustomization.yaml": `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - deployment.yaml`,
					"overlays/midstream/charts/my-chart/kustomization.yaml": `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../../../base/charts/my-chart`,
					"overlays/downstreams/charts/my-chart/kustomization.yaml": `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - ../../../midstream/charts/my-chart`,
					"base/charts/my-chart/deployment.yaml": `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: some-deployment
  labels:
    app: example
spec:
  selector:
    matchLabels:
      app: example
  template:
    metadata:
      labels:
        app: example
    spec:
      containers:
        - name: nginx
          image: nginx`,
					"upstream/app.yaml": `
apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: my-application
spec:
  statusInformers:
    - deployment/some-deployment`,
					"upstream/my-chart.yaml": `
apiVersion: kots.io/v1beta1
kind: HelmChart
metadata:
  name: my-chart
spec:
  namespace: repl{{ ConfigOption "deploy_namespace" }}
  helmUpgradeFlags:
    - --set
    - extraValue=repl{{ ConfigOption "deploy_extra_value" }}
  optionalValues:
  - when: "repl{{ HasLocalRegistry }}"
    recursiveMerge: true
    values:
      global:
        registry: ''
`,
					"upstream/my-other-chart.yaml": `
apiVersion: kots.io/v1beta2
kind: HelmChart
metadata:
  name: my-other-chart
spec:
  namespace: repl{{ ConfigOption "deploy_namespace" }}
  helmUpgradeFlags:
    - --set
    - extraValue=repl{{ ConfigOption "deploy_extra_value" }}
  optionalValues:
  - when: "repl{{ HasLocalRegistry }}"
    recursiveMerge: true
    values:
      global:
        registry: ''
`,
					"upstream/config.yaml": `
apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: my-config
spec:
  groups:
  - name: deployment_settings
    title: Deployment Settings
    items:
    - name: deploy_namespace
      title: Namespace
      type: text
      default: 'default'
    - name: deploy_extra_value
      title: An Extra Value
      type: text
      default: 'default'
`,
					"upstream/userdata/config.yaml": `
apiVersion: kots.io/v1beta1
kind: ConfigValues
metadata:
  name: some-chart
spec:
  values:
    deploy_namespace:
      value: my-namespace
    deploy_extra_value:
      value: my-extra-value
`,
				}
			)

			BeforeEach(func() {
				os.Setenv("KOTSADM_ENV", "test")
				previouslyDeployedSequence = -1
				mockCtrl = gomock.NewController(GinkgoT())
				mockStore = mock_store.NewMockStore(mockCtrl)

				mockClient = mock_client.NewMockClientInterface(mockCtrl)
				mockK8sClientset := fake.NewSimpleClientset()
				testOperator = operator.Init(mockClient, mockStore, clusterToken, mockK8sClientset)
			})

			AfterEach(func() {
				os.Setenv("KOTSADM_ENV", "")
				mockCtrl.Finish()

				err := os.RemoveAll(archiveDir)
				Expect(err).ToNot(HaveOccurred())
			})

			It("installs the helm chart using the templated namespace and upgrade flags", func() {
				mockStore.EXPECT().SetDownstreamVersionStatus(appID, sequence, gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

				app := &apptypes.App{
					ID:                    appID,
					Slug:                  "some-app-slug",
					IsAirgap:              false,
					RestoreInProgressName: "",
				}
				mockStore.EXPECT().GetApp(appID).Return(app, nil)

				downstreams := &downstreamtypes.Downstream{}
				mockStore.EXPECT().GetDownstream("").Return(downstreams, nil)

				mockStore.EXPECT().GetAppVersionArchive(appID, sequence, gomock.Any()).DoAndReturn(func(id string, seq int64, archDir string) error {
					archiveDir = archDir
					err := writeArchiveFiles(archiveDir, archiveFiles)
					Expect(err).ToNot(HaveOccurred())
					return nil
				})

				registrySettings := registrytypes.RegistrySettings{
					Hostname:   "hostname",
					Username:   "user",
					Password:   "pass",
					Namespace:  "namespace",
					IsReadOnly: false,
				}
				mockStore.EXPECT().GetRegistryDetailsForApp(appID).Return(registrySettings, nil)

				mockStore.EXPECT().GetPreviouslyDeployedSequence(appID, "").Return(previouslyDeployedSequence, nil)

				mockClient.EXPECT().DeployApp(gomock.Any()).Do(func(deployArgs operatortypes.DeployAppArgs) (bool, error) {
					// validate that the namespace and helm upgrade flags are templated when deploying
					Expect(deployArgs.KotsKinds.V1Beta1HelmCharts.Items[0].Spec.Namespace).To(Equal(expectedNamespace))
					Expect(deployArgs.KotsKinds.V1Beta1HelmCharts.Items[0].Spec.HelmUpgradeFlags).To(Equal(expectedHelmUpgradeFlags))
					Expect(deployArgs.KotsKinds.V1Beta2HelmCharts.Items[0].Spec.Namespace).To(Equal(expectedNamespace))
					Expect(deployArgs.KotsKinds.V1Beta2HelmCharts.Items[0].Spec.HelmUpgradeFlags).To(Equal(expectedHelmUpgradeFlags))
					return true, nil
				})

				mockClient.EXPECT().ApplyAppInformers(gomock.Any())

				_, err := testOperator.DeployApp(appID, sequence)
				Expect(err).ToNot(HaveOccurred())
			})

		})
	})
})

func writeArchiveFiles(archiveDir string, archiveFiles map[string]string) error {
	for path, content := range archiveFiles {
		archiveFilePath := fmt.Sprintf("%s/%s", archiveDir, path)
		err := os.MkdirAll(filepath.Dir(archiveFilePath), 0700)
		if err != nil {
			return fmt.Errorf("failed to create archive file path %s: %v", archiveFilePath, err)
		}

		err = os.WriteFile(archiveFilePath, []byte(content), 0644)
		if err != nil {
			return fmt.Errorf("failed to write archive file %s: %v", archiveFilePath, err)
		}
	}

	return nil
}
