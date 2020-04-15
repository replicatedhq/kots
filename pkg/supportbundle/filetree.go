package supportbundle

import "github.com/replicatedhq/kotsadm/pkg/supportbundle/types"

func archiveToFileTree(archivePath string) (*types.FileTree, error) {
	ft := types.FileTree{
		Nodes: []types.FileTreeNode{
			{
				Path: "test",
				Name: "test",
			},
		},
	}
	return &ft, nil
}
