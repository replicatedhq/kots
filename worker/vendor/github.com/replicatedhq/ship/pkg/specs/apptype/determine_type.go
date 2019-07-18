package apptype

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/specs/githubclient"
	"github.com/replicatedhq/ship/pkg/specs/gogetter"
	"github.com/replicatedhq/ship/pkg/specs/localgetter"
	"github.com/replicatedhq/ship/pkg/specs/stategetter"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/util"
	errors2 "github.com/replicatedhq/ship/pkg/util/errors"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

type LocalAppCopy interface {
	GetType() string
	GetLocalPath() string
	Remove(FS afero.Afero) error
}

type Inspector interface {
	// DetermineApplicationType loads and application from upstream,
	// returning the app type and the local path where its been downloaded (when applicable),
	DetermineApplicationType(
		ctx context.Context,
		upstream string,
	) (app LocalAppCopy, err error)
}

func NewInspector(
	logger log.Logger,
	fs afero.Afero,
	v *viper.Viper,
	stateManager state.Manager,
	ui cli.Ui,
) Inspector {
	return &inspector{
		logger: logger,
		fs:     fs,
		viper:  v,
		state:  stateManager,
		ui:     ui,
		isEdit: v.GetBool("isEdit"),
	}
}

type inspector struct {
	logger log.Logger
	fs     afero.Afero
	viper  *viper.Viper
	state  state.Manager
	ui     cli.Ui
	isEdit bool
}

type FileFetcher interface {
	GetFiles(ctx context.Context, upstream, savePath string) (string, error)
}

func (i *inspector) DetermineApplicationType(ctx context.Context, upstream string) (app LocalAppCopy, err error) {
	// hack hack hack
	isReplicatedApp := strings.HasPrefix(upstream, "replicated.app") ||
		strings.HasPrefix(upstream, "staging.replicated.app") ||
		strings.HasPrefix(upstream, "local.replicated.app")
	if isReplicatedApp {
		return &localAppCopy{AppType: "replicated.app"}, nil
	}

	parts := strings.SplitN(upstream, "?", 2)
	if _, err := os.Stat(parts[0]); err == nil && gogetter.IsShipYaml(parts[0]) {
		return &localAppCopy{AppType: "runbook.replicated.app", LocalPath: parts[0]}, nil
	}

	if i.isEdit {
		return i.fetchEditFiles(ctx)
	}

	i.ui.Info(fmt.Sprintf("Attempting to retrieve upstream %s ...", upstream))
	// use the integrated github client if the url is a github url and does not contain "//", unless perfer-git is set)
	if i.viper.GetBool("prefer-git") == false && util.IsGithubURL(upstream) {
		githubClient := githubclient.NewGithubClient(i.fs, i.logger)
		return i.determineTypeFromContents(ctx, upstream, githubClient)
	}

	if localgetter.IsLocalFile(&i.fs, upstream) {
		fetcher := localgetter.LocalGetter{Logger: i.logger, FS: i.fs}
		return i.determineTypeFromContents(ctx, upstream, &fetcher)
	}

	upstream, subdir, isSingleFile := gogetter.UntreeGithub(upstream)
	if !isSingleFile {
		isSingleFile = gogetter.IsShipYaml(upstream)
	}
	if gogetter.IsGoGettable(upstream) {
		// get with go-getter
		fetcher := gogetter.GoGetter{Logger: i.logger, FS: i.fs, Subdir: subdir, IsSingleFile: isSingleFile}
		return i.determineTypeFromContents(ctx, upstream, &fetcher)
	}

	return nil, errors.Errorf("upstream %s is not a replicated app, a github repo, or compatible with go-getter", upstream)
}

func (i *inspector) fetchEditFiles(ctx context.Context) (app LocalAppCopy, err error) {
	state, err := i.state.CachedState()
	if err != nil {
		return nil, errors.Wrap(err, "load app state")
	}
	upstreamContents := state.UpstreamContents()
	if upstreamContents == nil {
		return nil, fmt.Errorf("no upstream contents present")
	}

	if upstreamContents.AppRelease != nil {
		return &localAppCopy{AppType: "replicated.app"}, nil
	}

	// create a new fetcher class that gets things from the state file
	stateClient := stategetter.NewStateGetter(i.fs, i.logger, upstreamContents)
	return i.determineTypeFromContents(ctx, "ship statefile", stateClient)
}

func (i *inspector) determineTypeFromContents(ctx context.Context, upstream string, fetcher FileFetcher) (app LocalAppCopy, err error) {
	debug := level.Debug(i.logger)

	repoSavePath, err := i.fs.TempDir(constants.ShipPathInternalTmp, "repo")
	if err != nil {
		return nil, errors.Wrap(err, "create tmp dir")
	}

	finalPath, err := fetcher.GetFiles(ctx, upstream, repoSavePath)
	if err != nil {
		if _, ok := err.(errors2.FetchFilesError); ok {
			i.ui.Info(fmt.Sprintf("Failed to retrieve upstream %s", upstream))

			var retryError = err
			retries := i.viper.GetInt("retries")
			hasSucceeded := false
			for idx := 1; idx <= retries && !hasSucceeded; idx++ {
				debug.Log("event", "retry.getFiles", "attempt", idx)
				i.ui.Info(fmt.Sprintf("Retrying to retrieve upstream %s ...", upstream))

				time.Sleep(time.Second * 5)
				finalPath, retryError = fetcher.GetFiles(ctx, upstream, repoSavePath)

				if retryError != nil {
					i.ui.Info(fmt.Sprintf("Retry attempt %v out of %v to fetch upstream failed", idx, retries))
					level.Error(i.logger).Log("event", "getFiles", "err", retryError)
				} else {
					hasSucceeded = true
				}
			}

			if !hasSucceeded {
				return nil, retryError
			}
		} else {
			return nil, err
		}
	}

	// if there's a ship.yaml, assume its a replicated.app
	var isReplicatedApp bool
	for _, filename := range []string{"ship.yaml", "ship.yml"} {
		isReplicatedApp, err = i.fs.Exists(path.Join(finalPath, filename))
		if err != nil {
			return nil, errors.Wrapf(err, "check for %s", filename)
		}
		if isReplicatedApp {
			return &localAppCopy{AppType: "inline.replicated.app", LocalPath: finalPath, rootTempDir: repoSavePath}, nil
		}
	}

	// if there's a Chart.yaml, assume its a chart
	isChart, err := i.fs.Exists(path.Join(finalPath, "Chart.yaml"))
	if err != nil {
		isChart = false
	}
	debug.Log("event", "isChart.check", "isChart", isChart)

	if isChart {
		return &localAppCopy{AppType: "helm", LocalPath: finalPath, rootTempDir: repoSavePath}, nil
	}

	return &localAppCopy{AppType: "k8s", LocalPath: finalPath, rootTempDir: repoSavePath}, nil
}

func NewLocalAppCopy(
	appType string,
	localPath string,
	rootTempDir string,
) LocalAppCopy {
	return &localAppCopy{
		AppType:     appType,
		LocalPath:   localPath,
		rootTempDir: rootTempDir,
	}
}

type localAppCopy struct {
	AppType     string
	LocalPath   string
	rootTempDir string
}

func (c *localAppCopy) GetType() string {
	return c.AppType
}

func (c *localAppCopy) GetLocalPath() string {
	return c.LocalPath
}

func (c *localAppCopy) Remove(fs afero.Afero) error {
	if c.rootTempDir == "" {
		return nil
	}
	return fs.RemoveAll(c.rootTempDir)
}
