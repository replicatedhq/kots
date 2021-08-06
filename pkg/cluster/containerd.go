package cluster

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

// startContainerd will start a rootless containerd in a subdirectory of dataDir
// This should start containerd with a sock file in dataDir/containerd/containerd.sock
func startContainerd(ctx context.Context, dataDir string) error {

	// TODO make this not always download, it should only download when needed
	packageURI := `https://github.com/containerd/containerd/releases/download/v1.5.1/containerd-1.5.1-linux-amd64.tar.gz`
	resp, err := http.Get(packageURI)
	if err != nil {
		return errors.Wrap(err, "download containerd")
	}
	defer resp.Body.Close()

	// TODO install runc

	// extract containerd into a new directory
	installDir := filepath.Join(dataDir, "containerd")
	if _, err := os.Stat(installDir); err == nil {
		if err := os.RemoveAll(installDir); err != nil {
			return errors.Wrap(err, "remove previous containerd")
		}
	}

	if err := os.MkdirAll(installDir, 0755); err != nil {
		return errors.Wrap(err, "mkdir")
	}

	if err := extractArchiveStreamToDir(resp.Body, installDir); err != nil {
		return errors.Wrap(err, "extract")
	}

	if err := generateDefaultConfig(dataDir, installDir); err != nil {
		return errors.Wrap(err, "generate default config")
	}

	if err := spwanContainerd(installDir); err != nil {
		return errors.Wrap(err, "spawn containerd")
	}

	return nil
}

func spwanContainerd(installDir string) error {
	go func() {
		args := []string{}

		cmd := exec.Command(filepath.Join(installDir, "bin", "containerd"), args...)
		cmd.Env = os.Environ()

		// TODO stream the output of stdout and stderr to files
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			fmt.Printf("%s\n", stderr.String())
			panic(err)
		}

		fmt.Printf("%s\n", stdout.String())
	}()

	return nil
}

func extractArchiveStreamToDir(r io.ReadCloser, dest string) error {
	uncompressedStream, err := gzip.NewReader(r)
	if err != nil {
		return errors.Wrap(err, "create gzip reader")
	}

	tarReader := tar.NewReader(uncompressedStream)

	for true {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return errors.Wrap(err, "read next")
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.Mkdir(filepath.Join(dest, (header.Name)), 0755); err != nil {
				return errors.Wrap(err, "mkdir")
			}
		case tar.TypeReg:
			outFile, err := os.Create(filepath.Join(dest, header.Name))
			if err != nil {
				return errors.Wrap(err, "create")
			}
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return errors.Wrap(err, "copy")
			}
			if err := os.Chmod(filepath.Join(dest, header.Name), fs.FileMode(header.Mode)); err != nil {
				return errors.Wrap(err, "chmod")
			}

			outFile.Close()

		default:
			return errors.New("unknown type")
		}

	}

	return nil
}

func generateDefaultConfig(dataDir string, installDir string) error {
	cmd := exec.Command(filepath.Join(installDir, "bin", "containerd"), "config", "default")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "exec containerd config default")
	}

	configFile := filepath.Join(installDir, "config.toml")
	if _, err := os.Stat(configFile); err == nil {
		if err := os.RemoveAll(configFile); err != nil {
			return errors.Wrap(err, "remove containerd config")
		}
	}

	d := filepath.Dir(configFile)
	if _, err := os.Stat(d); os.IsNotExist(err) {
		if err := os.MkdirAll(d, 0755); err != nil {
			return errors.Wrap(err, "mkdir")
		}
	}

	if err := ioutil.WriteFile(configFile, out, 0644); err != nil {
		return errors.Wrap(err, "write containerd config")
	}

	return nil
}
