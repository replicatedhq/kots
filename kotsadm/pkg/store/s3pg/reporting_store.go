package s3pg

import (
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/k8s"
	"github.com/replicatedhq/kots/kotsadm/pkg/kurl"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
)

type downstreamInfo struct {
	downstreamCursor      string
	downstreamChannelID   string
	downstreamChannelName string
}

func (s S3PGStore) GetReportingInfo(appID string) *upstreamtypes.ReportingInfo {
	r := upstreamtypes.ReportingInfo{
		InstanceID: appID,
	}

	clusterID, err := s.getClusterID()
	if err != nil {
		logger.Error(errors.Wrap(err, "failed to get cluster id"))
	}
	r.ClusterID = clusterID

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
