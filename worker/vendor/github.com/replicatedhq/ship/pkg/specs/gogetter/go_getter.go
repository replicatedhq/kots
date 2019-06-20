package gogetter

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	getter "github.com/hashicorp/go-getter"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/util"
	errors2 "github.com/replicatedhq/ship/pkg/util/errors"
	"github.com/spf13/afero"
)

type GoGetter struct {
	Logger       log.Logger
	FS           afero.Afero
	Subdir       string
	IsSingleFile bool
}

// TODO figure out how to copy files from host into afero filesystem for testing, or how to force go-getter to fetch into afero
func (g *GoGetter) GetFiles(ctx context.Context, upstream, savePath string) (string, error) {
	debug := level.Debug(g.Logger)
	debug.Log("event", "gogetter.GetFiles", "upstream", upstream, "savePath", savePath)

	// Remove the directory because go-getter wants to create it
	err := g.FS.RemoveAll(savePath)
	if err != nil {
		return "", errors.Wrap(err, "remove dir")
	}

	if g.IsSingleFile {
		debug.Log("event", "gogetter.getSingleFile", "upstream", upstream, "savePath", savePath)
		return g.getSingleFile(ctx, upstream, savePath)
	}

	err = getter.GetAny(savePath, upstream)
	if err != nil {
		return "", errors2.FetchFilesError{Message: err.Error()}
	}

	// check if the upstream is a local file - if it is, we shouldn't remove the .git directory
	fileDetector := getter.FileDetector{}
	if _, foundFile, err := fileDetector.Detect(upstream, savePath); !foundFile || err != nil {
		// if there is a `.git` directory, remove it - it's dynamic and will break the content hash used by `ship update`
		gitPresent, err := g.FS.Exists(path.Join(savePath, ".git"))
		if err != nil {
			return "", errors.Wrap(err, "check for .git directory")
		}
		if gitPresent {
			err := g.FS.RemoveAll(path.Join(savePath, ".git"))
			if err != nil {
				return "", errors.Wrap(err, "remove .git directory")
			}
		}
		debug.Log("event", "gitPresent.check", "gitPresent", gitPresent)
	}

	return filepath.Join(savePath, g.Subdir), nil
}

func (g *GoGetter) getSingleFile(ctx context.Context, upstream, savePath string) (string, error) {
	tmpDir := filepath.Join(constants.ShipPathInternalTmp, "gogetter-file")

	err := getter.GetAny(tmpDir, upstream)
	if err != nil {
		return "", errors2.FetchFilesError{Message: err.Error()}
	}
	defer g.FS.RemoveAll(tmpDir)

	err = g.FS.MkdirAll(filepath.Dir(filepath.Join(savePath, g.Subdir)), os.FileMode(0777))
	if err != nil {
		return "", errors.Wrap(err, "make path to move file to")
	}

	err = g.FS.Rename(filepath.Join(tmpDir, g.Subdir), filepath.Join(savePath, g.Subdir))
	if err != nil {
		return "", errors.Wrap(err, "move downloaded file to destination")
	}

	return savePath, nil
}

func IsGoGettable(path string) bool {
	_, err := getter.Detect(path, "", getter.Detectors)
	if err != nil {
		return false
	}
	return true
}

// if this path is a github path of the form `github.com/OWNER/REPO/tree/REF/SUBDIR` or `github.com/OWNER/REPO/SUBDIR`,
// change it to the go-getter form of `github.com/OWNER/REPO?ref=REF//` with a default ref of master and return a subdir of SUBDIR
// otherwise return the unmodified path
// the final param is whether the github URL is a blob (and thus a single file)
func UntreeGithub(path string) (string, string, bool) {
	githubURL, err := util.ParseGithubURL(path, "master")
	if err != nil {
		return path, "", false
	}
	return fmt.Sprintf("github.com/%s/%s?ref=%s//", githubURL.Owner, githubURL.Repo, githubURL.Ref), githubURL.Subdir, githubURL.IsBlob
}

func IsShipYaml(path string) bool {
	base := filepath.Base(path)
	return base == "ship.yaml" || base == "ship.yml"
}
