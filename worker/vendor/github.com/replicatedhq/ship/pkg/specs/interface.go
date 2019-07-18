package specs

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/specs/apptype"
	"github.com/replicatedhq/ship/pkg/specs/replicatedapp"
	"github.com/replicatedhq/ship/pkg/util"
	yaml "gopkg.in/yaml.v2"
)

func (r *Resolver) ResolveUnforkRelease(ctx context.Context, upstream string, forked string) (*api.Release, error) {
	debug := log.With(level.Debug(r.Logger), "method", "ResolveUnforkReleases")
	r.ui.Info(fmt.Sprintf("Reading %s and %s ...", upstream, forked))

	// Prepare the upstream
	r.ui.Info("Determining upstream application type ...")
	upstreamApp, err := r.appTypeInspector.DetermineApplicationType(ctx, upstream)
	if err != nil {
		return nil, errors.Wrapf(err, "determine type of %s", upstream)
	}
	debug.Log("event", "applicationType.resolve", "type", upstreamApp.GetType())
	r.ui.Info(fmt.Sprintf("Detected upstream application type %s", upstreamApp.GetType()))

	debug.Log("event", "versionedUpstream.resolve", "type", upstreamApp.GetType())
	versionedUpstream, err := r.maybeCreateVersionedUpstream(upstream)
	if err != nil {
		return nil, errors.Wrap(err, "resolve versioned upstream")
	}

	debug.Log("event", "upstream.Serialize", "for", upstreamApp.GetLocalPath(), "upstream", versionedUpstream)
	err = r.StateManager.SerializeUpstream(versionedUpstream)
	if err != nil {
		return nil, errors.Wrapf(err, "write upstream")
	}

	// Prepare the fork
	r.ui.Info("Determining forked application type ...")
	forkedApp, err := r.appTypeInspector.DetermineApplicationType(ctx, forked)
	if err != nil {
		return nil, errors.Wrapf(err, "determine type of %s", forked)
	}

	debug.Log("event", "applicationType.resolve", "type", forkedApp.GetType())
	r.ui.Info(fmt.Sprintf("Detected forked application type %s", forkedApp.GetType()))

	if forkedApp.GetType() == "helm" && upstreamApp.GetType() == "k8s" {
		return nil, errors.New("Unsupported fork and upstream combination")
	}

	var forkedAsset api.Asset
	switch forkedApp.GetType() {
	case "helm":
		forkedAsset = api.Asset{
			Helm: &api.HelmAsset{
				AssetShared: api.AssetShared{
					Dest: constants.UnforkForkedBasePath,
				},
				Local: &api.LocalHelmOpts{
					ChartRoot: constants.HelmChartForkedPath,
				},
				ValuesFrom: &api.ValuesFrom{
					Path:        constants.HelmChartForkedPath,
					SaveToState: true,
				},
				Upstream: forked,
			},
		}
	case "k8s":
		forkedAsset = api.Asset{
			Local: &api.LocalAsset{
				AssetShared: api.AssetShared{
					Dest: constants.UnforkForkedBasePath,
				},
				Path: constants.HelmChartForkedPath,
			},
		}
	default:
		return nil, errors.Errorf("unknown forked application type %q", forkedApp.GetType())
	}

	var upstreamAsset api.Asset
	switch upstreamApp.GetType() {
	case "helm":
		upstreamAsset = api.Asset{
			Helm: &api.HelmAsset{
				AssetShared: api.AssetShared{
					Dest: constants.KustomizeBasePath,
				},
				Local: &api.LocalHelmOpts{
					ChartRoot: constants.HelmChartPath,
				},
				ValuesFrom: &api.ValuesFrom{
					Path: constants.HelmChartPath,
				},
				Upstream: upstream,
			},
		}
	case "k8s":
		upstreamAsset = api.Asset{
			Local: &api.LocalAsset{
				AssetShared: api.AssetShared{
					Dest: constants.KustomizeBasePath,
				},
				Path: constants.HelmChartPath,
			},
		}
	default:
		return nil, errors.Errorf("unknown upstream application type %q", upstreamApp.GetType())
	}

	defaultRelease := r.DefaultHelmUnforkRelease(upstreamAsset, forkedAsset)

	return r.resolveUnforkRelease(
		ctx,
		upstream,
		forked,
		upstreamApp,
		forkedApp,
		constants.HelmChartPath,
		constants.HelmChartForkedPath,
		&defaultRelease,
	)
}

