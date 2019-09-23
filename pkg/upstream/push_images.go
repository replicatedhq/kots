package upstream

import (
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/logger"
	kustomizeimage "sigs.k8s.io/kustomize/v3/pkg/image"
)

type PushUpstreamImageOptions struct {
	RootDir           string
	CreateAppDir      bool
	Log               *logger.Logger
	RegistryHost      string
	RegistryNamespace string
}

func (u *Upstream) TagAndPushUpstreamImages(options PushUpstreamImageOptions) ([]kustomizeimage.Image, error) {
	rootDir := options.RootDir
	if options.CreateAppDir {
		rootDir = path.Join(rootDir, u.Name)
	}
	imagesDir := path.Join(rootDir, "images")

	images := []kustomizeimage.Image{}

	err := filepath.Walk(imagesDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			pathWithoutRoot := path[len(imagesDir)+1:]

			rewrittenImage, err := image.ImageInfoFromFile(options.RegistryHost, options.RegistryNamespace, strings.Split(pathWithoutRoot, string(os.PathSeparator)))
			if err != nil {
				return errors.Wrap(err, "failed to decode image from path")
			}

			// copy to the registry
			options.Log.ChildActionWithSpinner("Pushing image %s:%s", rewrittenImage.NewName, rewrittenImage.NewTag)
			err = image.CopyFromFileToRegistry(path, rewrittenImage.NewName, rewrittenImage.NewTag, rewrittenImage.Digest)
			if err != nil {
				options.Log.FinishChildSpinner()
				return errors.Wrap(err, "failed to push image")
			}
			options.Log.FinishChildSpinner()

			images = append(images, rewrittenImage)
			return nil
		})

	if err != nil {
		return nil, errors.Wrap(err, "failed to walk images dir")
	}

	return images, nil
}
