package reporting

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/pkg/errors"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	ReportSecretNameFormat = "kotsadm-%s-report"
	ReportSecretKey        = "report"
	ReportEventLimit       = 4000
)

var (
	instanceReportMtx  = sync.Mutex{}
	preflightReportMtx = sync.Mutex{}
)

type Report struct {
	Events []ReportEvent `json:"events"`
}

type ReportEvent interface {
	GetReportSecretName(appSlug string) string
	GetReportSecretKey() string
	GetReportEventLimit() int
	GetReportMtx() *sync.Mutex
	GetReportType() string
}

var _ ReportEvent = &InstanceReportEvent{}
var _ ReportEvent = &PreflightReportEvent{}

type InstanceReport struct {
	Events []InstanceReportEvent `json:"events"`
}

type InstanceReportEvent struct {
	ReportedAt                int64  `json:"reported_at"`
	LicenseID                 string `json:"license_id"`
	InstanceID                string `json:"instance_id"`
	ClusterID                 string `json:"cluster_id"`
	AppStatus                 string `json:"app_status"`
	IsKurl                    bool   `json:"is_kurl"`
	KurlNodeCountTotal        int    `json:"kurl_node_count_total"`
	KurlNodeCountReady        int    `json:"kurl_node_count_ready"`
	K8sVersion                string `json:"k8s_version"`
	K8sDistribution           string `json:"k8s_distribution,omitempty"`
	KotsVersion               string `json:"kots_version"`
	KotsInstallID             string `json:"kots_install_id,omitempty"`
	KurlInstallID             string `json:"kurl_install_id,omitempty"`
	IsGitOpsEnabled           bool   `json:"is_gitops_enabled"`
	GitOpsProvider            string `json:"gitops_provider"`
	DownstreamChannelID       string `json:"downstream_channel_id,omitempty"`
	DownstreamChannelSequence uint64 `json:"downstream_channel_sequence,omitempty"`
	DownstreamChannelName     string `json:"downstream_channel_name,omitempty"`
	DownstreamSequence        *int64 `json:"downstream_sequence,omitempty"`
	DownstreamSource          string `json:"downstream_source,omitempty"`
	InstallStatus             string `json:"install_status,omitempty"`
	PreflightState            string `json:"preflight_state,omitempty"`
	SkipPreflights            bool   `json:"skip_preflights"`
	ReplHelmInstalls          int    `json:"repl_helm_installs"`
	NativeHelmInstalls        int    `json:"native_helm_installs"`
}

func (i *InstanceReportEvent) GetReportType() string {
	return "instance"
}

func (i *InstanceReportEvent) GetReportSecretName(appSlug string) string {
	return fmt.Sprintf(ReportSecretNameFormat, fmt.Sprintf("%s-%s", appSlug, i.GetReportType()))
}

func (i *InstanceReportEvent) GetReportSecretKey() string {
	return ReportSecretKey
}

func (i *InstanceReportEvent) GetReportEventLimit() int {
	return ReportEventLimit
}

func (i *InstanceReportEvent) GetReportMtx() *sync.Mutex {
	return &instanceReportMtx
}

type PreflightReport struct {
	Events []PreflightReportEvent `json:"events"`
}

type PreflightReportEvent struct {
	ReportedAt      int64  `json:"reported_at"`
	LicenseID       string `json:"license_id"`
	InstanceID      string `json:"instance_id"`
	ClusterID       string `json:"cluster_id"`
	Sequence        int64  `json:"sequence"`
	SkipPreflights  bool   `json:"skip_preflights"`
	InstallStatus   string `json:"install_status"`
	IsCLI           bool   `json:"is_cli"`
	PreflightStatus string `json:"preflight_status"`
	AppStatus       string `json:"app_status"`
	KotsVersion     string `json:"kots_version"`
}

func (p *PreflightReportEvent) GetReportType() string {
	return "preflight"
}

