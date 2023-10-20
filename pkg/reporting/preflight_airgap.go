package reporting

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/buildversion"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	PreflightReportSecretNameFormat = "kotsadm-%s-preflight-report"
	PreflightReportSecretKey        = "report"
	PreflightReportEventLimit       = 4000
)

var preflightReportMtx = sync.Mutex{}

func (r *AirgapReporter) SubmitPreflightData(license *kotsv1beta1.License, appID string, clusterID string, sequence int64, skipPreflights bool, installStatus storetypes.DownstreamVersionStatus, isCLI bool, preflightStatus string, appStatus string) error {
	app, err := r.store.GetApp(appID)
	if err != nil {
		if r.store.IsNotFound(err) {
			return nil
		}
		return errors.Wrap(err, "failed to get airgapped app")
	}

	event := PreflightReportEvent{
		ReportedAt:      time.Now().UTC().UnixMilli(),
		LicenseID:       license.Spec.LicenseID,
		InstanceID:      appID,
		ClusterID:       clusterID,
		Sequence:        sequence,
		SkipPreflights:  skipPreflights,
		InstallStatus:   string(installStatus),
		IsCLI:           isCLI,
		PreflightStatus: preflightStatus,
		AppStatus:       appStatus,
		KotsVersion:     buildversion.Version(),
	}

	if err := CreatePreflightReportEvent(r.clientset, util.PodNamespace, app.Slug, event); err != nil {
		return errors.Wrap(err, "failed to create preflight report event")
	}

	return nil
}

func CreatePreflightReportEvent(clientset kubernetes.Interface, namespace string, appSlug string, event PreflightReportEvent) error {
	preflightReportMtx.Lock()
	defer preflightReportMtx.Unlock()

	existingSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), fmt.Sprintf(PreflightReportSecretNameFormat, appSlug), metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get airgap preflight report secret")
	} else if kuberneteserrors.IsNotFound(err) {
		preflightReport := &PreflightReport{
			Events: []PreflightReportEvent{event},
		}
		data, err := EncodeAirgapReport(preflightReport)
		if err != nil {
			return errors.Wrap(err, "failed to encode preflight report")
		}

		uid, err := k8sutil.GetKotsadmDeploymentUID(clientset, namespace)
		if err != nil {
			return errors.Wrap(err, "failed to get kotsadm deployment uid")
		}

		secret := AirgapReportSecret(fmt.Sprintf(PreflightReportSecretNameFormat, appSlug), namespace, uid, data)

		_, err = clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create airgap preflight report secret")
		}

		return nil
	}

	if existingSecret.Data == nil {
		existingSecret.Data = map[string][]byte{}
	}

	existingPreflightReport := &PreflightReport{}
	if existingSecret.Data[PreflightReportSecretKey] != nil {
		if err := DecodeAirgapReport(existingSecret.Data[PreflightReportSecretKey], existingPreflightReport); err != nil {
			return errors.Wrap(err, "failed to load existing preflight report")
		}
	}

	existingPreflightReport.Events = append(existingPreflightReport.Events, event)
	if len(existingPreflightReport.Events) > PreflightReportEventLimit {
		existingPreflightReport.Events = existingPreflightReport.Events[len(existingPreflightReport.Events)-PreflightReportEventLimit:]
	}

	data, err := EncodeAirgapReport(existingPreflightReport)
	if err != nil {
		return errors.Wrap(err, "failed to encode existing preflight report")
	}

	existingSecret.Data[PreflightReportSecretKey] = data

	_, err = clientset.CoreV1().Secrets(namespace).Update(context.TODO(), existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update airgap preflight report secret")
	}

	return nil
}
