package archiveutil

import (
	"context"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/mholt/archives"
	"github.com/pkg/errors"
)

func ArchiveTGZ(ctx context.Context, filenames map[string]string, dest string) error {
	// Ensure the destination directory exists
	err := os.MkdirAll(filepath.Dir(dest), 0755)
	if err != nil {
		return errors.Wrap(err, "create destination directory")
	}

	// Create the destination file
	destFile, err := os.Create(dest)
	if err != nil {
		return errors.Wrap(err, "create destination file")
	}
	defer destFile.Close()

	// Set up the CompressedArchive format
	format := archives.CompressedArchive{
		Compression: archives.Gz{},
		Archival:    archives.Tar{},
	}

	// Get file info for the source directory
	fileInfos, err := archives.FilesFromDisk(ctx, nil, filenames)
	if err != nil {
		return errors.Wrap(err, "get file info from disk")
	}

	// Archive the source directory to the destination file
	if err = format.Archive(ctx, destFile, fileInfos); err != nil {
		return errors.Wrap(err, "failed to create archive")
	}

	return nil
}

func ExtractTGZ(ctx context.Context, archivePath string, dest string) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(dest, 0755); err != nil {
		return errors.Wrapf(err, "create destination directory %q", dest)
	}

	// Open the archive file
	srcFile, err := os.Open(archivePath)
	if err != nil {
		return errors.Wrap(err, "open archive file")
	}
	defer srcFile.Close()

	// Set up the CompressedArchive format
	format := archives.CompressedArchive{
		Compression: archives.Gz{},
		Archival:    archives.Tar{},
	}

	// Use the archiver's auto-detection to determine the archive type and extract
	if err := format.Extract(ctx, srcFile, func(ctx context.Context, fi archives.FileInfo) error {
		err := extractFileToDisk(fi, dest)
		if err != nil {
			return errors.Wrapf(err, "file %s", fi.NameInArchive)
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "extract")
	}

	return nil
}

// func filesFromDisk(ctx context.Context, srcs []string, dest string) ([]archives.FileInfo, error) {
// 	filenames := map[string]string{}

// 	var baseDir string
// 	if implicitTopLevelFolder && multipleTopLevels(srcs) {
// 		baseDir = folderNameFromFileName(dest)
// 	}

// 	for _, src := range srcs {
// 		filenames[src] = path.Join(baseDir, src)
// 	}

// 	return archives.FilesFromDisk(ctx, nil, filenames)
// }

// func makeNameInArchive(src string, baseDir string) (string, error) {
// 	fi, err := os.Stat(src)
// 	if err != nil {
// 		return "", errors.Wrapf(err, "stat %s", src)
// 	}

// 	name := filepath.Base(src) // start with the file or dir name
// 	if fi.IsDir() {
// 		src += "/" // if this is a directory, preserve the trailing slash
// 	}
// 	return path.Join(baseDir, name), nil // prepend the base directory
// }

func extractFileToDisk(fi archives.FileInfo, dest string) error {
	destPath := filepath.Join(dest, fi.NameInArchive)
	if fi.IsDir() {
		return os.MkdirAll(destPath, os.ModePerm)
	}

	src, err := fi.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}

	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	return os.Chmod(destPath, fi.Mode().Perm())
}

// multipleTopLevels returns true if the paths do not
// share a common top-level folder.
func multipleTopLevels(paths []string) bool {
	if len(paths) < 2 {
		return false
	}
	var lastTop string
	for _, p := range paths {
		p = strings.TrimPrefix(strings.Replace(p, `\`, "/", -1), "/")
		for {
			next := path.Dir(p)
			if next == "." {
				break
			}
			p = next
		}
		if lastTop == "" {
			lastTop = p
		}
		if p != lastTop {
			return true
		}
	}
	return false
}

// folderNameFromFileName returns a name for a folder
// that is suitable based on the filename, which will
// be stripped of its extensions.
func folderNameFromFileName(filename string) string {
	base := filepath.Base(filename)
	firstDot := strings.Index(base, ".")
	if firstDot > -1 {
		return base[:firstDot]
	}
	return base
}
