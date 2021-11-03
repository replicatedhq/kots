package types

type ReportingInfo struct {
	InstanceID string
	ClusterID  string
	Downstream DownstreamInfo
	AppStatus  string
	IsKurl     bool
	K8sVersion string
}

type DownstreamInfo struct {
	Cursor             string
	ChannelID          string
	ChannelName        string
	ReplHelmInstalls   int
	NativeHelmInstalls int
}
