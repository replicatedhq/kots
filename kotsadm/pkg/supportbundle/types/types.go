package types

type SupportBundle struct {
	ID string `json:"id"`
}

type FileTree struct {
	Nodes []FileTreeNode `json:",inline"`
}

type FileTreeNode struct {
	Name     string         `json:"name"`
	Path     string         `json:"path"`
	Children []FileTreeNode `json:"children,omitempty"`
}
