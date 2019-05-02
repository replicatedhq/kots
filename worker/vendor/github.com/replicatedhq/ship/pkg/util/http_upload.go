package util

import (
	"io/ioutil"
	"net/http"
	"os"
	"path"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mholt/archiver"
	"github.com/pkg/errors"
)

type AssetUploader interface {
	UploadAssets(target string) error
}

type assetUploader struct {
	logger log.Logger
	tar    archiver.Archiver
	client *http.Client
}

func NewAssetUploader(
	logger log.Logger,
	client *http.Client,
) AssetUploader {
	return &assetUploader{
		logger: log.With(logger, "struct", "assetUploader"),
		client: client,
	}

}

// i need tests
func (a *assetUploader) UploadAssets(target string) error {
	debug := log.With(level.Debug(a.logger), "method", "UploadAssets")
	currentWorkingDir, err := os.Getwd()
	if err != nil {
		return errors.Wrapf(err, "get working directory")
	}

	debug.Log("event", "tmpdir.create")
	// need a real tmpdir because archiver library doesn't support Afero
	tmpdir, err := ioutil.TempDir("", "ship-archive")
	if err != nil {
		return errors.Wrapf(err, "create temp dir")
	}
	defer os.RemoveAll(tmpdir)

	// in normal use this means that the targz archiver will be used, but tests can set something else if needed
	if a.tar == nil {
		a.tar = archiver.TarGz
	}

	debug.Log("event", "archive.create")
	archivePath := path.Join(tmpdir, "assets.tar.gz")
	err = a.tar.Make(archivePath, []string{currentWorkingDir})
	if err != nil {
		return errors.Wrapf(err, "create archive at ")
	}

	debug.Log("event", "archive.open")
	archive, err := os.Open(archivePath)
	if err != nil {
		return errors.Wrap(err, "open archive")
	}

	debug.Log("event", "request.create")
	request, err := http.NewRequest("PUT", target, archive)
	if err != nil {
		return errors.Wrap(err, "create request")
	}

	stat, err := archive.Stat()
	if err != nil {
		return errors.Wrap(err, "get archive info")
	}
	request.ContentLength = stat.Size()

	debug.Log("event", "request.send")
	resp, err := a.client.Do(request)
	if err != nil {
		return errors.Wrap(err, "send request")
	} else if resp == nil {
		return errors.Errorf("request returned no error, but was nil")
	}
	if resp.StatusCode > 299 {
		if resp.Body != nil {
			body, err := ioutil.ReadAll(resp.Body)
			if err == nil {
				return errors.Errorf("request returned status code %d and body %q", resp.StatusCode, string(body))
			}
		}
		return errors.Errorf("request returned status code %d", resp.StatusCode)
	}
	return nil
}
