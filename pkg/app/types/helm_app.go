package types

import (
	"time"

	helmrelease "helm.sh/helm/v3/pkg/release"
)

type HelmApp struct {
	Release           helmrelease.Release
	Labels            map[string]string
	Version           int64 // populated from labels
	Namespace         string
	IsConfigurable    bool
	ChartPath         string
	CreationTimestamp time.Time
	PathToValuesFile  string
}

func (a *HelmApp) GetID() string {
	return a.Release.Name
}

func (a *HelmApp) GetSlug() string {
	return a.Release.Name
}

func (a *HelmApp) GetCurrentSequence() int64 {
	return a.Version
}

func (a *HelmApp) GetIsAirgap() bool {
	return false // no airgap support yet
}
