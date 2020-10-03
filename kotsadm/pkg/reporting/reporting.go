package reporting

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/k8s"
	"github.com/replicatedhq/kots/kotsadm/pkg/kurl"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/store"
	"github.com/replicatedhq/kots/pkg/kotsutil"

	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
)

type DownstreamInfo struct {
	DownstreamCursor      string
	DownstreamChannelID   string
	DownstreamChannelName string
}

func GetReportingInfo(appID string) *upstreamtypes.ReportingInfo {
	r := upstreamtypes.ReportingInfo{
		InstanceID: appID,
	}

	clusterID, err := getClusterID()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get cluster id"))
	}
	r.ClusterID = clusterID

	di, err := getDownstreamInfo(appID)
	if err != nil {
		logger.Infof("failed to get downstream info: %v", err.Error())
	}
	if di != nil {
		r.DownstreamCursor = di.DownstreamCursor
		r.DownstreamChannelID = di.DownstreamChannelID
		r.DownstreamChannelName = di.DownstreamChannelName
	}

	// get kubernetes cluster version
	k8sVersion, err := getK8sVersion()
	if err != nil {
		logger.Infof("failed to get k8s version: %v", err.Error())
	} else {
		r.K8sVersion = k8sVersion
	}

	// get app status
	appStatus, err := store.GetStore().GetAppStatus(appID)
	if err != nil {
		logger.Infof("failed to get app status: %v", err.Error())
	} else {
		r.AppStatus = string(appStatus.State)
	}

	// check if embedded cluster
	r.IsKurl = kurl.IsKurl()

	return &r
}

func getClusterID() (string, error) {
	clusters, err := store.GetStore().ListClusters()
	if err != nil {
		return "", errors.Wrap(err, "failed to list clusters")
	}
	if len(clusters) == 0 {
		return "", errors.New("no clusters found")
	}
	return clusters[0].ClusterID, nil
}

func getDownstreamInfo(appID string) (*DownstreamInfo, error) {
	di := DownstreamInfo{}

	downstreams, err := store.GetStore().ListDownstreamsForApp(appID)
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
		deployedArchiveDir, err := store.GetStore().GetAppVersionArchive(appID, deployedAppSequence)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get app version archive")
		}

		deployedKotsKinds, err := kotsutil.LoadKotsKindsFromPath(deployedArchiveDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load kotskinds from path")
		}

		di.DownstreamCursor = deployedKotsKinds.Installation.Spec.UpdateCursor
		di.DownstreamChannelID = deployedKotsKinds.Installation.Spec.ChannelID
		di.DownstreamChannelName = deployedKotsKinds.Installation.Spec.ChannelName
	}

	return &di, nil
}

func getK8sVersion() (string, error) {
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
