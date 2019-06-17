package specs

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/libyaml"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/util"
	"github.com/spf13/afero"
	yaml "gopkg.in/yaml.v2"
)

func (r *Resolver) DefaultHelmUnforkRelease(upstreamAsset api.Asset, forkedAsset api.Asset) api.Spec {
	spec := api.Spec{
		Assets: api.Assets{
			V1: []api.Asset{
				upstreamAsset,
				forkedAsset,
			},
		},
		Lifecycle: api.Lifecycle{
			V1: []api.Step{
				{
					Render: &api.Render{
						StepShared: api.StepShared{
							ID:       "render",
							Requires: []string{"values"},
						},
						Root: ".",
					},
				},
				{
					Unfork: &api.Unfork{
						UpstreamBase: constants.KustomizeBasePath,
						ForkedBase:   constants.UnforkForkedBasePath,
						Overlay:      path.Join("overlays", "ship"),
						StepShared: api.StepShared{
							ID:       "kustomize",
							Requires: []string{"render"},
						},
						Dest: "rendered.yaml",
					},
				},
			},
		},
	}
	if !r.NoOutro {
		spec.Lifecycle.V1 = append(spec.Lifecycle.V1, api.Step{
			Message: &api.Message{
				StepShared: api.StepShared{
					ID: "outro",
					// Requires: []string{"kustomize"},
				},
				Contents: `
## Deploy

The application is ready to be deployed. To deploy it now, you can run:

	kubectl apply -f rendered.yaml

## Updates

Ship can now watch for any changes made to the application, and can download them, apply your patches, and create an updated version of the rendered.yaml. To watch for updates:

	ship watch && ship update

Running this command in the current directory will automate the process of downloading and preparing updates.

For continuous notification and preparation of application updates via email, webhook or automated pull request, create a free account at https://ship.replicated.com.
`},
		})
	}

	return spec
}

func (r *Resolver) DefaultHelmRelease(chartPath string, upstream string) api.Spec {
	valuesPath := ""

	if r.Viper.GetString("helm-values-file") != "" {
		valuesFile, err := filepath.Abs(r.Viper.GetString("helm-values-file"))
		if err != nil {
			level.Error(r.Logger).Log("event", "file not found", "file", r.Viper.GetString("helm-values-file"))
		}

		valuesPath = valuesFile
	}

	spec := api.Spec{
		Assets: api.Assets{
			V1: []api.Asset{
				{
					Helm: &api.HelmAsset{
						AssetShared: api.AssetShared{
							Dest: constants.KustomizeBasePath,
						},
						Local: &api.LocalHelmOpts{
							ChartRoot: chartPath,
						},
						ValuesFrom: &api.ValuesFrom{
							Path: constants.ShipPathInternalTmp,
						},
						Upstream: upstream,
					},
				},
			},
		},
		Lifecycle: api.Lifecycle{
			V1: []api.Step{
				{
					HelmIntro: &api.HelmIntro{
						IsUpdate: r.Viper.GetBool("IsUpdate"),
						StepShared: api.StepShared{
							ID: "intro",
						},
					},
				},
				{
					HelmValues: &api.HelmValues{
						StepShared: api.StepShared{
							ID:          "values",
							Requires:    []string{"intro"},
							Invalidates: []string{"render"},
						},
						Path: valuesPath,
					},
				},
				{
					Render: &api.Render{
						StepShared: api.StepShared{
							ID:       "render",
							Requires: []string{"values"},
						},
						Root: ".",
					},
				},
				{
					KustomizeIntro: &api.KustomizeIntro{
						StepShared: api.StepShared{
							ID: "kustomize-intro",
						},
					},
				},
				{
					Kustomize: &api.Kustomize{
						Base:    constants.KustomizeBasePath,
						Overlay: path.Join("overlays", "ship"),
						StepShared: api.StepShared{
							ID:       "kustomize",
							Requires: []string{"render"},
						},
						Dest: "rendered.yaml",
					},
				},
			},
		},
	}
	if !r.NoOutro {
		spec.Lifecycle.V1 = append(spec.Lifecycle.V1, api.Step{
			Message: &api.Message{
				StepShared: api.StepShared{
					ID:       "outro",
					Requires: []string{"kustomize"},
				},
				Contents: `
## Deploy

The application is ready to be deployed. To deploy it now, you can run:

	kubectl apply -f rendered.yaml

## Updates

Ship can now watch for any changes made to the application, and can download them, apply your patches, and create an updated version of the rendered.yaml. To watch for updates:

	ship watch && ship update

Running this command in the current directory will automate the process of downloading and preparing updates.

For continuous notification and preparation of application updates via email, webhook or automated pull request, create a free account at https://ship.replicated.com.
`},
		})
	}

	return spec
}

