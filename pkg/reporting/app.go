package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	downstreamtypes "github.com/replicatedhq/kots/pkg/api/downstream/types"
	"github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/embeddedcluster"
	"github.com/replicatedhq/kots/pkg/gitops"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/snapshot"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
	"github.com/segmentio/ksuid"
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"k8s.io/client-go/kubernetes"
)

type SnapshotReport struct {
	Provider        string
	FullSchedule    string
	FullTTL         string
	PartialSchedule string
	PartialTTL      string
}

func Init() error {
	err := initFromDownstream()
	if err != nil {
		return errors.Wrap(err, "failed to init from downstream")
	}

	if kotsadm.IsAirgap() {
		clientset, err := k8sutil.GetClientset()
		if err != nil {
			return errors.Wrap(err, "failed to get clientset")
		}
		reporter = &AirgapReporter{
			clientset: clientset,
			store:     store.GetStore(),
		}
	} else {
		reporter = &OnlineReporter{}
	}

	return nil
}

func initFromDownstream() error {
	// Retrieve the ClusterID from store
	clusters, err := store.GetStore().ListClusters()
	if err != nil {
		return errors.Wrap(err, "failed to list clusters")
	}
	if len(clusters) == 0 {
		return nil
	}
	clusterID := clusters[0].ClusterID

	isKotsadmIDGenerated, err := store.GetStore().IsKotsadmIDGenerated()
	if err != nil {
		return errors.Wrap(err, "failed to generate id")
	}
	cmpExists, err := k8sutil.IsKotsadmIDConfigMapPresent()
	if err != nil {
		return errors.Wrap(err, "failed to check configmap")
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	if isKotsadmIDGenerated && !cmpExists {
		kotsadmID := ksuid.New().String()
		err = k8sutil.CreateKotsadmIDConfigMap(clientset, kotsadmID)
	} else if !isKotsadmIDGenerated && !cmpExists {
		err = k8sutil.CreateKotsadmIDConfigMap(clientset, clusterID)
	} else if !isKotsadmIDGenerated && cmpExists {
		err = k8sutil.UpdateKotsadmIDConfigMap(clientset, clusterID)
	} else {
		// id exists and so as configmap, noop
	}
	if err == nil {
		err = store.GetStore().SetIsKotsadmIDGenerated()
	}

	return err
}

func GetReporter() Reporter {
	return reporter
}

func GetReportingInfo(appID string) *types.ReportingInfo {
	if os.Getenv("USE_MOCK_REPORTING") == "1" {
		return &types.ReportingInfo{
			InstanceID: appID,
		}
	}

	r := types.ReportingInfo{
		InstanceID:             appID,
		KOTSInstallID:          os.Getenv("KOTS_INSTALL_ID"),
		KURLInstallID:          os.Getenv("KURL_INSTALL_ID"),
		EmbeddedClusterID:      util.EmbeddedClusterID(),
		EmbeddedClusterVersion: util.EmbeddedClusterVersion(),
		UserAgent:              buildversion.GetUserAgent(),
	}

	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		logger.Warnf("failed to get cluster config: %v", err.Error())
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		logger.Warnf("failed to get clientset: %v", err.Error())
	}
	r.ClusterID = k8sutil.GetKotsadmID(clientset)

	di, err := getDownstreamInfo(appID)
	if err != nil {
		logger.Warnf("failed to get downstream info: %v", err.Error())
	}
	if di != nil {
		r.Downstream = *di
	}

	// get kubernetes cluster version
	k8sVersion, err := k8sutil.GetK8sVersion(clientset)
	if err != nil {
		logger.Warnf("failed to get k8s version: %v", err.Error())
	} else {
		r.K8sVersion = k8sVersion
	}

	if distribution := GetDistribution(clientset); distribution != UnknownDistribution {
		r.K8sDistribution = distribution.String()
	}

	// get app status
	appStatus, err := store.GetStore().GetAppStatus(appID)
	if err != nil {
		logger.Warnf("failed to get app status: %v", err.Error())
	} else {
		r.AppStatus = string(appStatus.State)
	}

	// kurl
	r.IsKurl, err = kurl.IsKurl(clientset)
	if err != nil {
		logger.Warnf(errors.Wrap(err, "failed to check if cluster is kurl").Error())
	}

	if r.IsKurl && clientset != nil {
		kurlNodes, err := kurl.GetNodes(clientset)
		if err != nil {
			logger.Warnf(errors.Wrap(err, "failed to get kurl nodes").Error())
		}
		if kurlNodes != nil {
			for _, kurlNode := range kurlNodes.Nodes {
				r.KurlNodeCountTotal++
				if kurlNode.IsConnected && kurlNode.IsReady {
					r.KurlNodeCountReady++
				}
			}
		}
	}

	// embedded cluster
	if util.IsEmbeddedCluster() && clientset != nil {
		ecNodes, err := embeddedcluster.GetNodes(context.TODO(), clientset)
		if err != nil {
			logger.Warnf("failed to get embedded cluster nodes: %v", err.Error())
		}
		if ecNodes != nil {
			marshalled, err := json.Marshal(ecNodes.Nodes)
			if err != nil {
				logger.Warnf("failed to marshal embedded cluster node: %v", err.Error())
			} else {
				r.EmbeddedClusterNodes = string(marshalled)
			}
		}
	}

	r.IsGitOpsEnabled, r.GitOpsProvider = getGitOpsReport(clientset, appID, r.ClusterID)

	veleroClient, err := k8sutil.GetKubeClient(context.TODO())
	if err != nil {
		logger.Warnf("failed to get velero client: %v", err.Error())
	}

	if clientset != nil && veleroClient != nil {
		bsl, err := snapshot.FindBackupStoreLocation(context.TODO(), clientset, veleroClient, util.PodNamespace)
		if err != nil {
			logger.Warnf("failed to find backup store location: %v", err.Error())
		} else {
			report, err := getSnapshotReport(store.GetStore(), bsl, appID, r.ClusterID)
			if err != nil {
				logger.Warnf("failed to get snapshot report: %v", err.Error())
			} else {
				r.SnapshotProvider = report.Provider
				r.SnapshotFullSchedule = report.FullSchedule
				r.SnapshotFullTTL = report.FullTTL
				r.SnapshotPartialSchedule = report.PartialSchedule
				r.SnapshotPartialTTL = report.PartialTTL
			}
		}
	}

	return &r
}

