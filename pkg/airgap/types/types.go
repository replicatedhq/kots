package types

type PendingApp struct {
	ID                string
	Slug              string
	Name              string
	LicenseData       string
	SelectedChannelID string
}

type InstallStatus struct {
	InstallStatus  string `json:"installStatus"`
	CurrentMessage string `json:"currentMessage"`
}

func (a *PendingApp) GetID() string {
	return a.ID
}

func (a *PendingApp) GetSlug() string {
	return a.Slug
}

func (a *PendingApp) GetCurrentSequence() int64 {
	return 0
}

func (a *PendingApp) GetIsAirgap() bool {
	return true
}

func (a *PendingApp) GetNamespace() string {
	return ""
}
