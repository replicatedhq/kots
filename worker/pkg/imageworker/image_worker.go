package imageworker

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/genuinetools/reg/registry"
	semver "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/store"
	"github.com/replicatedhq/ship-cluster/worker/pkg/version"
	"go.uber.org/zap"
)

type Worker struct {
	Config *config.Config
	Logger *zap.SugaredLogger

	Store store.Store
}

func (w *Worker) Run(ctx context.Context) error {
	w.Logger.Infow("starting imageworker",
		zap.String("version", version.Version()),
		zap.String("gitSHA", version.GitSHA()),
		zap.Time("buildTime", version.BuildTime()),
	)

	errCh := make(chan error, 2)

	go func() {
		err := w.startPollingDBForImagesNeedingCheck(context.Background())
		w.Logger.Errorw("imageworker dbpoller failed", zap.Error(err))
		errCh <- errors.Wrap(err, "ready poller ended")
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	select {
	case <-c:
		// TODO: possibly cleanup
		return nil
	case err := <-errCh:
		return err
	}
}

func (w *Worker) startPollingDBForImagesNeedingCheck(ctx context.Context) error {
	for {
		select {
		case <-time.After(w.Config.DBPollInterval):
			imageCheckIDs, err := w.Store.ListReadyImageChecks(ctx)
			if err != nil {
				w.Logger.Errorw("imageworker polling failed", zap.Error(err))
				continue
			}
			if len(imageCheckIDs) == 0 {
				continue
			}
			for _, imageCheckID := range imageCheckIDs {
				if err := w.checkImage(imageCheckID); err != nil {
					w.Logger.Errorw("imageworker checkimage failed", zap.Error(err))
				}
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (w *Worker) checkImage(imageCheckID string) error {
	imageCheck, err := w.Store.GetImageCheck(context.TODO(), imageCheckID)
	if err != nil {
		return errors.Wrap(err, "get imagecheck")
	}

	completedSuccessfully := false

	// Ensure that we don't get an image stuck in the queue
	defer func() {
		if !completedSuccessfully {
			if err := w.Store.UpdateImageCheck(context.TODO(), imageCheck); err != nil {
				w.Logger.Errorw("imageworker update image check failed", zap.Error(err))
			}
		}
	}()

	hostname, imageName, tag, err := parseImageName(imageCheck.Name)
	if err != nil {
		imageCheck.CheckError = err.Error()
		return errors.Wrap(err, "parse imagename")
	}

	imageCheck.DetectedVersion = tag

	reg, err := initRegistryClient(hostname)
	if err != nil {
		return errors.Wrap(err, "init registry client")
	}

	tags, err := fetchTags(reg, imageName)
	if err != nil {
		imageCheck.IsPrivate = true
		imageCheck.CheckError = err.Error()
		return errors.Wrap(err, "fetch tags")
	}

	imageCheck.IsPrivate = false

	semverTags, _ := parseTags(tags)

	detectedSemver, err := semver.NewVersion(tag)
	if err != nil {
		if err := w.Store.UpdateImageCheck(context.TODO(), imageCheck); err != nil {
			imageCheck.CheckError = err.Error()
			return errors.Wrap(err, "update image check")
		}

		completedSuccessfully = true
	} else {
		semverTags = append(semverTags, detectedSemver)
		collection := SemverTagCollection(semverTags)

		versionsBehind, err := collection.VersionsBehind(detectedSemver)
		if err != nil {
			imageCheck.CheckError = err.Error()
			return errors.Wrap(err, "true versions behind")
		}
		trueVersionsBehind := SemverTagCollection(versionsBehind).RemoveLeastSpecific()
		behind := len(trueVersionsBehind) - 1
		imageCheck.VersionsBehind = int64(behind)

		versionPaths, err := resolveTagDates(w.Logger, reg, imageName, trueVersionsBehind)
		if err != nil {
			imageCheck.CheckError = err.Error()
			return errors.Wrap(err, "resolve tag dates")
		}
		path, err := json.Marshal(versionPaths)
		if err != nil {
			imageCheck.CheckError = err.Error()
			return errors.Wrap(err, "marshal version path")
		}
		imageCheck.Path = string(path)

		imageCheck.LatestVersion = trueVersionsBehind[len(trueVersionsBehind)-1].String()

		if err := w.Store.UpdateImageCheck(context.TODO(), imageCheck); err != nil {
			imageCheck.CheckError = err.Error()
			return errors.Wrap(err, "update image check")
		}

		completedSuccessfully = true
	}

	return nil
}

func fetchTags(reg *registry.Registry, imageName string) ([]string, error) {
	tags, err := reg.Tags(imageName)
	if err != nil {
		return nil, errors.Wrap(err, "list tags")
	}

	return tags, nil
}

func parseTags(tags []string) ([]*semver.Version, []string) {
	semverTags := make([]*semver.Version, 0, 0)
	nonSemverTags := make([]string, 0, 0)

	for _, tag := range tags {
		v, err := semver.NewVersion(tag)
		if err != nil {
			nonSemverTags = append(nonSemverTags, tag)
		} else {
			semverTags = append(semverTags, v)
		}
	}

	return semverTags, nonSemverTags
}

func initRegistryClient(hostname string) (*registry.Registry, error) {
	auth := types.AuthConfig{
		Username:      "",
		Password:      "",
		ServerAddress: hostname,
	}

	reg, err := registry.New(auth, registry.Opt{
		Timeout: time.Duration(time.Second * 5),
	})
	if err != nil {
		return nil, errors.Wrap(err, "create registry client")
	}

	return reg, nil
}