// A resolver turns a target string into a release.
//
// A "target string" is something like
//
//   github.com/helm/charts/stable/nginx-ingress
//   replicated.app/cool-ci-tool?customer_id=...&installation_id=...
//   file::/home/bob/apps/ship.yaml
//   file::/home/luke/my-charts/proton-torpedoes
func (r *Resolver) ResolveRelease(ctx context.Context, upstream string) (*api.Release, error) {
	debug := log.With(level.Debug(r.Logger), "method", "ResolveRelease")
	r.ui.Info(fmt.Sprintf("Reading %s ...", upstream))

	r.ui.Info("Determining application type ...")
	app, err := r.appTypeInspector.DetermineApplicationType(ctx, upstream)
	if err != nil {
		return nil, errors.Wrapf(err, "determine type of %s", upstream)
	}
	debug.Log("event", "applicationType.resolve", "type", app.GetType())
	r.ui.Info(fmt.Sprintf("Detected application type %s", app.GetType()))

	debug.Log("event", "versionedUpstream.resolve", "type", app.GetType())
	versionedUpstream, err := r.maybeCreateVersionedUpstream(upstream)
	if err != nil {
		return nil, errors.Wrap(err, "resolve versioned upstream")
	}

	debug.Log("event", "upstream.Serialize", "for", app.GetLocalPath(), "upstream", versionedUpstream)

	if !r.isEdit {
		err = r.StateManager.SerializeUpstream(versionedUpstream)
		if err != nil {
			return nil, errors.Wrapf(err, "write upstream")
		}
	}

	if app.GetType() != "replicated.app" {
		debug.Log("event", "persist app state")
		persistPath := app.GetLocalPath()
		if app.GetType() == "runbook.replicated.app" {
			persistPath = filepath.Dir(app.GetLocalPath())
		}

		err = r.persistToState(persistPath)
		if err != nil {
			return nil, errors.Wrapf(err, "persist %s to state from path %s", app.GetType(), persistPath)
		}
	}

	switch app.GetType() {

	case "helm":
		defaultRelease := r.DefaultHelmRelease(app.GetLocalPath(), upstream)

		return r.resolveRelease(
			ctx,
			upstream,
			app,
			constants.HelmChartPath,
			&defaultRelease,
			true,
			true,
		)

	case "k8s":
		defaultRelease := r.DefaultRawRelease(constants.KustomizeBasePath)

		return r.resolveRelease(
			ctx,
			upstream,
			app,
			constants.KustomizeBasePath,
			&defaultRelease,
			false,
			true,
		)

	case "runbook.replicated.app":
		r.AppResolver.SetRunbook(app.GetLocalPath())
		fallthrough
	case "replicated.app":
		if r.isEdit {
			return r.AppResolver.ResolveEditRelease(ctx)
		}

		parsed, err := url.Parse(upstream)
		if err != nil {
			return nil, errors.Wrapf(err, "parse url %s", upstream)
		}
		selector := (&replicatedapp.Selector{}).UnmarshalFrom(parsed)
		return r.AppResolver.ResolveAppRelease(ctx, selector, app)

	case "inline.replicated.app":
		return r.resolveInlineShipYAMLRelease(
			ctx,
			upstream,
			app,
		)

	}

	return nil, errors.Errorf("unknown application type %q for upstream %q", app.GetType(), upstream)
}

