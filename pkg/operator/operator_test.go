package operator_test

import (
	"errors"
	"fmt"
	"os"
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
			)

			BeforeEach(func() {
				mockCtrl = gomock.NewController(GinkgoT())
				mockStore = mock_store.NewMockStore(mockCtrl)

				mockClient = mock_client.NewMockClientInterface(mockCtrl)
				testOperator = operator.Init(mockClient, mockStore, clusterToken)
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
					err := setupDirectoriesAndFiles(archiveDir, true)
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
				testOperator = operator.Init(mockClient, mockStore, clusterToken)
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
			)

			BeforeEach(func() {
				os.Setenv("KOTSADM_ENV", "test")
				previouslyDeployedSequence = -1
				mockCtrl = gomock.NewController(GinkgoT())
				mockStore = mock_store.NewMockStore(mockCtrl)

				mockClient = mock_client.NewMockClientInterface(mockCtrl)
				testOperator = operator.Init(mockClient, mockStore, clusterToken)
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
					err := setupDirectoriesAndFiles(archiveDir, true)
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
						err := setupDirectoriesAndFiles(archiveDir, true)
						Expect(err).ToNot(HaveOccurred())
						return nil
					})
					invalidPreviousDeployment := mockStore.EXPECT().GetAppVersionArchive(appID, sequence, gomock.Any()).DoAndReturn(func(id string, seq int64, archDir string) error {
						archiveDir = archDir
						err := setupDirectoriesAndFiles(archiveDir, false)
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
	})
})

func setupDirectoriesAndFiles(archiveDir string, validDeployment bool) error {
	overlaysDir := fmt.Sprintf("%s/overlays", archiveDir)
	midstreamDir := fmt.Sprintf("%s/midstream", overlaysDir)
	downstreamsDir := fmt.Sprintf("%s/downstreams", overlaysDir)

	if _, err := os.Stat(overlaysDir); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir(overlaysDir, 0700)
		if err != nil {
			return err
		}
	}

	if _, err := os.Stat(midstreamDir); errors.Is(err, os.ErrNotExist) {
		err = os.Mkdir(midstreamDir, 0700)
		if err != nil {
			return err
		}
	}
	if _, err := os.Stat(downstreamsDir); errors.Is(err, os.ErrNotExist) {
		err = os.Mkdir(downstreamsDir, 0700)
		if err != nil {
			return err
		}
	}

	err := writeKustomizationFile(fmt.Sprintf("%s/overlays/midstream/kustomization.yaml", archiveDir))
	if err != nil {
		return err
	}

	err = writeKustomizationFile(fmt.Sprintf("%s/overlays/downstreams/kustomization.yaml", archiveDir))
	if err != nil {
		return err
	}

	err = writeDeploymentFile(fmt.Sprintf("%s/overlays/downstreams/deployment.yaml", archiveDir), validDeployment)
	if err != nil {
		return err
	}

	err = writeAppFile(fmt.Sprintf("%s/overlays/downstreams/app.yaml", archiveDir))
	if err != nil {
		return err
	}

	return nil
}

func writeKustomizationFile(filename string) error {
	exampleKustomizationFileContents := `
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - deployment.yaml
  - app.yaml`

	kustomizeFile, err := os.Create(filename)
	if err != nil {
		return err
	}

	_, err = kustomizeFile.Write([]byte(exampleKustomizationFileContents))
	return err
}

func writeDeploymentFile(filename string, valid bool) error {
	name := `  name: "some-deployment"`
	if !valid {
		name = ""
	}
	exampleDeploymentFileContents := fmt.Sprintf(`
apiVersion: apps/v1
kind: Deployment
metadata:
%s
  labels:
    app: example
    component: nginx
spec:
  selector:
    matchLabels:
      app: example
      component: nginx
  template:
    metadata:
      labels:
        app: example
        component: nginx
    spec:
      containers:
        - name: nginx
          image: nginx
          envFrom:
          - configMapRef:
              name: example-configmap
          resources:
            limits:
              memory: '256Mi'
              cpu: '500m'
            requests:
              memory: '32Mi'
              cpu: '100m'`, name)

	deploymentFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	_, err = deploymentFile.Write([]byte(exampleDeploymentFileContents))

	return err
}

func writeAppFile(filename string) error {
	exampleAppFileContents := `
apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: my-application
spec:
  statusInformers:
    - deployment/some-deployment`

	deploymentFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	_, err = deploymentFile.Write([]byte(exampleAppFileContents))
	return err
}
