package types

type PendingApp struct {
	ID          string
	Slug        string
	Name        string
	LicenseData string
}

type InstallStatus struct {
	InstallStatus  string `json:"installStatus"`
	CurrentMessage string `json:"currentMessage"`
}
