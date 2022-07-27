package reporting

import (
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/helm"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kurl"
	kurltypes "github.com/replicatedhq/kots/pkg/kurl/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	"github.com/segmentio/ksuid"
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
		helmRelease, err := helm.HelmReleaseFromSecretData(secret.Data["release"])
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

	if isKotsadmIDGenerated && !cmpExists {
		kotsadmID := ksuid.New().String()
		err = k8sutil.CreateKotsadmIDConfigMap(kotsadmID)
	} else if !isKotsadmIDGenerated && !cmpExists {
		err = k8sutil.CreateKotsadmIDConfigMap(clusterID)
	} else if !isKotsadmIDGenerated && cmpExists {
		err = k8sutil.UpdateKotsadmIDConfigMap(clusterID)
	} else {
		// id exists and so as configmap, noop
	}
	if err == nil {
		err = store.GetStore().SetIsKotsadmIDGenerated()
	}

	return err
}

func SendAppInfo(appID string) error {
	a, err := store.GetStore().GetApp(appID)
	if err != nil {
		if store.GetStore().IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "failed to get license for app")
	}

	license, err := store.GetStore().GetLatestLicenseForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get license for app")
	}

	endpoint := license.Spec.Endpoint
	if !canReport(endpoint) {
		return nil
	}

	url := fmt.Sprintf("%s/kots_metrics/license_instance/info", endpoint)

	postReq, err := util.NewRequest("POST", url, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create http request")
	}
	postReq.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", license.Spec.LicenseID, license.Spec.LicenseID)))))
	postReq.Header.Set("Content-Type", "application/json")

	reportingInfo := GetReportingInfo(a.ID)
	InjectReportingInfoHeaders(postReq, reportingInfo)

	resp, err := http.DefaultClient.Do(postReq)
	if err != nil {
		return errors.Wrap(err, "failed to post request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	return nil
}

func GetReportingInfo(appID string) *types.ReportingInfo {
	ctx := context.TODO()

	r := types.ReportingInfo{
		InstanceID: appID,
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		logger.Debugf(errors.Wrap(err, "failed to get kubernetes clientset").Error())
	}

	if util.IsHelmManaged() {
		r.ClusterID = clusterID
	} else {
		configMap, err := k8sutil.GetKotsadmIDConfigMap()
		if err != nil {
			r.ClusterID = ksuid.New().String()
		} else if configMap != nil {
			r.ClusterID = configMap.Data["id"]
		} else {
			// configmap is missing for some reason, recreate with new guid, this will appear as a new instance in the report
			r.ClusterID = ksuid.New().String()
			k8sutil.CreateKotsadmIDConfigMap(r.ClusterID)
		}

		di, err := getDownstreamInfo(appID)
		if err != nil {
			logger.Debugf("failed to get downstream info: %v", err.Error())
		}
		if di != nil {
			r.Downstream = *di
		}
	}

	// get kubernetes cluster version
	k8sVersion, err := k8sutil.GetK8sVersion()
	if err != nil {
		logger.Debugf("failed to get k8s version: %v", err.Error())
	} else {
		r.K8sVersion = k8sVersion
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

	r.IsKurl, err = kurl.IsKurl()
	if err != nil {
		logger.Debugf(errors.Wrap(err, "failed to check if cluster is kurl").Error())
	}

	if r.IsKurl && clientset != nil {
		kurlNodes, err := cachedKurlGetNodes(ctx, clientset)
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

	return &r
}

var (
	cachedKurlNodes               *kurltypes.KurlNodes
	cachedKurlNodesLastUpdateTime time.Time
	cachedKurlNodesMu             sync.Mutex
)

const (
	cachedKurlNodesUpdateInterval = 5 * time.Minute
)

func cachedKurlGetNodes(ctx context.Context, clientset kubernetes.Interface) (*kurltypes.KurlNodes, error) {
	cachedKurlNodesMu.Lock()
	defer cachedKurlNodesMu.Unlock()

	if cachedKurlNodes != nil && time.Now().Sub(cachedKurlNodesLastUpdateTime) < cachedKurlNodesUpdateInterval {
		return cachedKurlNodes, nil
	}

	kurlNodes, err := kurl.GetNodes(clientset)
	if err != nil {
		return nil, err
	}

	cachedKurlNodes = kurlNodes
	cachedKurlNodesLastUpdateTime = time.Now()
	return cachedKurlNodes, nil
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

	deployedAppSequence, err := store.GetStore().GetCurrentParentSequence(appID, downstreams[0].ClusterID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current downstream parent sequence")
	}

	// info about the deployed app sequence
	if deployedAppSequence != -1 {
		deployedArchiveDir, err := ioutil.TempDir("", "kotsadm")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create temp dir")
		}
		defer os.RemoveAll(deployedArchiveDir)

		err = store.GetStore().GetAppVersionArchive(appID, deployedAppSequence, deployedArchiveDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get app version archive")
		}

		deployedKotsKinds, err := kotsutil.LoadKotsKindsFromPath(deployedArchiveDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load kotskinds from path")
		}

		di.Cursor = deployedKotsKinds.Installation.Spec.UpdateCursor
		di.ChannelID = deployedKotsKinds.Installation.Spec.ChannelID
		di.ChannelName = deployedKotsKinds.Installation.Spec.ChannelName

		if len(deployedKotsKinds.HelmCharts) > 0 {
			for _, chart := range deployedKotsKinds.HelmCharts {
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
