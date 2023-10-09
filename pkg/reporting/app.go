package reporting

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/gitops"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
	"github.com/segmentio/ksuid"
	helmrelease "helm.sh/helm/v3/pkg/release"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

var (
	clusterID string // set when in Helm managed mode
)

func Init() error {
	if util.IsHelmManaged() {
		err := initFromHelm()
		if err != nil {
			return errors.Wrap(err, "failed to init from Helm")
		}
	} else {
		err := initFromDownstream()
		if err != nil {
			return errors.Wrap(err, "failed to init from downstream")
		}
	}

	if kotsadm.IsAirgap() {
		reporter = &AirgapReporter{}
	} else {
		reporter = &OnlineReporter{}
	}

	return nil
}

func initFromHelm() error {
	// ClusterID in reporting will be the UID of the v1 of Admin Console secret
	clientSet, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}

	selectorLabels := map[string]string{
		"owner":   "helm",
		"version": "1",
	}
	listOpts := metav1.ListOptions{
		LabelSelector: labels.SelectorFromSet(selectorLabels).String(),
	}

	secrets, err := clientSet.CoreV1().Secrets(util.PodNamespace).List(context.TODO(), listOpts)
	if err != nil {
		return errors.Wrap(err, "failed to list secrets")
	}

	for _, secret := range secrets.Items {
		helmRelease, err := helmReleaseFromSecretData(secret.Data["release"])
		if err != nil {
			logger.Warnf("failed to parse helm chart in secret %s: %v", &secret.ObjectMeta.Name, err)
			continue
		}

		if helmRelease.Chart == nil || helmRelease.Chart.Metadata == nil {
			continue
		}
		if helmRelease.Chart.Metadata.Name != "admin-console" {
			continue
		}

		clusterID = string(secret.ObjectMeta.UID)
		return nil
	}

	return errors.New("admin-console secret v1 not found")
}

func helmReleaseFromSecretData(data []byte) (*helmrelease.Release, error) {
	base64Reader := base64.NewDecoder(base64.StdEncoding, bytes.NewReader(data))
	gzreader, err := gzip.NewReader(base64Reader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gzip reader")
	}
	defer gzreader.Close()

	releaseData, err := ioutil.ReadAll(gzreader)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read from gzip reader")
	}

	release := &helmrelease.Release{}
	err = json.Unmarshal(releaseData, &release)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal release data")
	}

	return release, nil
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
		InstanceID:    appID,
		KOTSInstallID: os.Getenv("KOTS_INSTALL_ID"),
		KURLInstallID: os.Getenv("KURL_INSTALL_ID"),
		KOTSVersion:   buildversion.Version(),
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Debugf(errors.Wrap(err, "failed to get kubernetes clientset").Error())
	}

	if util.IsHelmManaged() {
		r.ClusterID = clusterID
	} else {
		r.ClusterID = k8sutil.GetKotsadmID(clientset)

		di, err := getDownstreamInfo(appID)
		if err != nil {
			logger.Debugf("failed to get downstream info: %v", err.Error())
		}
		if di != nil {
			r.Downstream = *di
		}
	}

	// get kubernetes cluster version
	k8sVersion, err := k8sutil.GetK8sVersion(clientset)
	if err != nil {
		logger.Debugf("failed to get k8s version: %v", err.Error())
	} else {
		r.K8sVersion = k8sVersion
	}

	if distribution := GetDistribution(clientset); distribution != UnknownDistribution {
		r.K8sDistribution = distribution.String()
	}

	// get app status
	if util.IsHelmManaged() {
		logger.Infof("TODO: get app status in Helm managed mode")
	} else {
		appStatus, err := store.GetStore().GetAppStatus(appID)
		if err != nil {
			logger.Debugf("failed to get app status: %v", err.Error())
		} else {
			r.AppStatus = string(appStatus.State)
		}
	}

	r.IsKurl, err = kurl.IsKurl(clientset)
	if err != nil {
		logger.Debugf(errors.Wrap(err, "failed to check if cluster is kurl").Error())
	}

	if r.IsKurl && clientset != nil {
		kurlNodes, err := kurl.GetNodes(clientset)
		if err != nil {
			logger.Debugf(errors.Wrap(err, "failed to get kurl nodes").Error())
		}

		for _, kurlNode := range kurlNodes.Nodes {
			r.KurlNodeCountTotal++
			if kurlNode.IsConnected && kurlNode.IsReady {
				r.KurlNodeCountReady++
			}
		}
	}

	r.IsGitOpsEnabled, r.GitOpsProvider = getGitOpsReport(clientset, appID, r.ClusterID)
	return &r
}

func getDownstreamInfo(appID string) (*types.DownstreamInfo, error) {
	di := types.DownstreamInfo{}

	downstreams, err := store.GetStore().ListDownstreamsForApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list downstreams for app")
	}
	if len(downstreams) == 0 {
		return nil, errors.New("no downstreams found for app")
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

		deployedKotsKinds, err := kotsutil.LoadKotsKindsFromPath(filepath.Join(deployedArchiveDir, "upstream"))
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
			logger.Debugf("failed to unmarshal preflight results: %v", err.Error())
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
		logger.Debugf("failed to get gitops config: %v", err.Error())
		return false, ""
	}

	if gitOpsConfig != nil {
		return gitOpsConfig.IsConnected, gitOpsConfig.Provider
	}
	return false, ""
}