func getDownstreamInfo(appID string) (*types.DownstreamInfo, error) {
	di := types.DownstreamInfo{}

	downstreams, err := store.GetStore().ListDownstreamsForApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list downstreams for app")
	}
	if len(downstreams) == 0 {
		// this can happen during initial install workflow until the app is installed and has downstreams
		logger.Debugf("no downstreams found for app")
		return nil, nil
	}

	currentVersion, err := store.GetStore().GetCurrentDownstreamVersion(appID, downstreams[0].ClusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current downstream parent sequence")
	}

	// info about the deployed app sequence
	if currentVersion != nil {
		deployedArchiveDir, err := ioutil.TempDir("", "kotsadm")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create temp dir")
		}
		defer os.RemoveAll(deployedArchiveDir)

		err = store.GetStore().GetAppVersionArchive(appID, currentVersion.ParentSequence, deployedArchiveDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get app version archive")
		}

		deployedKotsKinds, err := kotsutil.LoadKotsKinds(deployedArchiveDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load kotskinds from path")
		}

		di.Cursor = deployedKotsKinds.Installation.Spec.UpdateCursor
		di.ChannelID = deployedKotsKinds.Installation.Spec.ChannelID
		di.ChannelName = deployedKotsKinds.Installation.Spec.ChannelName
		di.Sequence = &currentVersion.Sequence
		di.Source = currentVersion.Source
		di.Status = string(currentVersion.Status)

		var preflightResults *troubleshootpreflight.UploadPreflightResults
		if err := json.Unmarshal([]byte(currentVersion.PreflightResult), &preflightResults); err != nil {
			logger.Warnf("failed to unmarshal preflight results: %v", err.Error())
		}
		di.PreflightState = getPreflightState(preflightResults)
		di.SkipPreflights = currentVersion.PreflightSkipped

		if deployedKotsKinds.V1Beta1HelmCharts != nil && len(deployedKotsKinds.V1Beta1HelmCharts.Items) > 0 {
			for _, chart := range deployedKotsKinds.V1Beta1HelmCharts.Items {
				if chart.Spec.UseHelmInstall {
					di.NativeHelmInstalls++
				} else {
					di.ReplHelmInstalls++
				}
			}
		}
	}

	return &di, nil
}

func getGitOpsReport(clientset kubernetes.Interface, appID string, clusterID string) (bool, string) {
	gitOpsConfig, err := gitops.GetDownstreamGitOpsConfig(clientset, appID, clusterID)
	if err != nil {
		logger.Warnf("failed to get gitops config: %v", err.Error())
		return false, ""
	}

	if gitOpsConfig != nil {
		return gitOpsConfig.IsConnected, gitOpsConfig.Provider
	}
	return false, ""
}

func getSnapshotReport(kotsStore store.Store, bsl *velerov1.BackupStorageLocation, appID string, clusterID string) (*SnapshotReport, error) {
	report := &SnapshotReport{}

	if bsl == nil {
		return nil, errors.New("no backup store location found")
	}
	report.Provider = bsl.Spec.Provider

	clusters, err := kotsStore.ListClusters()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list clusters")
	}
	var downstream *downstreamtypes.Downstream
	for _, cluster := range clusters {
		if cluster.ClusterID == clusterID {
			downstream = cluster
			break
		}
	}
	if downstream == nil {
		return nil, fmt.Errorf("cluster %s not found", clusterID)
	}
	report.FullSchedule = downstream.SnapshotSchedule
	report.FullTTL = downstream.SnapshotTTL

	app, err := kotsStore.GetApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app")
	}
	report.PartialSchedule = app.SnapshotSchedule
	report.PartialTTL = app.SnapshotTTL

	return report, nil
}
