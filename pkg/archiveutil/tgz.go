package archiveutil

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archives"
	"github.com/pkg/errors"
)

func CreateTGZ(ctx context.Context, filenames map[string]string, dest string) error {
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
	return ExtractTGZStripComponents(ctx, archivePath, dest, 0)
}

func ExtractTGZStripComponents(ctx context.Context, archivePath string, dest string, stripComponents int) error {
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
		Extraction:  archives.Tar{},
	}

	// Use the archiver's auto-detection to determine the archive type and extract
	if err := format.Extract(ctx, srcFile, func(ctx context.Context, fi archives.FileInfo) error {
		err := extractFileToDisk(fi, dest, stripComponents)
		if err != nil {
			return errors.Wrapf(err, "file %s", fi.NameInArchive)
		}
		return nil
	}); err != nil {
		return errors.Wrap(err, "extract")
	}

	return nil
}

func extractFileToDisk(fi archives.FileInfo, dest string, stripComponents int) error {
	name := fi.NameInArchive
	if stripComponents > 0 {
		if strings.Count(name, "/") < stripComponents {
			return nil // skip path with fewer components
		}

		for i := 0; i < stripComponents; i++ {
			slash := strings.Index(name, "/")
			name = name[slash+1:]
		}
	}

	destPath := filepath.Join(dest, name)
	if fi.IsDir() {
		return os.MkdirAll(destPath, os.ModePerm)
	}

	src, err := fi.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	err = os.MkdirAll(filepath.Dir(destPath), os.ModePerm)
	if err != nil {
		return err
	}

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
