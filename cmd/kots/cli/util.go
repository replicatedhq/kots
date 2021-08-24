package cli

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

func ExpandDir(input string) string {
	if input == "" {
		return ""
	}

	if strings.HasPrefix(input, "~") {
		input = filepath.Join(homeDir(), strings.TrimPrefix(input, "~"))
	}

	uploadPath, err := filepath.Abs(input)
	if err != nil {
		panic(errors.Wrapf(err, "unable to expand %q to absolute path", input))
	}

	return uploadPath
}

func homeDir() string {
	if h := os.Getenv("HOME"); h != "" {
		return h
	}
	return os.Getenv("USERPROFILE")
}

func downloadFileFromURL(destination string, url string) error {
	out, err := os.Create(destination)
	if err != nil {
		return errors.Wrap(err, "failed to create file")
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return errors.Wrap(err, "failed to http get")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return errors.Wrap(err, "failed to copy to file")
	}

	return nil
}