func (r *Resolver) DefaultRawRelease(basePath string) api.Spec {
	spec := api.Spec{
		Assets: api.Assets{
			V1: []api.Asset{},
		},
		Config: api.Config{
			V1: []libyaml.ConfigGroup{},
		},
		Lifecycle: api.Lifecycle{
			V1: []api.Step{
				{
					Render: &api.Render{
						StepShared: api.StepShared{
							ID: "render",
						},
						Root: ".",
					},
				},
				{
					KustomizeIntro: &api.KustomizeIntro{
						StepShared: api.StepShared{
							ID: "kustomize-intro",
						},
					},
				},
				{
					Kustomize: &api.Kustomize{
						Base:    basePath,
						Overlay: path.Join("overlays", "ship"),
						StepShared: api.StepShared{
							ID:          "kustomize",
							Invalidates: []string{"diff"},
						},
						Dest: "rendered.yaml",
					},
				},
			},
		},
	}
	if !r.NoOutro {
		spec.Lifecycle.V1 = append(spec.Lifecycle.V1, api.Step{
			Message: &api.Message{
				StepShared: api.StepShared{
					ID:       "outro",
					Requires: []string{"kustomize"},
				},
				Contents: `
## Deploy

The application is ready to be deployed. To deploy it now, you can run:

	kubectl apply -f rendered.yaml

## Updates

Ship can now watch for any changes made to the application, and can download them, apply your patches, and create an updated version of the rendered.yaml. To watch for updates:

  	ship watch && ship update

Running this command in the current directory will automate the process of downloading and preparing updates.

For continuous notification and preparation of application updates via email, webhook or automated pull request, create a free account at https://ship.replicated.com.
`},
		})
	}
	return spec
}

func (r *Resolver) resolveMetadata(ctx context.Context, upstream, localPath string, applicationType string) (*api.ShipAppMetadata, error) {
	debug := level.Debug(log.With(r.Logger, "method", "ResolveHelmMetadata"))

	baseMetadata, err := r.NewContentProcessor().ResolveBaseMetadata(upstream, localPath)
	if err != nil {
		return nil, errors.Wrap(err, "resolve base metadata")
	}

	if r.isEdit {
		debug.Log("event", "releaseNotes.resolve.cancel")
		state, err := r.StateManager.TryLoad()
		if err != nil {
			return nil, errors.Wrap(err, "load state to fetch metadata")
		}

		if state.Versioned().V1 == nil || state.Versioned().V1.Metadata == nil {
			return baseMetadata, nil
		}

		stateMetadata := state.Versioned().V1.Metadata
		apiMetadata := api.ShipAppMetadata{
			ReleaseNotes: stateMetadata.ReleaseNotes,
			Version:      stateMetadata.Version,
			Icon:         stateMetadata.Icon,
			Name:         stateMetadata.Name,
		}

		return &apiMetadata, nil
	}

	if util.IsGithubURL(upstream) {
		releaseNotes, err := r.GitHubFetcher.ResolveReleaseNotes(ctx, upstream)
		if err != nil {
			debug.Log("event", "releaseNotes.resolve.fail", "upstream", upstream, "err", err)
		}
		baseMetadata.ReleaseNotes = releaseNotes
	}

	err = r.StateManager.SerializeContentSHA(baseMetadata.ContentSHA)
	if err != nil {
		return nil, errors.Wrap(err, "write content sha")
	}

	localChartPath := filepath.Join(localPath, "Chart.yaml")

	exists, err := r.FS.Exists(localChartPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read file from %s", localChartPath)
	}
	if !exists {
		return baseMetadata, nil
	}

	debug.Log("phase", "read-chart", "from", localChartPath)
	chart, err := r.FS.ReadFile(localChartPath)
	if err != nil {
		return nil, errors.Wrapf(err, "read file from %s", localChartPath)
	}

	debug.Log("phase", "unmarshal-chart.yaml")
	if err := yaml.Unmarshal(chart, &baseMetadata); err != nil {
		return nil, err
	}

	if err := r.StateManager.SerializeShipMetadata(*baseMetadata, applicationType); err != nil {
		return nil, errors.Wrap(err, "write metadata to state")
	}

	return baseMetadata, nil
}

func (r *Resolver) maybeGetShipYAML(ctx context.Context, localPath string) (*api.Spec, error) {
	localReleasePaths := []string{
		filepath.Join(localPath, "ship.yaml"),
		filepath.Join(localPath, "ship.yml"),
	}

	r.ui.Info("Looking for ship.yaml ...")

	for _, shipYAMLPath := range localReleasePaths {
		upstreamShipYAMLExists, err := r.FS.Exists(shipYAMLPath)
		if err != nil {
			return nil, errors.Wrapf(err, "check file %s exists", shipYAMLPath)
		}

		if !upstreamShipYAMLExists {
			continue
		}
		upstreamRelease, err := r.FS.ReadFile(shipYAMLPath)
		if err != nil {
			return nil, errors.Wrapf(err, "read file from %s", shipYAMLPath)
		}
		var spec api.Spec
		if err := yaml.UnmarshalStrict(upstreamRelease, &spec); err != nil {
			level.Debug(r.Logger).Log("event", "release.unmarshal.fail", "error", err)
			return nil, errors.Wrapf(err, "unmarshal ship.yaml")
		}
		return &spec, nil
	}

	return nil, nil
}

type shaSummer func(fs afero.Afero, localPath string) (string, error)

func calculateContentSHA(fs afero.Afero, root string) (string, error) {
	var contents []byte
	err := fs.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return errors.Wrapf(err, "fs walk")
		}

		// check if this file is a child of `.git`
		// if it is, ignore it for the purposes of content sha calculation
		if strings.Contains(path, ".git") {
			return nil
		}

		// include the filepath in the content to be hashed - this way if a file moves from a.txt to b.txt the hash will change
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return errors.Wrapf(err, "get relative path to file %s", path)
		}
		contents = append(contents, []byte(relPath)...)

		if !info.Mode().IsRegular() {
			return nil
		}

		// TODO: Use checksum writer instead of loading ALL FILES in memory.
		fileContents, err := fs.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "read file")
		}

		contents = append(contents, fileContents...)
		return nil
	})

	if err != nil {
		return "", errors.Wrapf(err, "calculate content sha")
	}

	return fmt.Sprintf("%x", sha256.Sum256(contents)), nil
}
