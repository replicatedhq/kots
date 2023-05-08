package rendered

import (
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/helmdeploy"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/kustomize"
	"github.com/replicatedhq/kots/pkg/logger"
	midstream "github.com/replicatedhq/kots/pkg/midstream"
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
	ProcessImageOptions midstream.ProcessImageOptions
	Clientset           kubernetes.Interface
}

func WriteRenderedApp(opts *WriteOptions) error {
	if err := kustomize.WriteRenderedApp(kustomize.WriteOptions{
		BaseDir:          opts.BaseDir,
		OverlaysDir:      opts.OverlaysDir,
		RenderedDir:      opts.RenderedDir,
		Downstreams:      opts.Downstreams,
		KustomizeBinPath: opts.KustomizeBinPath,
	}); err != nil {
		return errors.Wrap(err, "failed to write kustomize rendered")
	}

	if err := helmdeploy.WriteRenderedHelmCharts(helmdeploy.WriteOptions{
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
