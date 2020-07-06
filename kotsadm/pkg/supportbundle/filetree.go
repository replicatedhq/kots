package supportbundle

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archiver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/supportbundle/types"
)

func archiveToFileTree(archivePath string) (*types.FileTree, error) {
	workDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create workdir")
	}
	defer os.RemoveAll(workDir)

	// extract the current archive to this root
	tarGz := &archiver.TarGz{
		Tar: &archiver.Tar{
			ImplicitTopLevelFolder: false,
		},
	}
	if err := tarGz.Unarchive(archivePath, workDir); err != nil {
		return nil, errors.Wrap(err, "failed to unarchive")
	}

	directories := []string{}
	files := []string{}

	err = filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if len(path) <= len(workDir) {
			return nil
		}

		if info.IsDir() {
			directories = append(directories, path[len(workDir)+1:])
			return nil
		}

		files = append(files, path[len(workDir)+1:])
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to walk")
	}

	ft := types.FileTree{
		Nodes: []types.FileTreeNode{},
	}

	for _, directory := range directories {
		if !strings.Contains(directory, string(os.PathSeparator)) {
			node, err := getDirectoryNode(directory, directories, files)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get directory node")
			}

			ft.Nodes = append(ft.Nodes, *node)
		}
	}

	for _, file := range files {
		if !strings.Contains(file, string(os.PathSeparator)) {
			node := types.FileTreeNode{
				Name: file,
				Path: file,
			}

			ft.Nodes = append(ft.Nodes, node)
		}
	}

	return &ft, nil
}

func getDirectoryNode(directory string, directories []string, files []string) (*types.FileTreeNode, error) {
	_, f := filepath.Split(directory)
	node := types.FileTreeNode{
		Name: f,
		Path: directory,
	}

	children := []types.FileTreeNode{}

	for _, file := range files {
		d, f := filepath.Split(file)
		if strings.TrimSuffix(d, string(os.PathSeparator)) == directory {
			child := types.FileTreeNode{
				Name: f,
				Path: file,
			}
			children = append(children, child)
		}
	}

	for _, d := range directories {
		parent, _ := filepath.Split(d)
		if strings.TrimSuffix(parent, string(os.PathSeparator)) == directory {
			child, err := getDirectoryNode(d, directories, files)
			if err != nil {
				return nil, errors.Wrap(err, "failed to recursively call getDirectoryNode")
			}

			children = append(children, *child)
		}
	}

	if len(children) > 0 {
		node.Children = children
	}

	return &node, nil
}
