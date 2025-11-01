package lint

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/lint/types"
)

// LoadFromDirectory loads spec files from a directory on the filesystem
func LoadFromDirectory(dirPath string) (types.SpecFiles, error) {
	specFiles := types.SpecFiles{}

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Read file content
		content, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to read file %s", path)
		}

		// Get relative path from the base directory
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return errors.Wrapf(err, "failed to get relative path for %s", path)
		}

		// Convert Windows path separators to Unix-style
		relPath = filepath.ToSlash(relPath)

		specFile := types.SpecFile{
			Name:    d.Name(),
			Path:    relPath,
			Content: string(content),
		}

		specFiles = append(specFiles, specFile)
		return nil
	})

	if err != nil {
		return nil, errors.Wrapf(err, "failed to walk directory %s", dirPath)
	}

	return specFiles, nil
}

// LoadFromTar loads spec files from a tar archive reader
func LoadFromTar(reader io.Reader) (types.SpecFiles, error) {
	return types.SpecFilesFromTar(reader)
}

// LoadFiles loads spec files from a path (directory or tar archive)
func LoadFiles(path string) (types.SpecFiles, error) {
	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		// If path doesn't exist, check if it's a tar file being piped
		if os.IsNotExist(err) {
			return nil, errors.Errorf("path %s does not exist", path)
		}
		return nil, errors.Wrapf(err, "failed to stat path %s", path)
	}

	// If it's a directory, load from directory
	if info.IsDir() {
		return LoadFromDirectory(path)
	}

	// If it's a file, check if it's a tar archive
	if strings.HasSuffix(path, ".tar") || strings.HasSuffix(path, ".tar.gz") || strings.HasSuffix(path, ".tgz") {
		file, err := os.Open(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to open file %s", path)
		}
		defer file.Close()

		return LoadFromTar(file)
	}

	// If it's a single file, treat it as a single spec file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file %s", path)
	}

	specFile := types.SpecFile{
		Name:    filepath.Base(path),
		Path:    filepath.Base(path),
		Content: string(content),
	}

	return types.SpecFiles{specFile}, nil
}
