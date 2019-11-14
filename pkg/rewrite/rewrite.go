package rewrite

import (
	"io"
	"path/filepath"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/downstream"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	"github.com/replicatedhq/kots/pkg/upstream"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/v3/pkg/image"
)

type RewriteOptions struct {
	RootDir           string
	UpstreamURI       string
	UpstreamPath      string
	LocalCursor       string
	Downstreams       []string
	K8sNamespace      string
	Silent            bool
	CreateAppDir      bool
	ExcludeKotsKinds  bool
	License           *kotsv1beta1.License
	ConfigValues      *kotsv1beta1.ConfigValues
	ReportWriter      io.Writer
	RegistryEndpoint  string
	RegistryUsername  string
	RegistryPassword  string
	RegistryNamespace string
}

func Rewrite(rewriteOptions RewriteOptions) error {
	log := logger.NewLogger()

	if rewriteOptions.Silent {
		log.Silence()
	}

	log.Initialize()

	fetchOptions := &upstream.FetchOptions{
		RootDir:       rewriteOptions.RootDir,
		LocalPath:     rewriteOptions.UpstreamPath,
		CurrentCursor: rewriteOptions.LocalCursor,
		License:       rewriteOptions.License,
	}

	log.ActionWithSpinner("Pulling upstream")
	u, err := upstream.FetchUpstream(rewriteOptions.UpstreamURI, fetchOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to load upstream")
	}

	includeAdminConsole := false

	writeUpstreamOptions := upstream.WriteOptions{
		RootDir:             rewriteOptions.RootDir,
		CreateAppDir:        rewriteOptions.CreateAppDir,
		IncludeAdminConsole: includeAdminConsole,
	}
	if err := u.WriteUpstream(writeUpstreamOptions); err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to write upstream")
	}
	log.FinishSpinner()

	replicatedRegistryInfo := registry.ProxyEndpointFromLicense(rewriteOptions.License)

	var pullSecret *corev1.Secret
	var images []image.Image
	var objects []*k8sdoc.Doc

	// Rewrite all images
	writeUpstreamImageOptions := upstream.WriteUpstreamImageOptions{
		RootDir:      rewriteOptions.RootDir,
		CreateAppDir: rewriteOptions.CreateAppDir,
		ReportWriter: rewriteOptions.ReportWriter,
		Log:          log,
		SourceRegistry: registry.RegistryOptions{
			Endpoint:      replicatedRegistryInfo.Registry,
			ProxyEndpoint: replicatedRegistryInfo.Proxy,
		},
	}
	if fetchOptions.License != nil {
		writeUpstreamImageOptions.AppSlug = fetchOptions.License.Spec.AppSlug
		writeUpstreamImageOptions.SourceRegistry.Username = fetchOptions.License.Spec.LicenseID
		writeUpstreamImageOptions.SourceRegistry.Password = fetchOptions.License.Spec.LicenseID
	}
	if err := u.WriteUpstreamImages(writeUpstreamImageOptions); err != nil {
		return errors.Wrap(err, "failed to write upstream images")
	}

	// If the request includes a rewrite image options host name, then also
	// push the images
	pushUpstreamImageOptions := upstream.PushUpstreamImageOptions{
		RootDir:      rewriteOptions.RootDir,
		ImagesDir:    imagesDirFromOptions(u, rewriteOptions),
		CreateAppDir: rewriteOptions.CreateAppDir,
		Log:          log,
		ReplicatedRegistry: registry.RegistryOptions{
			Endpoint:      replicatedRegistryInfo.Registry,
			ProxyEndpoint: replicatedRegistryInfo.Proxy,
		},
		ReportWriter: rewriteOptions.ReportWriter,
		DestinationRegistry: registry.RegistryOptions{
			Endpoint:  rewriteOptions.RegistryEndpoint,
			Namespace: rewriteOptions.RegistryNamespace,
			Username:  rewriteOptions.RegistryUsername,
			Password:  rewriteOptions.RegistryPassword,
		},
	}
	if fetchOptions.License != nil {
		pushUpstreamImageOptions.ReplicatedRegistry.Username = fetchOptions.License.Spec.LicenseID
		pushUpstreamImageOptions.ReplicatedRegistry.Password = fetchOptions.License.Spec.LicenseID
	}
	rewrittenImages, err := u.TagAndPushUpstreamImages(pushUpstreamImageOptions)
	if err != nil {
		return errors.Wrap(err, "failed to push upstream images")
	}

	findObjectsOptions := upstream.FindObjectsWithImagesOptions{
		RootDir:      rewriteOptions.RootDir,
		CreateAppDir: rewriteOptions.CreateAppDir,
		Log:          log,
	}
	affectedObjects, err := u.FindObjectsWithImages(findObjectsOptions)
	if err != nil {
		return errors.Wrap(err, "failed to find objects with images")
	}

	registryUser := rewriteOptions.RegistryUsername
	registryPass := rewriteOptions.RegistryPassword
	if registryUser == "" {
		// this will only work when envoked from CLI
		registryUser, registryPass, err = registry.LoadAuthForRegistry(rewriteOptions.RegistryEndpoint)
		if err != nil {
			return errors.Wrapf(err, "failed to load registry auth for %q", rewriteOptions.RegistryEndpoint)
		}
	}

	pullSecret, err = registry.PullSecretForRegistries(
		[]string{rewriteOptions.RegistryEndpoint},
		registryUser,
		registryPass,
		rewriteOptions.K8sNamespace,
	)
	if err != nil {
		return errors.Wrap(err, "create pull secret")
	}
	images = rewrittenImages
	objects = affectedObjects

	renderOptions := base.RenderOptions{
		SplitMultiDocYAML: true,
		Namespace:         rewriteOptions.K8sNamespace,
		Log:               log,
	}
	log.ActionWithSpinner("Creating base")
	b, err := base.RenderUpstream(u, &renderOptions)
	if err != nil {
		return errors.Wrap(err, "failed to render upstream")
	}
	log.FinishSpinner()

	writeBaseOptions := base.WriteOptions{
		BaseDir:          u.GetBaseDir(writeUpstreamOptions),
		Overwrite:        true,
		ExcludeKotsKinds: rewriteOptions.ExcludeKotsKinds,
	}
	if err := b.WriteBase(writeBaseOptions); err != nil {
		return errors.Wrap(err, "failed to write base")
	}

	log.ActionWithSpinner("Creating midstream")

	m, err := midstream.CreateMidstream(b, images, objects, pullSecret)
	if err != nil {
		return errors.Wrap(err, "failed to create midstream")
	}
	log.FinishSpinner()

	writeMidstreamOptions := midstream.WriteOptions{
		MidstreamDir: filepath.Join(b.GetOverlaysDir(writeBaseOptions), "midstream"),
		BaseDir:      u.GetBaseDir(writeUpstreamOptions),
	}
	if err := m.WriteMidstream(writeMidstreamOptions); err != nil {
		return errors.Wrap(err, "failed to write midstream")
	}

	for _, downstreamName := range rewriteOptions.Downstreams {
		log.ActionWithSpinner("Creating downstream %q", downstreamName)
		d, err := downstream.CreateDownstream(m, downstreamName)
		if err != nil {
			return errors.Wrap(err, "failed to create downstream")
		}

		writeDownstreamOptions := downstream.WriteOptions{
			DownstreamDir: filepath.Join(b.GetOverlaysDir(writeBaseOptions), "downstreams", downstreamName),
			MidstreamDir:  writeMidstreamOptions.MidstreamDir,
		}
		if err := d.WriteDownstream(writeDownstreamOptions); err != nil {
			return errors.Wrap(err, "failed to write downstream")
		}

		log.FinishSpinner()
	}

	return nil
}

func imagesDirFromOptions(upstream *upstream.Upstream, rewriteOptions RewriteOptions) string {
	if rewriteOptions.CreateAppDir {
		return filepath.Join(rewriteOptions.RootDir, upstream.Name, "images")
	}

	return filepath.Join(rewriteOptions.RootDir, "images")
}
