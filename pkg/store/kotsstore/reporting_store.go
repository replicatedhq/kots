package kotsstore

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8s"
	"github.com/replicatedhq/kots/pkg/kotsadm/types"
	downstream "github.com/replicatedhq/kots/pkg/kotsadmdownstream"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/logger"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/segmentio/ksuid"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type downstreamInfo struct {
	downstreamCursor      string
	downstreamChannelID   string
	downstreamChannelName string
}

var (
	kotsadmIDConfigMapName = "kotsadm-id"
)

func (s KOTSStore) GetReportingInfo(appID string) *upstreamtypes.ReportingInfo {
	r := upstreamtypes.ReportingInfo{
		InstanceID: appID,
	}
	configMap, err := getKotsadmIDConfigMap()
	if err == nil && configMap != nil {
		r.ClusterID = configMap.Data["id"]
	} else if err == nil && configMap == nil {
		//generate guid and use that as clusterId to identify that as a different install
		r.ClusterID = ksuid.New().String()
		CreateKotsadmIDConfigMap(r.ClusterID)
	} else {
		r.ClusterID = ksuid.New().String()
	}

	di, err := s.getDownstreamInfo(appID)
	if err != nil {
		logger.Infof("failed to get downstream info: %v", err.Error())
	}
	if di != nil {
		r.DownstreamCursor = di.downstreamCursor
		r.DownstreamChannelID = di.downstreamChannelID
		r.DownstreamChannelName = di.downstreamChannelName
	}

	// get kubernetes cluster version
	k8sVersion, err := s.getK8sVersion()
	if err != nil {
		logger.Infof("failed to get k8s version: %v", err.Error())
	} else {
		r.K8sVersion = k8sVersion
	}

	// get app status
	appStatus, err := s.GetAppStatus(appID)
	if err != nil {
		logger.Infof("failed to get app status: %v", err.Error())
	} else {
		r.AppStatus = string(appStatus.State)
	}

	// check if embedded cluster
	r.IsKurl = kurl.IsKurl()

	return &r
}

func (s KOTSStore) getClusterID() (string, error) {
	clusters, err := s.ListClusters()
	if err != nil {
		return "", errors.Wrap(err, "failed to list clusters")
	}
	if len(clusters) == 0 {
		return "", errors.New("no clusters found")
	}
	return clusters[0].ClusterID, nil
}

func (s KOTSStore) getDownstreamInfo(appID string) (*downstreamInfo, error) {
	di := downstreamInfo{}

	downstreams, err := s.ListDownstreamsForApp(appID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list downstreams for app")
	}
	if len(downstreams) == 0 {
		return nil, errors.New("no downstreams found for app")
	}

	deployedAppSequence, err := downstream.GetCurrentParentSequence(appID, downstreams[0].ClusterID)
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

		err = s.GetAppVersionArchive(appID, deployedAppSequence, deployedArchiveDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get app version archive")
		}

		deployedKotsKinds, err := kotsutil.LoadKotsKindsFromPath(deployedArchiveDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load kotskinds from path")
		}

		di.downstreamCursor = deployedKotsKinds.Installation.Spec.UpdateCursor
		di.downstreamChannelID = deployedKotsKinds.Installation.Spec.ChannelID
		di.downstreamChannelName = deployedKotsKinds.Installation.Spec.ChannelName
	}

	return &di, nil
}

func (s KOTSStore) getK8sVersion() (string, error) {
	clientset, err := k8s.Clientset()
	if err != nil {
		return "", errors.Wrap(err, "failed to create kubernetes clientset")
	}
	k8sVersion, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return "", errors.Wrap(err, "failed to get kubernetes server version")
	}
	return k8sVersion.GitVersion, nil
}

func getKotsadmIDConfigMap() (*corev1.ConfigMap, error) {
	clientset, err := k8s.Clientset()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}
	namespace := os.Getenv("POD_NAMESPACE")
	existingConfigmap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), kotsadmIDConfigMapName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed to get configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		return nil, nil
	}
	return existingConfigmap, nil
}

// CreateKotsadmIDConfigMap creates an id for an kotsadm instance and stores in configmap
func CreateKotsadmIDConfigMap(kotsadmID string) error {
	var err error = nil
	clientset, err := k8s.Clientset()
	if err != nil {
		return err
	}
	configmap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      kotsadmIDConfigMapName,
			Namespace: os.Getenv("POD_NAMESPACE"),
			Labels: map[string]string{
				types.KotsadmKey: types.KotsadmLabelValue,
				types.ExcludeKey: types.ExcludeValue,
			},
		},
		Data: map[string]string{"id": kotsadmID},
	}
	_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Create(context.TODO(), &configmap, metav1.CreateOptions{})
	return err
}

// IsKotsadmIDConfigMapPresent checks if the configmap for kotsadm-id exists
func IsKotsadmIDConfigMapPresent() (bool, error) {
	clientset, err := k8s.Clientset()
	if err != nil {
		return false, errors.Wrap(err, "failed to get clientset")
	}
	namespace := os.Getenv("POD_NAMESPACE")
	_, err = clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), kotsadmIDConfigMapName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return false, errors.Wrap(err, "failed to get configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		return false, nil
	}
	return true, nil
}

// UpdateKotsadmIDConfigMap creates an id for an kotsadm instance and stores in configmap
func UpdateKotsadmIDConfigMap(kotsadmID string) error {
	clientset, err := k8s.Clientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}
	namespace := os.Getenv("POD_NAMESPACE")
	existingConfigMap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), kotsadmIDConfigMapName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		return nil
	}
	if existingConfigMap.Data == nil {
		existingConfigMap.Data = map[string]string{}
	}
	existingConfigMap.Data["id"] = kotsadmID

	_, err = clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Update(context.Background(), existingConfigMap, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update config map")
	}
	return nil
}
