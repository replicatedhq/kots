package pull

import (
	"io/ioutil"
	"path"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/downstream"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	"github.com/replicatedhq/kots/pkg/upstream"
	"k8s.io/client-go/kubernetes/scheme"
)

type PullOptions struct {
	HelmRepoURI      string
	RootDir          string
	Overwrite        bool
	Namespace        string
	Downstreams      []string
	LocalPath        string
	LicenseFile      string
	ExcludeKotsKinds bool
}

func Pull(upstreamURI string, pullOptions PullOptions) error {
	log := logger.NewLogger()
	log.Initialize()

	fetchOptions := upstream.FetchOptions{}
	fetchOptions.HelmRepoURI = pullOptions.HelmRepoURI
	fetchOptions.LocalPath = pullOptions.LocalPath

	if pullOptions.LicenseFile != "" {
		license, err := parseLicenseFromFile(pullOptions.LicenseFile)
		if err != nil {
			return errors.Wrap(err, "failed to parse license from file")
		}

		fetchOptions.License = license
	}

	log.ActionWithSpinner("Pulling upstream")
	u, err := upstream.FetchUpstream(upstreamURI, &fetchOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to fetch upstream")
	}

	writeUpstreamOptions := upstream.WriteOptions{
		RootDir:      pullOptions.RootDir,
		CreateAppDir: true,
		Overwrite:    pullOptions.Overwrite,
	}
	if err := u.WriteUpstream(writeUpstreamOptions); err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to write upstream")
	}
	log.FinishSpinner()

	renderOptions := base.RenderOptions{
		SplitMultiDocYAML: true,
		Namespace:         pullOptions.Namespace,
	}
	log.ActionWithSpinner("Creating base")
	b, err := base.RenderUpstream(u, &renderOptions)
	if err != nil {
		return errors.Wrap(err, "failed to render upstream")
	}
	log.FinishSpinner()

	writeBaseOptions := base.WriteOptions{
		BaseDir:          u.GetBaseDir(writeUpstreamOptions),
		Overwrite:        pullOptions.Overwrite,
		ExcludeKotsKinds: pullOptions.ExcludeKotsKinds,
	}
	if err := b.WriteBase(writeBaseOptions); err != nil {
		return errors.Wrap(err, "failed to write base")
	}

	log.ActionWithSpinner("Creating midstream")
	m, err := midstream.CreateMidstream(b)
	if err != nil {
		return errors.Wrap(err, "failed to create midstream")
	}
	log.FinishSpinner()

	writeMidstreamOptions := midstream.WriteOptions{
		MidstreamDir: path.Join(b.GetOverlaysDir(writeBaseOptions), "midstream"),
		BaseDir:      u.GetBaseDir(writeUpstreamOptions),
		Overwrite:    pullOptions.Overwrite,
	}
	if err := m.WriteMidstream(writeMidstreamOptions); err != nil {
		return errors.Wrap(err, "failed to write midstream")
	}

	for _, downstreamName := range pullOptions.Downstreams {
		log.ActionWithSpinner("Creating downstream %q", downstreamName)
		d, err := downstream.CreateDownstream(m, downstreamName)
		if err != nil {
			return errors.Wrap(err, "failed to create downstream")
		}

		writeDownstreamOptions := downstream.WriteOptions{
			DownstreamDir: path.Join(b.GetOverlaysDir(writeBaseOptions), "downstreams", downstreamName),
			MidstreamDir:  writeMidstreamOptions.MidstreamDir,
		}

		if err := d.WriteDownstream(writeDownstreamOptions); err != nil {
			return errors.Wrap(err, "failed to write downstream")
		}

		log.FinishSpinner()
	}
	return nil
}

func parseLicenseFromFile(filename string) (*kotsv1beta1.License, error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read license file")
	}

	kotsscheme.AddToScheme(scheme.Scheme)
	decode := scheme.Codecs.UniversalDeserializer().Decode
	license, gvk, err := decode(contents, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode license file")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "License" {
		return nil, errors.New("not an application license")
	}

	return license.(*kotsv1beta1.License), nil
}