func (r *Resolver) resolveUnforkRelease(
	ctx context.Context,
	upstream string,
	forked string,
	upstreamApp apptype.LocalAppCopy,
	forkedApp apptype.LocalAppCopy,
	destUpstreamPath string,
	destForkedPath string,
	defaultSpec *api.Spec,
) (*api.Release, error) {
	var releaseName string
	debug := log.With(level.Debug(r.Logger), "method", "resolveUnforkReleases")

	if r.Viper.GetBool("rm-asset-dest") {
		err := r.FS.RemoveAll(destUpstreamPath)
		if err != nil {
			return nil, errors.Wrapf(err, "remove asset dest %s", destUpstreamPath)
		}

		err = r.FS.RemoveAll(destForkedPath)
		if err != nil {
			return nil, errors.Wrapf(err, "remove asset dest %s", destForkedPath)
		}
	}

	err := util.BailIfPresent(r.FS, destUpstreamPath, debug)
	if err != nil {
		return nil, errors.Wrapf(err, "backup %s", destUpstreamPath)
	}

	err = r.FS.MkdirAll(filepath.Dir(destUpstreamPath), 0777)
	if err != nil {
		return nil, errors.Wrapf(err, "mkdir %s", destUpstreamPath)
	}

	err = r.FS.MkdirAll(filepath.Dir(destForkedPath), 0777)
	if err != nil {
		return nil, errors.Wrapf(err, "mkdir %s", destForkedPath)
	}

	err = r.FS.Rename(upstreamApp.GetLocalPath(), destUpstreamPath)
	if err != nil {
		return nil, errors.Wrapf(err, "move %s to %s", upstreamApp.GetLocalPath(), destUpstreamPath)
	}

	err = r.FS.Rename(forkedApp.GetLocalPath(), destForkedPath)
	if err != nil {
		return nil, errors.Wrapf(err, "move %s to %s", forkedApp.GetLocalPath(), destForkedPath)
	}

	if forkedApp.GetType() == "k8s" {
		// Pre-emptively need to split here in order to get the release name before
		// helm template is run on the upstream
		if err := util.MaybeSplitMultidocYaml(ctx, r.FS, destForkedPath); err != nil {
			return nil, errors.Wrapf(err, "maybe split multidoc in %s", destForkedPath)
		}

		debug.Log("event", "maybeGetReleaseName")
		releaseName, err = r.maybeGetReleaseName(destForkedPath)
		if err != nil {
			return nil, errors.Wrap(err, "maybe get release name")
		}
	}

	upstreamMetadata, err := r.resolveMetadata(context.Background(), upstream, destUpstreamPath, upstreamApp.GetType())
	if err != nil {
		return nil, errors.Wrapf(err, "resolve metadata for %s", destUpstreamPath)
	}

	release := &api.Release{
		Metadata: api.ReleaseMetadata{
			ShipAppMetadata: *upstreamMetadata,
		},
		Spec: *defaultSpec,
	}

	if releaseName == "" {
		releaseName = release.Metadata.ReleaseName()
	}

	if err := r.StateManager.SerializeReleaseName(releaseName); err != nil {
		debug.Log("event", "serialize.releaseName.fail", "err", err)
		return nil, errors.Wrapf(err, "serialize helm release name")
	}

	return release, nil
}

func (r *Resolver) maybeGetReleaseName(path string) (string, error) {
	type k8sReleaseMetadata struct {
		Metadata struct {
			Labels struct {
				Release string `yaml:"release"`
			} `yaml:"labels"`
		} `yaml:"metadata"`
	}

	files, err := r.FS.ReadDir(path)
	if err != nil {
		return "", errors.Wrapf(err, "read dir %s", path)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".yaml" || filepath.Ext(file.Name()) == ".yml" {
			fileB, err := r.FS.ReadFile(filepath.Join(path, file.Name()))
			if err != nil {
				return "", errors.Wrapf(err, "read file %s", path)
			}

			releaseMetadata := k8sReleaseMetadata{}
			if err := yaml.Unmarshal(fileB, &releaseMetadata); err != nil {
				return "", errors.Wrapf(err, "unmarshal for release metadata %s", path)
			}

			if releaseMetadata.Metadata.Labels.Release != "" {
				return releaseMetadata.Metadata.Labels.Release, nil
			}
		}
	}

	return "", nil
}

