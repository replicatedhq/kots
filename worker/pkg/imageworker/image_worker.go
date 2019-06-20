package imageworker

import (
	"context"
	"encoding/json"
	"os"
	"os/signal"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/genuinetools/reg/registry"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	semver "github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/config"
	"github.com/replicatedhq/ship-cluster/worker/pkg/store"
	"github.com/replicatedhq/ship-cluster/worker/pkg/version"
)

type Worker struct {
	Config *config.Config
	Logger log.Logger

	Store store.Store
}

func (w *Worker) Run(ctx context.Context) error {
	logger := log.With(w.Logger, "method", "imageworker.Worker.Execute")

	level.Info(logger).Log("phase", "initialize",
		"version", version.Version(),
		"gitSHA", version.GitSHA(),
		"buildTime", version.BuildTime(),
		"buildTimeFallback", version.GetBuild().TimeFallback,
	)

	errCh := make(chan error, 2)

	go func() {
		level.Info(logger).Log("event", "db.poller.ready.start")
		err := w.startPollingDBForImagesNeedingCheck(context.Background())
		level.Info(logger).Log("event", "db.poller.ready.fail", "err", err)
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
	logger := log.With(w.Logger, "method", "watchworker.Worker.startPollingDBForImagesNeedingCheck")

	for {
		select {
		case <-time.After(w.Config.DBPollInterval):
			imageCheckIDs, err := w.Store.ListReadyImageChecks(ctx)
			if err != nil {
				level.Error(logger).Log("event", "store.list.ready.image.checks.fail", "err", err)
				continue
			}
			if len(imageCheckIDs) == 0 {
				continue
			}
			for _, imageCheckID := range imageCheckIDs {
				if err := w.checkImage(imageCheckID); err != nil {
					level.Error(logger).Log("event", "check.image", "err", err)
				}
			}

		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (w *Worker) checkImage(imageCheckID string) error {
	debug := level.Debug(log.With(w.Logger, "method", "imageworker.Worker.checkImage"))
	debug.Log("event", "checkImage", "imageWatchID", imageCheckID)

	imageCheck, err := w.Store.GetImageCheck(context.TODO(), imageCheckID)
	if err != nil {
		return errors.Wrap(err, "get imagecheck")
	}

	debug.Log("name", imageCheck.Name)

	completedSuccessfully := false

	// Ensure that we don't get an image stuck in the queue
	defer func() {
		if !completedSuccessfully {
			if err := w.Store.UpdateImageCheck(context.TODO(), imageCheck); err != nil {
				level.Error(w.Logger).Log("event", "update image check with err", "err", err)
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

		debug.Log("event", "resolveTagDates")
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
