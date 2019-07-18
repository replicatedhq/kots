package specs

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	version "github.com/hashicorp/go-version"
	"github.com/mitchellh/cli"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/fs"
	"github.com/replicatedhq/ship/pkg/specs/apptype"
	"github.com/replicatedhq/ship/pkg/specs/githubclient"
	"github.com/replicatedhq/ship/pkg/specs/replicatedapp"
	"github.com/replicatedhq/ship/pkg/state"
	"github.com/replicatedhq/ship/pkg/util"
	"github.com/spf13/afero"
	"github.com/spf13/viper"
)

type ContentProcessor struct {
	Logger           log.Logger
	FS               afero.Afero
	AppResolver      replicatedapp.Resolver
	GitHubFetcher    githubclient.GitHubFetcher
	ui               cli.Ui
	appTypeInspector apptype.Inspector
	shaSummer        shaSummer
	rootDir          string
}

func NewContentProcessor(v *viper.Viper) (*ContentProcessor, error) {
	logger := level.NewFilter(log.NewLogfmtLogger(os.Stdout), level.AllowInfo())
	fs := fs.NewBaseFilesystem()

	dir, err := fs.TempDir("", "watch")
	if err != nil {
		return nil, err
	}
	constants.SetShipRootDir(dir)

	if err := os.MkdirAll(constants.ShipPathInternalTmp, 0755); err != nil {
		return nil, err
	}

	ui := &cli.BasicUi{
		Reader:      os.Stdin,
		Writer:      ioutil.Discard,
		ErrorWriter: os.Stderr,
	}
	gqlClient, err := replicatedapp.NewGraphqlClient(v, http.DefaultClient)
	if err != nil {
		return nil, errors.Wrap(err, "create gql client")
	}
	stateManager, err := state.NewDisposableManager(logger, fs, v)
	if err != nil {
		return nil, errors.Wrap(err, "create state manager")
	}
	appTypeInspector := apptype.NewInspector(logger, fs, v, stateManager, ui)
	return &ContentProcessor{
		Logger:           logger,
		FS:               fs,
		AppResolver:      replicatedapp.NewAppResolver(v, logger, fs, gqlClient, stateManager, ui),
		GitHubFetcher:    githubclient.NewGithubClient(fs, logger),
		ui:               ui,
		appTypeInspector: appTypeInspector,
		shaSummer:        calculateContentSHA,
		rootDir:          dir,
	}, nil
}

func (r *Resolver) NewContentProcessor() *ContentProcessor {
	return &ContentProcessor{
		Logger:           r.Logger,
		FS:               r.FS,
		AppResolver:      r.AppResolver,
		GitHubFetcher:    r.GitHubFetcher,
		ui:               r.ui,
		appTypeInspector: r.appTypeInspector,
		shaSummer:        r.shaSummer,
	}
}

// MaybeResolveVersionedUpstream returns an upstream with the UpstreamVersionToken replaced with the
// latest version fetched from github unless the latest version is unable to be fetched or if the version
// stored in state is greater than the latest version. All other upstreams will be returned unmodified.
func (c *ContentProcessor) MaybeResolveVersionedUpstream(ctx context.Context, upstream string, existingState state.State) (string, error) {
	debug := level.Debug(log.With(c.Logger, "method", "resolveVersionedUpstream"))

	if !util.IsGithubURL(upstream) {
		return upstream, nil
	}

	debug.Log("event", "resolve latest release")
	latestReleaseVersion, err := c.GitHubFetcher.ResolveLatestRelease(ctx, upstream)
	if err != nil {
		if strings.Contains(upstream, UpstreamVersionToken) {
			return "", errors.Wrap(err, "resolve latest release")
		}
		return upstream, nil
	}

	maybeVersionedUpstream := strings.Replace(upstream, UpstreamVersionToken, latestReleaseVersion, 1)

	debug.Log("event", "check previous version")
	if existingState.Versioned().V1.Metadata != nil && existingState.Versioned().V1.Metadata.Version != "" {
		if strings.Contains(upstream, UpstreamVersionToken) {
			previousVersion, err := version.NewVersion(existingState.Versioned().V1.Metadata.Version)
			if err != nil {
				return maybeVersionedUpstream, nil
			}

			latestVersion, err := version.NewVersion(latestReleaseVersion)
			if err != nil {
				return maybeVersionedUpstream, nil
			}

			if latestVersion.LessThan(previousVersion) {
				return "", errors.New("Latest version less than previous")
			}
		}
	}

	return maybeVersionedUpstream, nil
}

