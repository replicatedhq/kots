package reporting

import (
	"github.com/replicatedhq/kots/pkg/store"
	storetypes "github.com/replicatedhq/kots/pkg/store/types"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
	"k8s.io/client-go/kubernetes"
)

type Distribution int64

const (
	UnknownDistribution Distribution = iota
	AKS
	DigitalOcean
	EKS
	GKE
	GKEAutoPilot
	K0s
	K3s
	Kind
	Kurl
	MicroK8s
	Minikube
	OpenShift
	RKE2
	Tanzu
)

type Reporter interface {
	SubmitAppInfo(appID string) error
	SubmitPreflightData(license *kotsv1beta1.License, appID string, clusterID string, sequence int64, skipPreflights bool, installStatus storetypes.DownstreamVersionStatus, isCLI bool, preflightStatus string, appStatus string) error
}

var reporter Reporter

type AirgapReporter struct {
	clientset kubernetes.Interface
	store     store.Store
}

var _ Reporter = &AirgapReporter{}

type OnlineReporter struct {
}

var _ Reporter = &OnlineReporter{}

func (d Distribution) String() string {
	switch d {
	case AKS:
		return "aks"
	case DigitalOcean:
		return "digital-ocean"
	case EKS:
		return "eks"
	case GKE:
		return "gke"
	case GKEAutoPilot:
		return "gke-autopilot"
	case K0s:
		return "k0s"
	case K3s:
		return "k3s"
	case Kind:
		return "kind"
	case Kurl:
		return "kurl"
	case MicroK8s:
		return "microk8s"
	case Minikube:
		return "minikube"
	case OpenShift:
		return "openshift"
	case RKE2:
		return "rke2"
	case Tanzu:
		return "tanzu"
	}
	return "unknown"
}

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
