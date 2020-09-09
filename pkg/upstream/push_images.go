package upstream

import (
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	kotsadmtypes "github.com/replicatedhq/kots/pkg/kotsadm/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
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

func TagAndPushUpstreamImages(u *types.Upstream, options PushUpstreamImageOptions) ([]kustomizetypes.Image, error) {
	pushOpts := kotsadmtypes.PushImagesOptions{
		Registry:       options.DestinationRegistry,
		Log:            options.Log,
		ProgressWriter: options.ReportWriter,
		LogForUI:       true,
	}
	images, err := kotsadm.TagAndPushAppImages(options.RootDir, pushOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to push images")
	}

	withAltNames := make([]kustomizetypes.Image, 0)
	for _, i := range images {
		withAltNames = append(withAltNames, buildImageAltNames(i)...)
	}

	return withAltNames, nil
}

func buildImageAltNames(rewrittenImage kustomizetypes.Image) []kustomizetypes.Image {
	// kustomize does string based comparison, so all of these are treated as different images:
	// docker.io/library/redis:latest
	// redis:latest
	// redis
	// As a workaround we add all 3 to the list

	// similarly, docker.io/notlibrary/image:tag needs to be rewritten
	// as notlibrary/image:tag (and the same handling for 'latest')

	images := []kustomizetypes.Image{rewrittenImage}

	rewrittenName := rewrittenImage.Name
	if strings.HasPrefix(rewrittenName, "docker.io/library/") {
		rewrittenName = strings.TrimPrefix(rewrittenName, "docker.io/library/")
		images = append(images, kustomizetypes.Image{
			Name:    rewrittenName,
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
	} else if strings.HasPrefix(rewrittenName, "docker.io/") {
		rewrittenName = strings.TrimPrefix(rewrittenName, "docker.io/")
		images = append(images, kustomizetypes.Image{
			Name:    rewrittenName,
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})

	}

	if strings.HasSuffix(rewrittenName, ":latest") {
		rewrittenName = strings.TrimSuffix(rewrittenName, ":latest")
		images = append(images, kustomizetypes.Image{
			Name:    rewrittenName,
			NewName: rewrittenImage.NewName,
			NewTag:  rewrittenImage.NewTag,
			Digest:  rewrittenImage.Digest,
		})
	}

	return images
}

type ProgressReport struct {
	// set to "progressReport"
	Type string `json:"type"`
	// the same progress text that used to be sent in unstructured message
	CompatibilityMessage string `json:"compatibilityMessage"`
	// all images found in archive
	Images []ProgressImage `json:"images"`
}

type ProgressImage struct {
	// image name and tag, "nginx:latest"
	DisplayName string `json:"displayName"`
	// image upload status: queued, uploading, uploaded, failed
	Status string `json:"status"`
	// error string set when status is failed
	Error string `json:"error"`
	// amount currently uploaded (currently number of layers)
	Current int64 `json:"current"`
	// total amount that needs to be uploaded (currently number of layers)
	Total int64 `json:"total"`
	// time when image started uploading
	StartTime time.Time `json:"startTime"`
	// time when image finished uploading
	EndTime time.Time `json:"endTime"`
}