// read the content sha without writing anything to state
func (c *ContentProcessor) ReadContentSHAForWatch(ctx context.Context, upstream string) (string, error) {

	debug := level.Debug(log.With(c.Logger, "method", "ReadContentSHAForWatch"))
	debug.Log("event", "fetch latest chart")
	app, err := c.appTypeInspector.DetermineApplicationType(ctx, upstream)
	if err != nil {
		return "", errors.Wrapf(err, "resolve app type for %s", upstream)
	}
	debug.Log("event", "apptype.inspect", "type", app.GetType(), "localPath", app.GetLocalPath())

	defer func() {
		if err := app.Remove(c.FS); err != nil {
			level.Error(c.Logger).Log("event", "remove watch dir", "err", err)
		}
	}()

	// this switch block is kinda duped from above, and we ought to centralize parts of this,
	// but in this case we only want to read the metadata without persisting anything to state,
	// and there doesn't seem to be a good way to evolve that abstraction cleanly from what we have, at least not just yet
	switch app.GetType() {
	case "helm":
		fallthrough
	case "k8s":
		fallthrough
	case "inline.replicated.app":
		metadata, err := c.ResolveBaseMetadata(upstream, app.GetLocalPath())
		if err != nil {
			return "", errors.Wrapf(err, "resolve metadata and content sha for %s %s", app.GetType(), upstream)
		}
		return metadata.ContentSHA, nil

	case "runbook.replicated.app":
		c.AppResolver.SetRunbook(app.GetLocalPath())
		fallthrough
	case "replicated.app":
		parsed, err := url.Parse(upstream)
		if err != nil {
			return "", errors.Wrapf(err, "parse url %s", upstream)
		}
		selector := (&replicatedapp.Selector{}).UnmarshalFrom(parsed)
		release, err := c.AppResolver.FetchRelease(ctx, selector)
		if err != nil {
			return "", errors.Wrap(err, "fetch release")
		}

		release.Entitlements.Signature = "" // entitlements signature is not stable
		releaseJSON, err := json.Marshal(release)
		if err != nil {
			return "", errors.Wrap(err, "marshal release for sha256")
		}

		return fmt.Sprintf("%x", sha256.Sum256(releaseJSON)), nil
	}

	return "", errors.Errorf("Could not continue with application type %q of upstream %s", app.GetType(), upstream)
}

// ResolveBaseMetadata resolves URL, ContentSHA, and Readme for the resource
func (c *ContentProcessor) ResolveBaseMetadata(upstream string, localPath string) (*api.ShipAppMetadata, error) {
	debug := level.Debug(log.With(c.Logger, "method", "resolveBaseMetaData"))
	var md api.ShipAppMetadata
	md.URL = upstream
	debug.Log("phase", "calculate-sha", "for", localPath)
	contentSHA, err := c.shaSummer(c.FS, localPath)
	if err != nil {
		return nil, errors.Wrapf(err, "calculate chart sha")
	}
	md.ContentSHA = contentSHA

	localReadmePath := filepath.Join(localPath, "README.md")
	debug.Log("phase", "read-readme", "from", localReadmePath)
	readme, err := c.FS.ReadFile(localReadmePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrapf(err, "read file from %s", localReadmePath)
		}
	}
	if readme != nil {
		md.Readme = string(readme)
	} else {
		// TODO default README better
		md.Readme = fmt.Sprintf(`
Deployment Generator
===========================

This is a deployment generator for

    ship init %s

Sources for your app have been generated at %s. This installer will walk you through
customizing these resources and preparing them to deploy to your infrastructure.
`, upstream, localPath)
	}
	return &md, nil
}

func (c *ContentProcessor) RemoveAll() error {
	return c.FS.RemoveAll(c.rootDir)
}
