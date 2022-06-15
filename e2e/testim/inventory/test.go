package inventory

type Test struct {
	Name            string
	Suite           string
	Label           string
	Namespace       string
	UpstreamURI     string
	UseMinimalRBAC  bool
	NeedsSnapshots  bool
	NeedsMonitoring bool
	NeedsRegistry   bool
}
