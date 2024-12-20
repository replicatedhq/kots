package plan

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	apptypes "github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/archives"
	"github.com/replicatedhq/kots/pkg/filestore"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/plan/types"
	"github.com/replicatedhq/kots/pkg/replicatedapp"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/update"
	"github.com/replicatedhq/kots/pkg/util"
)

func getReleaseManifests(a *apptypes.App, versionLabel, channelID, updateCursor string) (map[string][]byte, error) {
	if a.IsAirgap {
		return getAppManifestsFromAirgap(a, versionLabel, channelID, updateCursor)
	}
	return getAppManifestsFromOnline(a, versionLabel, channelID, updateCursor)
}

func getAppManifestsFromAirgap(a *apptypes.App, versionLabel, channelID, updateCursor string) (map[string][]byte, error) {
	manifests := make(map[string][]byte)

	airgapArchive, err := update.GetAirgapUpdate(a.Slug, channelID, updateCursor)
	if err != nil {
		return nil, errors.Wrap(err, "get airgap update")
	}

	appArchive, err := archives.GetFileContentFromTGZArchive("app.tar.gz", airgapArchive)
	if err != nil {
		return nil, errors.Wrap(err, "extract app archive")
	}

	tempDir, err := os.MkdirTemp("", "kotsadm")
	if err != nil {
		return nil, errors.Wrap(err, "create temp dir")
	}
	defer os.RemoveAll(tempDir)

	if err := archives.ExtractTGZArchiveFromReader(bytes.NewReader(appArchive), tempDir); err != nil {
		return nil, errors.Wrap(err, "extract app archive")
	}

	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		contents, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrap(err, "read file")
		}

		manifests[path] = contents
		return nil
	})
	if err != nil {
		return nil, errors.Wrap(err, "walk temp dir")
	}

	return manifests, nil
}

func getAppManifestsFromOnline(a *apptypes.App, versionLabel, channelID, updateCursor string) (map[string][]byte, error) {
	manifests := make(map[string][]byte)

	u, err := url.ParseRequestURI(fmt.Sprintf("replicated://%s", a.Slug))
	if err != nil {
		return nil, errors.Wrap(err, "parse request uri failed")
	}

	replicatedUpstream, err := replicatedapp.ParseReplicatedURL(u)
	if err != nil {
		return nil, errors.Wrap(err, "parse replicated upstream")
	}

	license, err := kotsutil.LoadLicenseFromBytes([]byte(a.License))
	if err != nil {
		return nil, errors.Wrap(err, "parse app license")
	}

	getReq, err := replicatedUpstream.GetRequest("GET", license, updateCursor, channelID)
	if err != nil {
		return nil, errors.Wrap(err, "create http request")
	}

	reporting.InjectReportingInfoHeaders(getReq, reporting.GetReportingInfo(a.ID))

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return nil, errors.Wrap(err, "execute get request")
	}
	defer getResp.Body.Close()

	if getResp.StatusCode >= 300 {
		body, _ := io.ReadAll(getResp.Body)
		if len(body) > 0 {
			return nil, util.ActionableError{Message: string(body)}
		}
		return nil, errors.Errorf("unexpected result from get request: %d", getResp.StatusCode)
	}

	gzipReader, err := gzip.NewReader(getResp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "create new gzip reader")
	}

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "get next file from reader")
		}

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			contents, err := io.ReadAll(tarReader)
			if err != nil {
				return nil, errors.Wrap(err, "read file from tar")
			}
			manifests[header.Name] = contents
		}
	}

	return manifests, nil
}

func getAppUpgradeServiceOutput(p *types.Plan) (map[string]string, error) {
	var ausOutput map[string]string
	for _, s := range p.Steps {
		if s.Type != types.StepTypeAppUpgradeService {
			continue
		}
		output := s.Output.(string)
		if output == "" {
			return nil, errors.New("app upgrade service step output not found")
		}
		if err := json.Unmarshal([]byte(output), &ausOutput); err != nil {
			return nil, errors.Wrap(err, "unmarshal app upgrade service step output")
		}
		break
	}
	if ausOutput == nil {
		return nil, errors.New("app upgrade service step output not found")
	}
	return ausOutput, nil
}

func getAppArchive(path string) (string, error) {
	if path == "" {
		return "", errors.New("path is empty")
	}

	tgzArchive, err := filestore.GetStore().ReadArchive(path)
	if err != nil {
		return "", errors.Wrap(err, "read archive")
	}
	defer os.RemoveAll(tgzArchive)

	archiveDir, err := os.MkdirTemp("", "kotsadm")
	if err != nil {
		return "", errors.Wrap(err, "create temp dir")
	}

	if err := util.ExtractTGZArchive(tgzArchive, archiveDir); err != nil {
		return "", errors.Wrap(err, "extract app archive")
	}

	return archiveDir, nil
}
