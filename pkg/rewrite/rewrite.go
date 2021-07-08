package rewrite

import (
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	reportingtypes "github.com/replicatedhq/kots/pkg/api/reporting/types"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/crypto"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/downstream"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	"github.com/replicatedhq/kots/pkg/upstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	corev1 "k8s.io/api/core/v1"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

type RewriteOptions struct {
	RootDir            string
	UpstreamURI        string
	UpstreamPath       string
	Downstreams        []string
	K8sNamespace       string
	Silent             bool
	CreateAppDir       bool
	ExcludeKotsKinds   bool
	Installation       *kotsv1beta1.Installation
	License            *kotsv1beta1.License
	ConfigValues       *kotsv1beta1.ConfigValues
	ReportWriter       io.Writer
	CopyImages         bool // can be false even if registry is not read-only
	IsAirgap           bool
	RegistryEndpoint   string
	RegistryUsername   string
	RegistryPassword   string
	RegistryNamespace  string
	RegistryIsReadOnly bool
	AppSlug            string
	IsGitOps           bool
	AppSequence        int64
	ReportingInfo      *reportingtypes.ReportingInfo
	HTTPProxyEnvValue  string
	HTTPSProxyEnvValue string
	NoProxyEnvValue    string
}

func Rewrite(rewriteOptions RewriteOptions) error {
	log := logger.NewCLILogger()

	if rewriteOptions.Silent {
		log.Silence()
	}

	log.Initialize()

	if rewriteOptions.ReportWriter == nil {
		rewriteOptions.ReportWriter = ioutil.Discard
	}

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s clientset")
	}

	fetchOptions := &upstreamtypes.FetchOptions{
		RootDir:             rewriteOptions.RootDir,
		LocalPath:           rewriteOptions.UpstreamPath,
		CurrentCursor:       rewriteOptions.Installation.Spec.UpdateCursor,
		CurrentVersionLabel: rewriteOptions.Installation.Spec.VersionLabel,
		EncryptionKey:       rewriteOptions.Installation.Spec.EncryptionKey,
		License:             rewriteOptions.License,
		AppSequence:         rewriteOptions.AppSequence,
		AppSlug:             rewriteOptions.AppSlug,
		LocalRegistry: upstreamtypes.LocalRegistry{
			Host:      rewriteOptions.RegistryEndpoint,
			Namespace: rewriteOptions.RegistryNamespace,
			Username:  rewriteOptions.RegistryUsername,
			Password:  rewriteOptions.RegistryPassword,
			ReadOnly:  rewriteOptions.RegistryIsReadOnly,
		},
		ReportingInfo: rewriteOptions.ReportingInfo,
	}

	log.ActionWithSpinner("Pulling upstream")
	io.WriteString(rewriteOptions.ReportWriter, "Pulling upstream\n")
	u, err := upstream.FetchUpstream(rewriteOptions.UpstreamURI, fetchOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to load upstream")
	}

	includeAdminConsole := false

	writeUpstreamOptions := upstreamtypes.WriteOptions{
		RootDir:              rewriteOptions.RootDir,
		CreateAppDir:         rewriteOptions.CreateAppDir,
		IncludeAdminConsole:  includeAdminConsole,
		PreserveInstallation: true,
	}
	if err := upstream.WriteUpstream(u, writeUpstreamOptions); err != nil {
		log.FinishSpinnerWithError()
		return errors.Wrap(err, "failed to write upstream")
	}
	log.FinishSpinner()

	replicatedRegistryInfo := registry.ProxyEndpointFromLicense(rewriteOptions.License)

	renderOptions := base.RenderOptions{
		SplitMultiDocYAML:       true,
		Namespace:               rewriteOptions.K8sNamespace,
		LocalRegistryHost:       rewriteOptions.RegistryEndpoint,
		LocalRegistryNamespace:  rewriteOptions.RegistryNamespace,
		LocalRegistryUsername:   rewriteOptions.RegistryUsername,
		LocalRegistryPassword:   rewriteOptions.RegistryPassword,
		LocalRegistryIsReadOnly: rewriteOptions.RegistryIsReadOnly,
		ExcludeKotsKinds:        rewriteOptions.ExcludeKotsKinds,
		Log:                     log,
		AppSlug:                 rewriteOptions.AppSlug,
		Sequence:                rewriteOptions.AppSequence,
		IsAirgap:                rewriteOptions.IsAirgap,
	}
	log.ActionWithSpinner("Creating base")
	io.WriteString(rewriteOptions.ReportWriter, "Creating base\n")
	b, err := base.RenderUpstream(u, &renderOptions)
	if err != nil {
		return errors.Wrap(err, "failed to render upstream")
	}

	if ff := b.ListErrorFiles(); len(ff) > 0 {
		files := make([]kotsv1beta1.InstallationYAMLError, 0, len(ff))
		for _, f := range ff {
			file := kotsv1beta1.InstallationYAMLError{
				Path: f.Path,
			}
			if f.Error != nil {
				file.Error = f.Error.Error()
			}
			files = append(files, file)
		}

		newInstallation, err := upstream.LoadInstallation(u.GetUpstreamDir(writeUpstreamOptions))
		if err != nil {
			return errors.Wrap(err, "failed to load installation")
		}
		newInstallation.Spec.YAMLErrors = files

		err = upstream.SaveInstallation(newInstallation, u.GetUpstreamDir(writeUpstreamOptions))
		if err != nil {
			return errors.Wrap(err, "failed to save installation")
		}
	}

	log.FinishSpinner()

	writeBaseOptions := base.WriteOptions{
		BaseDir:          u.GetBaseDir(writeUpstreamOptions),
		SkippedDir:       u.GetSkippedDir(writeUpstreamOptions),
		Overwrite:        true,
		ExcludeKotsKinds: rewriteOptions.ExcludeKotsKinds,
	}
	if err := b.WriteBase(writeBaseOptions); err != nil {
		return errors.Wrap(err, "failed to write base")
	}

	var pullSecret *corev1.Secret
	var images []kustomizetypes.Image
	var objects []k8sdoc.K8sDoc

	identitySpec, err := upstream.LoadIdentity(u.GetUpstreamDir(writeUpstreamOptions))
	if err != nil {
		return errors.Wrap(err, "failed to load identity")
	}

	identityConfig, err := upstream.LoadIdentityConfig(u.GetUpstreamDir(writeUpstreamOptions))
	if err != nil {
		return errors.Wrap(err, "failed to load identity config")
	}

	// do not fail on being unable to get dockerhub credentials, since they're just used to increase the rate limit
	dockerHubRegistryCreds, _ := registry.GetDockerHubCredentials(clientset, rewriteOptions.K8sNamespace)

	// TODO (ethan): rewrite dex image?

	if rewriteOptions.CopyImages || rewriteOptions.RegistryEndpoint != "" {
		// When CopyImages is set, we copy images, rewrite all images, and use registry
		// settings to create secrets for all objects that have images.
		// When only registry endpoint is set, we don't need to copy images, but still
		// need to rewrite them and create secrets.

		newInstallation, err := upstream.LoadInstallation(u.GetUpstreamDir(writeUpstreamOptions))
		if err != nil {
			return errors.Wrap(err, "failed to load installation")
		}
		application, err := upstream.LoadApplication(u.GetUpstreamDir(writeUpstreamOptions))
		if err != nil {
			return errors.Wrap(err, "failed to load application")
		}

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
			DockerHubRegistry: registry.RegistryOptions{
				Username: dockerHubRegistryCreds.Username,
				Password: dockerHubRegistryCreds.Password,
			},
			Installation: newInstallation,
			Application:  application,
			IsAirgap:     rewriteOptions.IsAirgap,
			CopyImages:   rewriteOptions.CopyImages,
		}

		if fetchOptions.License != nil {
			writeUpstreamImageOptions.AppSlug = fetchOptions.License.Spec.AppSlug
			writeUpstreamImageOptions.SourceRegistry.Username = fetchOptions.License.Spec.LicenseID
			writeUpstreamImageOptions.SourceRegistry.Password = fetchOptions.License.Spec.LicenseID
		}

		copyResult, err := base.ProcessUpstreamImages(writeUpstreamImageOptions)
		if err != nil {
			return errors.Wrap(err, "failed to write upstream images")
		}

		newInstallation.Spec.KnownImages = copyResult.CheckedImages
		err = upstream.SaveInstallation(newInstallation, u.GetUpstreamDir(writeUpstreamOptions))
		if err != nil {
			return errors.Wrap(err, "failed to save installation")
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

		images = copyResult.Images
		objects = affectedObjects
	} else {
		application, err := upstream.LoadApplication(u.GetUpstreamDir(writeUpstreamOptions))
		if err != nil {
			return errors.Wrap(err, "failed to load application")
		}

		allPrivate := false
		if application != nil {
			allPrivate = application.Spec.ProxyPublicImages
		}

		// When CopyImages is not set, we only rewrite private images and use license to create secrets
		// for all objects that have private images
		findPrivateImagesOptions := base.FindPrivateImagesOptions{
			BaseDir: writeBaseOptions.BaseDir,
			AppSlug: fetchOptions.License.Spec.AppSlug,
			ReplicatedRegistry: registry.RegistryOptions{
				Endpoint:      replicatedRegistryInfo.Registry,
				ProxyEndpoint: replicatedRegistryInfo.Proxy,
			},
			DockerHubRegistry: registry.RegistryOptions{
				Username: dockerHubRegistryCreds.Username,
				Password: dockerHubRegistryCreds.Password,
			},
			Installation:     rewriteOptions.Installation,
			AllImagesPrivate: allPrivate,
		}
		findResult, err := base.FindPrivateImages(findPrivateImagesOptions)
		if err != nil {
			return errors.Wrap(err, "failed to find private images")
		}

		newInstallation, err := upstream.LoadInstallation(u.GetUpstreamDir(writeUpstreamOptions))
		if err != nil {
			return errors.Wrap(err, "failed to load installation")
		}
		newInstallation.Spec.KnownImages = findResult.CheckedImages
		err = upstream.SaveInstallation(newInstallation, u.GetUpstreamDir(writeUpstreamOptions))
		if err != nil {
			return errors.Wrap(err, "failed to save installation")
		}

		if len(findResult.Docs) > 0 {
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

		images = findResult.Images
		objects = findResult.Docs
	}

	log.ActionWithSpinner("Creating midstream")
	io.WriteString(rewriteOptions.ReportWriter, "Creating midstream\n")

	m, err := midstream.CreateMidstream(b, images, objects, pullSecret, identitySpec, identityConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create midstream")
	}
	log.FinishSpinner()

	builder, _, err := base.NewConfigContextTemplateBuilder(u, &renderOptions)
	if err != nil {
		return errors.Wrap(err, "failed to create new config context template builder")
	}

	cipher, err := crypto.AESCipherFromString(rewriteOptions.Installation.Spec.EncryptionKey)
	if err != nil {
		return errors.Wrap(err, "failed to create cipher from installation spec")
	}

	writeMidstreamOptions := midstream.WriteOptions{
		MidstreamDir:       filepath.Join(b.GetOverlaysDir(writeBaseOptions), "midstream"),
		BaseDir:            u.GetBaseDir(writeUpstreamOptions),
		AppSlug:            rewriteOptions.AppSlug,
		IsGitOps:           rewriteOptions.IsGitOps,
		IsOpenShift:        k8sutil.IsOpenShift(clientset),
		Cipher:             *cipher,
		Builder:            *builder,
		HTTPProxyEnvValue:  rewriteOptions.HTTPProxyEnvValue,
		HTTPSProxyEnvValue: rewriteOptions.HTTPSProxyEnvValue,
		NoProxyEnvValue:    rewriteOptions.NoProxyEnvValue,
	}
	if err := m.WriteMidstream(writeMidstreamOptions); err != nil {
		return errors.Wrap(err, "failed to write midstream")
	}

	for _, downstreamName := range rewriteOptions.Downstreams {
		log.ActionWithSpinner("Creating downstream %q", downstreamName)
		io.WriteString(rewriteOptions.ReportWriter, fmt.Sprintf("Creating downstream %q\n", downstreamName))
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
