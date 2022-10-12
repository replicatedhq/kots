package types

import (
	"time"

	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
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
	// TODO: This is values the user is editing on the Config screen. This is a temporary solution while we figure out the UX.
	TempConfigValues  map[string]kotsv1beta1.ConfigValue
	ConfigValuesSaved bool
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

func (a *HelmApp) GetNamespace() string {
	return a.Namespace
}
