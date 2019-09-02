package upstream

type UpstreamFile struct {
	Path    string
	Content []byte
}

type Upstream struct {
	URI          string
	Name         string
	Type         string
	Files        []UpstreamFile
	UpdateCursor string
	VersionLabel string
}
