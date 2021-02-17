package s3pg

import (
	"context"
	"io/ioutil"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/k8s"
	"github.com/replicatedhq/kots/kotsadm/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/segmentio/ksuid"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
)

type downstreamInfo struct {
	downstreamCursor      string
	downstreamChannelID   string
	downstreamChannelName string
}

var (
	configMapName = "kotsadm-id"
)

func (s S3PGStore) GetReportingInfo(appID string) *upstreamtypes.ReportingInfo {
	r := upstreamtypes.ReportingInfo{
		InstanceID: appID,
	}

	clusterID, err := s.getClusterID()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get cluster id"))
	}
	r.ClusterID = clusterID
	configMap, err := getAdminIDConfigMap()
	if err == nil && configMap != nil {
		r.ClusterID = configMap.Data["id"]
	} else if err == nil && configMap == nil {
		//generate guid and use that as clusterId to identify that as a different install
		clusterID = ksuid.New().String()
		r.ClusterID = clusterID
		_, err = CreateAdminIDConfigMap(clusterID)
		if err != nil {
			logger.Errorf("Failed to to create config map %v", err)
		}
	} else {
		logger.Errorf("Config map check error %v", err)
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

func (s S3PGStore) getClusterID() (string, error) {
	clusters, err := s.ListClusters()
	if err != nil {
		return "", errors.Wrap(err, "failed to list clusters")
	}
	if len(clusters) == 0 {
		return "", errors.New("no clusters found")
	}
	return clusters[0].ClusterID, nil
}

func (s S3PGStore) getDownstreamInfo(appID string) (*downstreamInfo, error) {
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

func (s S3PGStore) getK8sVersion() (string, error) {
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

func getAdminIDConfigMap() (*corev1.ConfigMap, error) {

	cfg, err := k8sconfig.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kubernetes config")
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}
	namespace := os.Getenv("POD_NAMESPACE")
	existingConfigmap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return nil, errors.Wrap(err, "failed to get configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		return nil, nil
	}
	if existingConfigmap != nil {
		log.Println("Existing config map", existingConfigmap.Data["id"])
		return existingConfigmap, nil
	}
	return nil, nil

}

// CreateAdminIDConfigMap creates an id for an kotsadm instance and stores in configmap
func CreateAdminIDConfigMap(clusterID string) (*corev1.ConfigMap, error) {
	cfg, err := k8sconfig.GetConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kubernetes config")
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get clientset")
	}
	configmap := corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: os.Getenv("POD_NAMESPACE"),
			Labels: map[string]string{
				"kots.io/kotsadm": "true",
			},
		},
		Data: map[string]string{"id": clusterID},
	}

	createdConfigmap, err := clientset.CoreV1().ConfigMaps(os.Getenv("POD_NAMESPACE")).Create(context.TODO(), &configmap, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create configmap")
	}

	log.Println("Created Admin config map", createdConfigmap.Data["id"])
	return createdConfigmap, nil

}

// IsAdminIDConfigMapPresent checks if the configmap for kotsadm-id exists
func IsAdminIDConfigMapPresent() (bool, error) {

	cfg, err := k8sconfig.GetConfig()
	if err != nil {
		return false, errors.Wrap(err, "failed to get kubernetes config")
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return false, errors.Wrap(err, "failed to get clientset")
	}
	namespace := os.Getenv("POD_NAMESPACE")
	existingConfigmap, err := clientset.CoreV1().ConfigMaps(namespace).Get(context.TODO(), configMapName, metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return false, errors.Wrap(err, "failed to get configmap")
	} else if kuberneteserrors.IsNotFound(err) {
		return false, nil
	}
	if existingConfigmap != nil {
		log.Println("Existing config map", existingConfigmap.Data["id"])
		return true, nil
	}
	return false, nil

}
