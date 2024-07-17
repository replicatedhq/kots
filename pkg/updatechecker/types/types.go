package types

type CheckForUpdatesOpts struct {
	AppID                  string
	DeployLatest           bool
	DeployVersionLabel     string
	IsAutomatic            bool
	SkipPreflights         bool
	SkipCompatibilityCheck bool
	IsCLI                  bool
	Wait                   bool
}

type UpdateCheckResponse struct {
	AvailableUpdates  int64
	CurrentRelease    *UpdateCheckRelease
	AvailableReleases []UpdateCheckRelease
	DeployingRelease  *UpdateCheckRelease
}

type UpdateCheckRelease struct {
	Sequence int64
	Version  string
}
