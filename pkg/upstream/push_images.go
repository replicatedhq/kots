package upstream

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	kustomizeimage "sigs.k8s.io/kustomize/v3/pkg/image"
)

type PushUpstreamImageOptions struct {
	RootDir             string
	ImagesDir           string
	CreateAppDir        bool
	Log                 *logger.Logger
	ReplicatedRegistry  registry.RegistryOptions
	ReportWriter        io.Writer
	DestinationRegistry registry.RegistryOptions
}

func TagAndPushUpstreamImages(u *types.Upstream, options PushUpstreamImageOptions) ([]kustomizeimage.Image, error) {
	formatDirs, err := ioutil.ReadDir(options.ImagesDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read images dir")
	}

	images := []kustomizeimage.Image{}
	for _, f := range formatDirs {
		if !f.IsDir() {
			continue
		}

		formatRoot := path.Join(options.ImagesDir, f.Name())
		err := filepath.Walk(formatRoot,
			func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				pathWithoutRoot := path[len(formatRoot)+1:]

				rewrittenImage, err := image.ImageInfoFromFile(options.DestinationRegistry, strings.Split(pathWithoutRoot, string(os.PathSeparator)))
				if err != nil {
					return errors.Wrap(err, "failed to decode image from path")
				}

				// copy to the registry
				options.Log.ChildActionWithSpinner("Pushing image %s:%s", rewrittenImage.NewName, rewrittenImage.NewTag)

				registryAuth := image.RegistryAuth{
					Username: options.DestinationRegistry.Username,
					Password: options.DestinationRegistry.Password,
				}
				err = image.CopyFromFileToRegistry(path, rewrittenImage.NewName, rewrittenImage.NewTag, rewrittenImage.Digest, registryAuth, options.ReportWriter)
				if err != nil {
					options.Log.FinishChildSpinner()
					return errors.Wrap(err, "failed to push image")
				}
				options.Log.FinishChildSpinner()

				images = append(images, rewrittenImage)

				// kustomize does string based comparison, so all of these are treated as different images:
				// docker.io/library/redis:latest
				// redis:latest
				// redis
				// As a workaround we add all 3 to the list

				rewrittenName := rewrittenImage.Name
				if strings.HasPrefix(rewrittenName, "docker.io/library/") {
					rewrittenName = strings.TrimPrefix(rewrittenName, "docker.io/library/")
					images = append(images, kustomizeimage.Image{
						Name:    rewrittenName,
						NewName: rewrittenImage.NewName,
						NewTag:  rewrittenImage.NewTag,
						Digest:  rewrittenImage.Digest,
					})
				}

				if strings.HasSuffix(rewrittenName, ":latest") {
					rewrittenName = strings.TrimSuffix(rewrittenName, ":latest")
					images = append(images, kustomizeimage.Image{
						Name:    rewrittenName,
						NewName: rewrittenImage.NewName,
						NewTag:  rewrittenImage.NewTag,
						Digest:  rewrittenImage.Digest,
					})
				}

				return nil
			})

		if err != nil {
			return nil, errors.Wrap(err, "failed to walk images dir")
		}
	}

	return images, nil
}
