package supportbundle

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mholt/archiver/v3"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/supportbundle/types"
)

var (
	SupportBundleNameRegex = regexp.MustCompile(`^\/?support-bundle-(\d{4})-(\d{2})-(\d{2})T(\d{2})_(\d{2})_(\d{2})\/?`)
)

func archiveToFileTree(archivePath string) (*types.FileTree, error) {
	workDir, err := os.MkdirTemp("", "kotsadm")
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

		relPath, err := filepath.Rel(workDir, path)
		if err != nil {
			return errors.Wrap(err, "failed to get relative path")
		}

		// don't include the top level subdirectory in the file tree
		trimmedRelPath := SupportBundleNameRegex.ReplaceAllString(relPath, "")
		if trimmedRelPath == "" {
			return nil
		}

		if info.IsDir() {
			directories = append(directories, trimmedRelPath)
			return nil
		}

		files = append(files, trimmedRelPath)
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
