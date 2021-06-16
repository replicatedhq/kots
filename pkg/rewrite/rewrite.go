package rewrite

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
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
	NativeHelmInstall  bool
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
		IsOpenShift:          k8sutil.IsOpenShift(clientset),
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

	builder, err := base.NewConfigContextTemplateBuidler(u, &renderOptions)
	if err != nil {
		return errors.Wrap(err, "failed to create new config context template builder")
	}

	cipher, err := crypto.AESCipherFromString(rewriteOptions.Installation.Spec.EncryptionKey)
	if err != nil {
		return errors.Wrap(err, "failed to create cipher from installation spec")
	}

	commonWriteMidstreamOptions := midstream.WriteOptions{
		AppSlug:            rewriteOptions.AppSlug,
		IsGitOps:           rewriteOptions.IsGitOps,
		IsOpenShift:        k8sutil.IsOpenShift(clientset),
		Cipher:             *cipher,
		Builder:            *builder,
		HTTPProxyEnvValue:  rewriteOptions.HTTPProxyEnvValue,
		HTTPSProxyEnvValue: rewriteOptions.HTTPSProxyEnvValue,
		NoProxyEnvValue:    rewriteOptions.NoProxyEnvValue,
	}

	var commonBase *base.Base
	var m *midstream.Midstream
	helmMidstreams := []*midstream.Midstream{}
	if rewriteOptions.NativeHelmInstall {
		for _, base := range b.Bases {
			if base.Path == "common" {
				b := base
				commonBase = &b
			} else {
				helmMidstream, err := midstream.CreateMidstream(&base, images, objects, nil, nil, nil)
				if err != nil {
					return errors.Wrapf(err, "failed to create helm midstream %s", base.Path)
				}

				writeMidstreamOptions := commonWriteMidstreamOptions
				writeMidstreamOptions.MidstreamDir = filepath.Join(b.GetOverlaysDir(writeBaseOptions), "midstream", base.Path)
				writeMidstreamOptions.BaseDir = filepath.Join(u.GetBaseDir(writeUpstreamOptions), base.Path)
				if err := helmMidstream.WriteMidstream(writeMidstreamOptions); err != nil {
					return errors.Wrapf(err, "failed to write helm midstream %s", base.Path)
				}

				helmMidstreams = append(helmMidstreams, helmMidstream)
			}
		}
	} else {
		commonBase = b
	}

	if commonBase == nil {
		return errors.New("failed to find common base")
	}

	m, err = midstream.CreateMidstream(commonBase, images, objects, pullSecret, identitySpec, identityConfig)
	if err != nil {
		return errors.Wrap(err, "failed to create midstream")
	}

	writeMidstreamOptions := commonWriteMidstreamOptions
	writeMidstreamOptions.MidstreamDir = filepath.Join(b.GetOverlaysDir(writeBaseOptions), "midstream")
	writeMidstreamOptions.BaseDir = filepath.Join(u.GetBaseDir(writeUpstreamOptions), commonBase.Path)
	fmt.Println("+++ writeMidstreamOptions.BaseDir", writeMidstreamOptions.MidstreamDir, writeMidstreamOptions.BaseDir)
	if err := m.WriteMidstream(writeMidstreamOptions); err != nil {
		return errors.Wrap(err, "failed to write common midstream")
	}

	log.FinishSpinner()

	if err := writeDownstreams(rewriteOptions, b.GetOverlaysDir(writeBaseOptions), m, helmMidstreams, log); err != nil {
		return errors.Wrap(err, "failed to write downstreams")
	}

	return nil
}

func writeDownstreams(options RewriteOptions, overlaysDir string, m *midstream.Midstream, helmMidstreams []*midstream.Midstream, log *logger.CLILogger) error {
	for _, downstreamName := range options.Downstreams {
		log.ActionWithSpinner("Creating downstream %q", downstreamName)
		io.WriteString(options.ReportWriter, fmt.Sprintf("Creating downstream %q\n", downstreamName))

		d, err := downstream.CreateDownstream(m)
		if err != nil {
			return errors.Wrapf(err, "failed to create downstream %s", downstreamName)
		}

		writeDownstreamOptions := downstream.WriteOptions{
			DownstreamDir: filepath.Join(overlaysDir, "downstreams", downstreamName),
			MidstreamDir:  filepath.Join(overlaysDir, "midstream"),
		}
		if err := d.WriteDownstream(writeDownstreamOptions); err != nil {
			return errors.Wrapf(err, "failed to write downstream %s", downstreamName)
		}

		if options.NativeHelmInstall {
			combinedDownstreamBases := []string{"../"}

			for _, mid := range helmMidstreams {
				d, err := downstream.CreateDownstream(mid)
				if err != nil {
					return errors.Wrapf(err, "failed to create downstream %s for midstream %s", downstreamName, mid.Base.Path)
				}

				writeDownstreamOptions := downstream.WriteOptions{
					DownstreamDir: filepath.Join(overlaysDir, "downstreams", downstreamName, mid.Base.Path),
					MidstreamDir:  filepath.Join(overlaysDir, "midstream", mid.Base.Path),
				}
				if err := d.WriteDownstream(writeDownstreamOptions); err != nil {
					return errors.Wrapf(err, "failed to write downstream %s for midstream %s", downstreamName, mid.Base.Path)
				}

				combinedDownstreamBases = append(combinedDownstreamBases, path.Join("..", mid.Base.Path))
			}

			if err := writeCombinedDownstreamBase(downstreamName, combinedDownstreamBases, filepath.Join(overlaysDir, "downstreams", downstreamName, "combined")); err != nil {
				return errors.Wrap(err, "failed to write combined downstream base")
			}
		}

		log.FinishSpinner()
	}

	return nil
}

func writeCombinedDownstreamBase(downstreamName string, bases []string, renderDir string) error {
	if _, err := os.Stat(renderDir); os.IsNotExist(err) {
		if err := os.MkdirAll(renderDir, 0744); err != nil {
			return errors.Wrap(err, "failed to mkdir")
		}
	}

	kustomization := kustomizetypes.Kustomization{
		TypeMeta: kustomizetypes.TypeMeta{
			APIVersion: "kustomize.config.k8s.io/v1beta1",
			Kind:       "Kustomization",
		},
		Bases: bases,
	}
	if err := k8sutil.WriteKustomizationToFile(kustomization, filepath.Join(renderDir, "kustomization.yaml")); err != nil {
		return errors.Wrap(err, "failed to write kustomization to file")
	}

	return nil
}
