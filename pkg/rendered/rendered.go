package rendered

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/apparchive"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"k8s.io/client-go/kubernetes"
)

type WriteOptions struct {
	BaseDir             string
	OverlaysDir         string
	RenderedDir         string
	Downstreams         []string
	KustomizeBinPath    string
	HelmDir             string
	Log                 *logger.CLILogger
	KotsKinds           *kotsutil.KotsKinds
	ProcessImageOptions image.ProcessImageOptions
	Clientset           kubernetes.Interface
}

func WriteRenderedApp(opts *WriteOptions) error {
	if err := apparchive.WriteRenderedApp(apparchive.AppWriteOptions{
		BaseDir:          opts.BaseDir,
		OverlaysDir:      opts.OverlaysDir,
		RenderedDir:      opts.RenderedDir,
		Downstreams:      opts.Downstreams,
		KustomizeBinPath: opts.KustomizeBinPath,
	}); err != nil {
		return errors.Wrap(err, "failed to write kustomize rendered")
	}

	if err := apparchive.WriteRenderedHelmCharts(apparchive.HelmWriteOptions{
		HelmDir:             opts.HelmDir,
		RenderedDir:         opts.RenderedDir,
		Log:                 opts.Log,
		Downstreams:         opts.Downstreams,
		KotsKinds:           opts.KotsKinds,
		ProcessImageOptions: opts.ProcessImageOptions,
		Clientset:           opts.Clientset,
	}); err != nil {
		return errors.Wrap(err, "failed to write helm rendered")
	}

	return nil
}
