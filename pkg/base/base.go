package base

type Base struct {
	Files []BaseFile
}

type BaseFile struct {
	Path    string
	Content []byte
}