func (r *Resolver) resolveRelease(
	ctx context.Context,
	upstream string,
	app apptype.LocalAppCopy,
	destPath string,
	defaultSpec *api.Spec,
	keepOriginal bool,
	tryUseUpstreamShipYAML bool,
) (*api.Release, error) {
	debug := log.With(level.Debug(r.Logger), "method", "resolveRelease")

	if r.Viper.GetBool("rm-asset-dest") {
		err := r.FS.RemoveAll(destPath)
		if err != nil {
			return nil, errors.Wrapf(err, "remove asset dest %s", destPath)
		}
	}

	err := util.BailIfPresent(r.FS, destPath, debug)
	if err != nil {
		return nil, errors.Wrapf(err, "backup %s", destPath)
	}

	if !keepOriginal {
		err = r.FS.Rename(app.GetLocalPath(), destPath)
		if err != nil {
			return nil, errors.Wrapf(err, "move %s to %s", app.GetLocalPath(), destPath)
		}
	} else {
		// instead of renaming, copy files from localPath to destPath
		err = r.recursiveCopy(app.GetLocalPath(), destPath)
		if err != nil {
			return nil, errors.Wrapf(err, "copy %s to %s", app.GetLocalPath(), destPath)
		}
	}

	metadata, err := r.resolveMetadata(context.Background(), upstream, destPath, app.GetType())
	if err != nil {
		return nil, errors.Wrapf(err, "resolve metadata for %s", destPath)
	}

	var spec *api.Spec
	if tryUseUpstreamShipYAML {
		debug.Log("event", "check upstream for ship.yaml")
		spec, err = r.maybeGetShipYAML(ctx, destPath)
		if err != nil {
			return nil, errors.Wrapf(err, "resolve ship.yaml release for %s", destPath)
		}
	}

	if spec == nil {
		debug.Log("event", "no ship.yaml for release")
		r.ui.Info("ship.yaml not found in upstream, generating default lifecycle for application ...")
		spec = defaultSpec
	}

	if metadata == nil {
		metadata = &api.ShipAppMetadata{}
	}

	release := &api.Release{
		Metadata: api.ReleaseMetadata{
			ShipAppMetadata: *metadata,
			Type:            app.GetType(),
		},
		Spec: *spec,
	}

	currentState, err := r.StateManager.CachedState()
	if err != nil {
		return nil, errors.Wrap(err, "try load")
	}

	releaseName := currentState.CurrentReleaseName()
	if releaseName == "" {
		debug.Log("event", "resolve.releaseName.fromRelease")
		releaseName = release.Metadata.ReleaseName()
	}

	if err := r.StateManager.SerializeReleaseName(releaseName); err != nil {
		debug.Log("event", "serialize.releaseName.fail", "err", err)
		return nil, errors.Wrapf(err, "serialize helm release name")
	}

	return release, nil
}

func (r *Resolver) recursiveCopy(sourceDir, destDir string) error {
	err := r.FS.MkdirAll(destDir, os.FileMode(0777))
	if err != nil {
		return errors.Wrapf(err, "create dest dir %s", destDir)
	}
	srcFiles, err := r.FS.ReadDir(sourceDir)
	if err != nil {
		return errors.Wrapf(err, "")
	}
	for _, file := range srcFiles {
		if file.IsDir() {
			err = r.recursiveCopy(filepath.Join(sourceDir, file.Name()), filepath.Join(destDir, file.Name()))
			if err != nil {
				return errors.Wrapf(err, "copy dir %s", file.Name())
			}
		} else {
			// is file
			contents, err := r.FS.ReadFile(filepath.Join(sourceDir, file.Name()))
			if err != nil {
				return errors.Wrapf(err, "read file %s to copy", file.Name())
			}

			err = r.FS.WriteFile(filepath.Join(destDir, file.Name()), contents, file.Mode())
			if err != nil {
				return errors.Wrapf(err, "write file %s to copy", file.Name())
			}
		}
	}
	return nil
}

func (r *Resolver) resolveInlineShipYAMLRelease(
	ctx context.Context,
	upstream string,
	app apptype.LocalAppCopy,
) (*api.Release, error) {
	debug := log.With(level.Debug(r.Logger), "method", "resolveInlineShipYAMLRelease")
	metadata, err := r.resolveMetadata(context.Background(), upstream, app.GetLocalPath(), app.GetType())
	if err != nil {
		return nil, errors.Wrapf(err, "resolve metadata for %s", app.GetLocalPath())
	}
	debug.Log("event", "check upstream for ship.yaml")
	spec, err := r.maybeGetShipYAML(ctx, app.GetLocalPath())
	if err != nil || spec == nil {
		return nil, errors.Wrapf(err, "resolve ship.yaml release for %s", app.GetLocalPath())
	}
	release := &api.Release{
		Metadata: api.ReleaseMetadata{
			ShipAppMetadata: *metadata,
			Type:            app.GetType(),
		},
		Spec: *spec,
	}
	releaseName := release.Metadata.ReleaseName()
	debug.Log("event", "resolve.releaseName")
	if err := r.StateManager.SerializeReleaseName(releaseName); err != nil {
		debug.Log("event", "serialize.releaseName.fail", "err", err)
		return nil, errors.Wrapf(err, "serialize helm release name")
	}
	return release, nil
}