func (p *PreflightReportEvent) GetReportSecretName(appSlug string) string {
	return fmt.Sprintf(ReportSecretNameFormat, fmt.Sprintf("%s-%s", appSlug, p.GetReportType()))
}

func (p *PreflightReportEvent) GetReportSecretKey() string {
	return ReportSecretKey
}

func (p *PreflightReportEvent) GetReportEventLimit() int {
	return ReportEventLimit
}

func (p *PreflightReportEvent) GetReportMtx() *sync.Mutex {
	return &preflightReportMtx
}

func CreateReportEvent(clientset kubernetes.Interface, namespace string, appSlug string, event ReportEvent) error {
	event.GetReportMtx().Lock()
	defer event.GetReportMtx().Unlock()

	existingSecret, err := clientset.CoreV1().Secrets(namespace).Get(context.TODO(), event.GetReportSecretName(appSlug), metav1.GetOptions{})
	if err != nil && !kuberneteserrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get report secret")
	} else if kuberneteserrors.IsNotFound(err) {
		report := &Report{
			Events: []ReportEvent{event},
		}
		data, err := EncodeReport(report)
		if err != nil {
			return errors.Wrap(err, "failed to encode report")
		}

		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      event.GetReportSecretName(appSlug),
				Namespace: namespace,
				Labels:    kotsadmtypes.GetKotsadmLabels(),
			},
			Data: map[string][]byte{
				event.GetReportSecretKey(): data,
			},
		}

		_, err = clientset.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to create report secret")
		}

		return nil
	}

	if existingSecret.Data == nil {
		existingSecret.Data = map[string][]byte{}
	}

	existingReport := &Report{}
	if existingSecret.Data[event.GetReportSecretKey()] != nil {
		if err := DecodeReport(existingSecret.Data[event.GetReportSecretKey()], existingReport, event.GetReportType()); err != nil {
			return errors.Wrap(err, "failed to load existing report")
		}
	}

	existingReport.Events = append(existingReport.Events, event)
	if len(existingReport.Events) > event.GetReportEventLimit() {
		existingReport.Events = existingReport.Events[len(existingReport.Events)-event.GetReportEventLimit():]
	}

	data, err := EncodeReport(existingReport)
	if err != nil {
		return errors.Wrap(err, "failed to encode existing report")
	}

	existingSecret.Data[event.GetReportSecretKey()] = data

	_, err = clientset.CoreV1().Secrets(namespace).Update(context.TODO(), existingSecret, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update instance report secret")
	}

	return nil
}

func EncodeReport(r *Report) ([]byte, error) {
	data, err := json.Marshal(r)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal report")
	}
	compressedData, err := util.GzipData(data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to gzip report")
	}
	encodedData := base64.StdEncoding.EncodeToString(compressedData)

	return []byte(encodedData), nil
}

func DecodeReport(encodedData []byte, existingReport *Report, reportType string) error {
	decodedData, err := base64.StdEncoding.DecodeString(string(encodedData))
	if err != nil {
		return errors.Wrap(err, "failed to decode report")
	}
	decompressedData, err := util.GunzipData(decodedData)
	if err != nil {
		return errors.Wrap(err, "failed to gunzip report")
	}

	switch reportType {
	case "instance":
		r := &InstanceReport{}
		if err := json.Unmarshal(decompressedData, r); err != nil {
			return errors.Wrap(err, "failed to unmarshal report")
		}
		for _, event := range r.Events {
			existingReport.Events = append(existingReport.Events, &event)
		}
	case "preflight":
		r := &PreflightReport{}
		if err := json.Unmarshal(decompressedData, r); err != nil {
			return errors.Wrap(err, "failed to unmarshal report")
		}
		for _, event := range r.Events {
			existingReport.Events = append(existingReport.Events, &event)
		}
	default:
		return errors.Errorf("unknown report type %q", reportType)
	}

	return nil
}
