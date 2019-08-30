package pull

import (
	"io/ioutil"
	"net/url"
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
	HelmRepoURI         string
	RootDir             string
	Namespace           string
	Downstreams         []string
	LocalPath           string
	LicenseFile         string
	ExcludeKotsKinds    bool
	ExcludeAdminConsole bool
	SharedPassword      string
}

// PullApplicationMetadata will return the application metadata yaml, if one is
// available for the upstream
func PullApplicationMetadata(upstreamURI string) ([]byte, error) {
	u, err := url.ParseRequestURI(upstreamURI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse uri")
	}

	// metadata is only currently supported on licensed apps
	if u.Scheme != "replicated" {
		return nil, nil
	}

	data, err := upstream.GetApplicationMetadata(u)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get application metadata")
	}

	return data, nil
}

// CanPullUpstream will return a bool indicating if the specified upstream
// is accessible and authenticed for us.
func CanPullUpstream(upstreamURI string, pullOptions PullOptions) (bool, error) {
	u, err := url.ParseRequestURI(upstreamURI)
	if err != nil {
		return false, errors.Wrap(err, "failed to parse uri")
	}

	if u.Scheme != "replicated" {
		return true, nil
	}

	// For now, we shortcut http checks because all replicated:// app types
	// require a license to pull.
	return pullOptions.LicenseFile != "", nil
}

// Pull will download the application specified in upstreamURI using the options
// specified in pullOptions. It returns the directory that the app was pulled to
func Pull(upstreamURI string, pullOptions PullOptions) (string, error) {
	log := logger.NewLogger()
	log.Initialize()

	fetchOptions := upstream.FetchOptions{}
	fetchOptions.HelmRepoURI = pullOptions.HelmRepoURI
	fetchOptions.LocalPath = pullOptions.LocalPath
	fetchOptions.ExcludeAdminConsole = pullOptions.ExcludeAdminConsole
	fetchOptions.SharedPassword = pullOptions.SharedPassword

	if pullOptions.LicenseFile != "" {
		license, err := parseLicenseFromFile(pullOptions.LicenseFile)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse license from file")
		}

		fetchOptions.License = license
	}

	log.ActionWithSpinner("Pulling upstream")
	u, err := upstream.FetchUpstream(upstreamURI, &fetchOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to fetch upstream")
	}

	writeUpstreamOptions := upstream.WriteOptions{
		RootDir:      pullOptions.RootDir,
		CreateAppDir: true,
	}
	if err := u.WriteUpstream(writeUpstreamOptions); err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to write upstream")
	}
	log.FinishSpinner()

	renderOptions := base.RenderOptions{
		SplitMultiDocYAML: true,
		Namespace:         pullOptions.Namespace,
	}
	log.ActionWithSpinner("Creating base")
	b, err := base.RenderUpstream(u, &renderOptions)
	if err != nil {
		return "", errors.Wrap(err, "failed to render upstream")
	}
	log.FinishSpinner()

	writeBaseOptions := base.WriteOptions{
		BaseDir:          u.GetBaseDir(writeUpstreamOptions),
		Overwrite:        true,
		ExcludeKotsKinds: pullOptions.ExcludeKotsKinds,
	}
	if err := b.WriteBase(writeBaseOptions); err != nil {
		return "", errors.Wrap(err, "failed to write base")
	}

	log.ActionWithSpinner("Creating midstream")
	m, err := midstream.CreateMidstream(b)
	if err != nil {
		return "", errors.Wrap(err, "failed to create midstream")
	}
	log.FinishSpinner()

	writeMidstreamOptions := midstream.WriteOptions{
		MidstreamDir: path.Join(b.GetOverlaysDir(writeBaseOptions), "midstream"),
		BaseDir:      u.GetBaseDir(writeUpstreamOptions),
	}
	if err := m.WriteMidstream(writeMidstreamOptions); err != nil {
		return "", errors.Wrap(err, "failed to write midstream")
	}

	for _, downstreamName := range pullOptions.Downstreams {
		log.ActionWithSpinner("Creating downstream %q", downstreamName)
		d, err := downstream.CreateDownstream(m, downstreamName)
		if err != nil {
			return "", errors.Wrap(err, "failed to create downstream")
		}

		writeDownstreamOptions := downstream.WriteOptions{
			DownstreamDir: path.Join(b.GetOverlaysDir(writeBaseOptions), "downstreams", downstreamName),
			MidstreamDir:  writeMidstreamOptions.MidstreamDir,
		}

		if err := d.WriteDownstream(writeDownstreamOptions); err != nil {
			return "", errors.Wrap(err, "failed to write downstream")
		}

		log.FinishSpinner()
	}

	return path.Join(pullOptions.RootDir, u.Name), nil
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
