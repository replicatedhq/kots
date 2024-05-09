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
	OKE
	OpenShift
	RKE2
	Tanzu
	EmbeddedCluster
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
	case OKE:
		return "oke"
	case OpenShift:
		return "openshift"
	case RKE2:
		return "rke2"
	case Tanzu:
		return "tanzu"
	case EmbeddedCluster:
		return "embedded-cluster"
	}
	return "unknown"
}
