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
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/kustomize/v3/pkg/image"
)

type RewriteOptions struct {
	RootDir           string
	UpstreamURI       string
	UpstreamPath      string
	Downstreams       []string
	K8sNamespace      string
	Silent            bool
	CreateAppDir      bool
	ExcludeKotsKinds  bool
	Installation      *kotsv1beta1.Installation
	License           *kotsv1beta1.License
	ConfigValues      *kotsv1beta1.ConfigValues
	ReportWriter      io.Writer
	CopyImages        bool
	IsAirgap          bool
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
		RootDir:             rewriteOptions.RootDir,
		LocalPath:           rewriteOptions.UpstreamPath,
		CurrentCursor:       rewriteOptions.Installation.Spec.UpdateCursor,
		CurrentVersionLabel: rewriteOptions.Installation.Spec.VersionLabel,
		EncryptionKey:       rewriteOptions.Installation.Spec.EncryptionKey,
		License:             rewriteOptions.License,
	}

	log.ActionWithSpinner("Pulling upstream")
	u, err := upstream.FetchUpstream(rewriteOptions.UpstreamURI, fetchOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to load upstream")
	}

	includeAdminConsole := false

	writeUpstreamOptions := upstreamtypes.WriteOptions{
		RootDir:             rewriteOptions.RootDir,
		CreateAppDir:        rewriteOptions.CreateAppDir,
		IncludeAdminConsole: includeAdminConsole,
	}
	if err := upstream.WriteUpstream(u, writeUpstreamOptions); err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to write upstream")
	}
	log.FinishSpinner()

	replicatedRegistryInfo := registry.ProxyEndpointFromLicense(rewriteOptions.License)

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

	var pullSecret *corev1.Secret
	var images []image.Image
	var objects []*k8sdoc.Doc

	if rewriteOptions.CopyImages || rewriteOptions.RegistryEndpoint != "" {
		// When CopyImages is set, we copy images, rewrite all images, and use registry
		// settings to create secrets for all objects that have images.
		// When only registry endpoint is set, we don't need to copy images, but still
		// need to rewrite them and create secrets.
		writeUpstreamImageOptions := base.WriteUpstreamImageOptions{
			BaseDir:      writeBaseOptions.BaseDir,
			ReportWriter: rewriteOptions.ReportWriter,
			Log:          log,
			SourceRegistry: registry.RegistryOptions{
				Endpoint:      replicatedRegistryInfo.Registry,
				ProxyEndpoint: replicatedRegistryInfo.Proxy,
			},
			DestRegistry: registry.RegistryOptions{
				Endpoint:  rewriteOptions.RegistryEndpoint,
				Namespace: rewriteOptions.RegistryNamespace,
				Username:  rewriteOptions.RegistryUsername,
				Password:  rewriteOptions.RegistryPassword,
			},
			DryRun:   !rewriteOptions.CopyImages,
			IsAirgap: rewriteOptions.IsAirgap,
		}
		if fetchOptions.License != nil {
			writeUpstreamImageOptions.AppSlug = fetchOptions.License.Spec.AppSlug
			writeUpstreamImageOptions.SourceRegistry.Username = fetchOptions.License.Spec.LicenseID
			writeUpstreamImageOptions.SourceRegistry.Password = fetchOptions.License.Spec.LicenseID
		}

		rewrittenImages, err := base.CopyUpstreamImages(writeUpstreamImageOptions)
		if err != nil {
			return errors.Wrap(err, "failed to write upstream images")
		}

		findObjectsOptions := base.FindObjectsWithImagesOptions{
			BaseDir: writeBaseOptions.BaseDir,
		}
		affectedObjects, err := base.FindObjectsWithImages(findObjectsOptions)
		if err != nil {
			return errors.Wrap(err, "failed to find objects with images")
		}

		registryUser := rewriteOptions.RegistryUsername
		registryPass := rewriteOptions.RegistryPassword
		if registryUser == "" {
			// this will only work when envoked from CLI where `docker login` command has been executed
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
			return errors.Wrap(err, "failed to create private registry pull secret")
		}

		images = rewrittenImages
		objects = affectedObjects
	} else {
		// When CopyImages is not set, we only rewrite private images and use license to create secrets
		// for all objects that have private images
		findPrivateImagesOptions := base.FindPrivateImagesOptions{
			BaseDir: writeBaseOptions.BaseDir,
			AppSlug: fetchOptions.License.Spec.AppSlug,
			ReplicatedRegistry: registry.RegistryOptions{
				Endpoint:      replicatedRegistryInfo.Registry,
				ProxyEndpoint: replicatedRegistryInfo.Proxy,
			},
		}
		rewrittenImages, affectedObjects, err := base.FindPrivateImages(findPrivateImagesOptions)
		if err != nil {
			return errors.Wrap(err, "failed to find private images")
		}

		if len(affectedObjects) > 0 {
			replicatedRegistryInfo := registry.ProxyEndpointFromLicense(rewriteOptions.License)
			pullSecret, err = registry.PullSecretForRegistries(
				replicatedRegistryInfo.ToSlice(),
				rewriteOptions.License.Spec.LicenseID,
				rewriteOptions.License.Spec.LicenseID,
				rewriteOptions.K8sNamespace,
			)
			if err != nil {
				return errors.Wrap(err, "failed to create Replicated registry pull secret")
			}
		}

		images = rewrittenImages
		objects = affectedObjects
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

func imagesDirFromOptions(upstream *upstreamtypes.Upstream, rewriteOptions RewriteOptions) string {
	if rewriteOptions.CreateAppDir {
		return filepath.Join(rewriteOptions.RootDir, upstream.Name, "images")
	}

	return filepath.Join(rewriteOptions.RootDir, "images")
}
