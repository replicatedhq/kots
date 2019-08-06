package helm

import (
	"time"

	"github.com/pkg/errors"
	"k8s.io/helm/pkg/helm"
)

const (
	// tillerHost = "tiller-deploy.kube-system.svc.cluster.local:44134"
	tillerHost = "127.0.0.1:44134"
)

type HelmApplication struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Version   int32  `json:"version"`

	FirstDeployedAt time.Time `json:"firstDeployedAt"`
	LastDeployedAt  time.Time `json:"lastDeployedAt"`
	IsDeleted       bool      `json:"isDeleted"`

	ChartVersion string   `json:"chartVersion"`
	AppVersion   string   `json:"appVersion"`
	Sources      []string `json:"sources"`

	Values map[string]interface{} `json:"values"`
}

func isHelmInstalled() bool {
	helmClient := helm.NewClient(helm.Host(tillerHost))

	err := helmClient.PingTiller()
	if err != nil {
		return false
	}

	return true
}

func listHelmApplications() ([]*HelmApplication, error) {
	helmClient := helm.NewClient(helm.Host(tillerHost))

	releases, err := helmClient.ListReleases()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list releases from helm")
	}

	helmApplications := make([]*HelmApplication, 0, 0)
	for _, release := range releases.GetReleases() {
		helmApplication := HelmApplication{
			Name:      release.GetName(),
			Namespace: release.GetNamespace(),
			Version:   release.GetVersion(),
		}

		helmApplication.FirstDeployedAt = time.Unix(release.GetInfo().GetFirstDeployed().GetSeconds(), 0)
		helmApplication.LastDeployedAt = time.Unix(release.GetInfo().GetLastDeployed().GetSeconds(), 0)
		helmApplication.IsDeleted = release.GetInfo().GetDeleted() != nil

		helmApplication.ChartVersion = release.GetChart().GetMetadata().GetVersion()
		helmApplication.AppVersion = release.GetChart().GetMetadata().GetAppVersion()
		helmApplication.Sources = release.GetChart().GetMetadata().GetSources()

		helmApplication.Values = map[string]interface{}{}
		for key, value := range release.GetConfig().GetValues() {
			helmApplication.Values[key] = value.GetValue()
		}

		helmApplications = append(helmApplications, &helmApplication)
	}

	return helmApplications, nil
}
