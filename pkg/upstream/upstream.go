package upstream

import (
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"k8s.io/client-go/kubernetes/scheme"
)

func init() {
	kotsscheme.AddToScheme(scheme.Scheme)
}

type UpstreamFile struct {
	Path    string
	Content []byte
}

type Upstream struct {
	URI           string
	Name          string
	Type          string
	Files         []UpstreamFile
	UpdateCursor  string
	VersionLabel  string
	ReleaseNotes  string
	EncryptionKey string
}
