package upstream

type UpstreamFile struct {
	Path    string
	Content []byte
}

type Upstream struct {
	URI   string
	Type  string
	Files []UpstreamFile
}
