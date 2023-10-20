package reporting

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/util"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	InstanceReportSecretNameFormat = "kotsadm-%s-instance-report"
	InstanceReportSecretKey        = "report"
	InstanceReportEventLimit       = 4000
)

var instanceReportMtx = sync.Mutex{}

func (r *AirgapReporter) SubmitAppInfo(appID string) error {
	a, err := r.store.GetApp(appID)
	if err != nil {
		if r.store.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "failed to get airgapped app")
	}

	license, err := r.store.GetLatestLicenseForApp(a.ID)
	if err != nil {
		return errors.Wrap(err, "failed to get license for airgapped app")
	}
	reportingInfo := GetReportingInfo(appID)

	event := GetInstanceReportEvent(license.Spec.LicenseID, reportingInfo)

	if err := CreateInstanceReportEvent(r.clientset, util.PodNamespace, a.Slug, event); err != nil {
		return errors.Wrap(err, "failed to create instance report event")
	}

	return nil
}

func GetInstanceReportEvent(licenseID string, reportingInfo *types.ReportingInfo) InstanceReportEvent {
	// not using the "cursor" packages because it doesn't provide access to the underlying int64
	downstreamSequence, err := strconv.ParseUint(reportingInfo.Downstream.Cursor, 10, 64)
	if err != nil {
		logger.Debugf("failed to parse downstream cursor %q: %v", reportingInfo.Downstream.Cursor, err)
	}

	return InstanceReportEvent{
		ReportedAt:                time.Now().UTC().UnixMilli(),
		LicenseID:                 licenseID,
		InstanceID:                reportingInfo.InstanceID,
		ClusterID:                 reportingInfo.ClusterID,
		AppStatus:                 reportingInfo.AppStatus,
		IsKurl:                    reportingInfo.IsKurl,
		KurlNodeCountTotal:        reportingInfo.KurlNodeCountTotal,
		KurlNodeCountReady:        reportingInfo.KurlNodeCountReady,
		K8sVersion:                reportingInfo.K8sVersion,
		K8sDistribution:           reportingInfo.K8sDistribution,
		KotsVersion:               reportingInfo.KOTSVersion,
		KotsInstallID:             reportingInfo.KOTSInstallID,
		KurlInstallID:             reportingInfo.KURLInstallID,
		IsGitOpsEnabled:           reportingInfo.IsGitOpsEnabled,
		GitOpsProvider:            reportingInfo.GitOpsProvider,
		DownstreamChannelID:       reportingInfo.Downstream.ChannelID,
		DownstreamChannelSequence: downstreamSequence,
		DownstreamChannelName:     reportingInfo.Downstream.ChannelName,
		DownstreamSequence:        reportingInfo.Downstream.Sequence,
		DownstreamSource:          reportingInfo.Downstream.Source,
		InstallStatus:             reportingInfo.Downstream.Status,
		PreflightState:            reportingInfo.Downstream.PreflightState,
		SkipPreflights:            reportingInfo.Downstream.SkipPreflights,
		ReplHelmInstalls:          reportingInfo.Downstream.ReplHelmInstalls,
		NativeHelmInstalls:        reportingInfo.Downstream.NativeHelmInstalls,
	}
}

func CreateInstanceReportEvent(clientset kubernetes.Interface, namespace string, appSlug string, event InstanceReportEvent) error {
	instanceReportMtx.Lock()
	defer instanceReportMtx.Unlock()

	existingSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf(InstanceReportSecretNameFormat, appSlug), metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get airgap instance report secret")
	} else if kuberneteserrors.IsNotFound(err) {
		instanceReport := &InstanceReport{
			Events: []InstanceReportEvent{event},
		}
		data, err := EncodeAirgapReport(instanceReport)
		if err != nil {
			return errors.Wrap(err, "failed to encode instance report")
		}

		uid, err := k8sutil.GetKotsadmDeploymentUID(clientset, namespace)
		if err != nil {
			return errors.Wrap(err, "failed to get kotsadm deployment uid")
		}

		secret := AirgapReportSecret(fmt.Sprintf(InstanceReportSecretNameFormat, appSlug), namespace, uid, data)

		_, err = clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create airgap instance report secret")
		}

		return nil
	}

	if existingSecret.Data == nil {
		existingSecret.Data = map[string][]byte{}
	}

	existingInstanceReport := &InstanceReport{}
	if existingSecret.Data[InstanceReportSecretKey] != nil {
		if err := DecodeAirgapReport(existingSecret.Data[InstanceReportSecretKey], existingInstanceReport); err != nil {
			return errors.Wrap(err, "failed to load existing instance report")
		}
	}

	existingInstanceReport.Events = append(existingInstanceReport.Events, event)
	if len(existingInstanceReport.Events) > InstanceReportEventLimit {
		existingInstanceReport.Events = existingInstanceReport.Events[len(existingInstanceReport.Events)-InstanceReportEventLimit:]
	}

	data, err := EncodeAirgapReport(existingInstanceReport)
	if err != nil {
		return errors.Wrap(err, "failed to encode existing instance report")
	}

	existingSecret.Data[InstanceReportSecretKey] = data

	_, err = clientset.CoreV1().Secrets(namespace).Update(context.TODO(), existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update airgap instance report secret")
	}

	return nil
}
