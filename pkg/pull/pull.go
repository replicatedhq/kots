package pull

import (
	"io"
	"io/ioutil"
	"net/url"
	"os"
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
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/kustomize/v3/pkg/image"
)

type PullOptions struct {
	HelmRepoURI         string
	RootDir             string
	Namespace           string
	Downstreams         []string
	LocalPath           string
	LicenseFile         string
	InstallationFile    string
	AirgapRoot          string
	ConfigFile          string
	UpdateCursor        string
	ExcludeKotsKinds    bool
	ExcludeAdminConsole bool
	SharedPassword      string
	CreateAppDir        bool
	Silent              bool
	RewriteImages       bool
	RewriteImageOptions RewriteImageOptions
	HelmOptions         []string
	ReportWriter        io.Writer
}

type RewriteImageOptions struct {
	ImageFiles string
	Host       string
	Namespace  string
	Username   string
	Password   string
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
// is accessible and authenticated for us.
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

	if pullOptions.Silent {
		log.Silence()
	}

	log.Initialize()

	uri, err := url.ParseRequestURI(upstreamURI)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse uri")
	}

	fetchOptions := upstream.FetchOptions{}
	fetchOptions.HelmRepoURI = pullOptions.HelmRepoURI
	fetchOptions.RootDir = pullOptions.RootDir
	fetchOptions.UseAppDir = pullOptions.CreateAppDir
	fetchOptions.LocalPath = pullOptions.LocalPath
	fetchOptions.CurrentCursor = pullOptions.UpdateCursor

	if pullOptions.LicenseFile != "" {
		license, err := parseLicenseFromFile(pullOptions.LicenseFile)
		if err != nil {
			if errors.Cause(err) == ErrSignatureInvalid {
				return "", ErrSignatureInvalid
			}
			if errors.Cause(err) == ErrSignatureMissing {
				return "", ErrSignatureMissing
			}
			return "", errors.Wrap(err, "failed to parse license from file")
		}

		fetchOptions.License = license
	}
	if pullOptions.ConfigFile != "" {
		config, err := parseConfigValuesFromFile(pullOptions.ConfigFile)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse license from file")
		}
		fetchOptions.ConfigValues = config
	}
	if pullOptions.InstallationFile != "" {
		installation, err := parseInstallationFromFile(pullOptions.InstallationFile)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse installation from file")
		}
		if installation != nil {
			fetchOptions.EncryptionKey = installation.Spec.EncryptionKey
		}
	}

	if pullOptions.AirgapRoot != "" {
		airgap, err := findAirgapMetaInDir(pullOptions.AirgapRoot)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse license from file")
		}

		if err := publicKeysMatch(fetchOptions.License, airgap); err != nil {
			return "", errors.Wrap(err, "failed to validate app key")
		}

		fetchOptions.Airgap = airgap
	}

	log.ActionWithSpinner("Pulling upstream")
	u, err := upstream.FetchUpstream(upstreamURI, &fetchOptions)
	if err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to fetch upstream")
	}

	includeAdminConsole := uri.Scheme == "replicated" && !pullOptions.ExcludeAdminConsole

	writeUpstreamOptions := upstreamtypes.WriteOptions{
		RootDir:             pullOptions.RootDir,
		CreateAppDir:        pullOptions.CreateAppDir,
		IncludeAdminConsole: includeAdminConsole,
		SharedPassword:      pullOptions.SharedPassword,
	}
	if err := upstream.WriteUpstream(u, writeUpstreamOptions); err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to write upstream")
	}
	log.FinishSpinner()

	replicatedRegistryInfo := registry.ProxyEndpointFromLicense(fetchOptions.License)

	var pullSecret *corev1.Secret
	var images []image.Image
	var objects []*k8sdoc.Doc
	if pullOptions.RewriteImages {

		// Rewrite all images
		if pullOptions.RewriteImageOptions.ImageFiles == "" {
			writeUpstreamImageOptions := upstream.WriteUpstreamImageOptions{
				RootDir:      pullOptions.RootDir,
				CreateAppDir: pullOptions.CreateAppDir,
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

			if pullOptions.RewriteImageOptions.Host != "" {
				writeUpstreamImageOptions.DestRegistry = registry.RegistryOptions{
					Endpoint:  pullOptions.RewriteImageOptions.Host,
					Namespace: pullOptions.RewriteImageOptions.Namespace,
					Username:  pullOptions.RewriteImageOptions.Username,
					Password:  pullOptions.RewriteImageOptions.Password,
				}
			}

			newImages, err := upstream.CopyUpstreamImages(u, writeUpstreamImageOptions)
			if err != nil {
				return "", errors.Wrap(err, "failed to write upstream images")
			}
			images = newImages
		}

		// If the request includes a rewrite image options host name, then also
		// push the images
		if pullOptions.RewriteImageOptions.Host != "" {
			pushUpstreamImageOptions := upstream.PushUpstreamImageOptions{
				RootDir:      pullOptions.RootDir,
				ImagesDir:    imagesDirFromOptions(u, pullOptions),
				CreateAppDir: pullOptions.CreateAppDir,
				Log:          log,
				ReplicatedRegistry: registry.RegistryOptions{
					Endpoint:      replicatedRegistryInfo.Registry,
					ProxyEndpoint: replicatedRegistryInfo.Proxy,
				},
				ReportWriter: pullOptions.ReportWriter,
				DestinationRegistry: registry.RegistryOptions{
					Endpoint:  pullOptions.RewriteImageOptions.Host,
					Namespace: pullOptions.RewriteImageOptions.Namespace,
					Username:  pullOptions.RewriteImageOptions.Username,
					Password:  pullOptions.RewriteImageOptions.Password,
				},
			}
			if fetchOptions.License != nil {
				pushUpstreamImageOptions.ReplicatedRegistry.Username = fetchOptions.License.Spec.LicenseID
				pushUpstreamImageOptions.ReplicatedRegistry.Password = fetchOptions.License.Spec.LicenseID
			}

			// only run the TagAndPushImagesFromFiles code if the "copy directly" code hasn't already run
			var rewrittenImages []image.Image
			if images == nil {
				rewrittenImages, err = upstream.TagAndPushUpstreamImages(u, pushUpstreamImageOptions)
				if err != nil {
					return "", errors.Wrap(err, "failed to push upstream images")
				}
			}

			findObjectsOptions := upstream.FindObjectsWithImagesOptions{
				RootDir:      pullOptions.RootDir,
				CreateAppDir: pullOptions.CreateAppDir,
				Log:          log,
			}
			affectedObjects, err := upstream.FindObjectsWithImages(u, findObjectsOptions)
			if err != nil {
				return "", errors.Wrap(err, "failed to find objects with images")
			}

			registryUser := pullOptions.RewriteImageOptions.Username
			registryPass := pullOptions.RewriteImageOptions.Password
			if registryUser == "" {
				registryUser, registryPass, err = registry.LoadAuthForRegistry(pullOptions.RewriteImageOptions.Host)
				if err != nil {
					return "", errors.Wrapf(err, "failed to load registry auth for %q", pullOptions.RewriteImageOptions.Host)
				}
			}

			pullSecret, err = registry.PullSecretForRegistries(
				[]string{pullOptions.RewriteImageOptions.Host},
				registryUser,
				registryPass,
				pullOptions.Namespace,
			)
			if err != nil {
				return "", errors.Wrap(err, "create pull secret")
			}

			if rewrittenImages != nil {
				images = rewrittenImages
			}
			objects = affectedObjects
		}
	} else if fetchOptions.License != nil {

		// Rewrite private images
		findPrivateImagesOptions := upstream.FindPrivateImagesOptions{
			RootDir:      pullOptions.RootDir,
			CreateAppDir: pullOptions.CreateAppDir,
			AppSlug:      fetchOptions.License.Spec.AppSlug,
			ReplicatedRegistry: registry.RegistryOptions{
				Endpoint:      replicatedRegistryInfo.Registry,
				ProxyEndpoint: replicatedRegistryInfo.Proxy,
			},
			Log: log,
		}
		rewrittenImages, affectedObjects, err := upstream.FindPrivateImages(u, findPrivateImagesOptions)
		if err != nil {
			return "", errors.Wrap(err, "failed to push upstream images")
		}

		// Note that there maybe no rewritten images if only replicated private images are being used.
		// We still need to create the secret in that case.
		if len(affectedObjects) > 0 {
			pullSecret, err = registry.PullSecretForRegistries(
				replicatedRegistryInfo.ToSlice(),
				fetchOptions.License.Spec.LicenseID,
				fetchOptions.License.Spec.LicenseID,
				pullOptions.Namespace,
			)
			if err != nil {
				return "", errors.Wrap(err, "create pull secret")
			}
		}
		images = rewrittenImages
		objects = affectedObjects
	}

	renderOptions := base.RenderOptions{
		SplitMultiDocYAML: true,
		Namespace:         pullOptions.Namespace,
		HelmOptions:       pullOptions.HelmOptions,
		Log:               log,
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

	m, err := midstream.CreateMidstream(b, images, objects, pullSecret)
	if err != nil {
		return "", errors.Wrap(err, "failed to create midstream")
	}
	log.FinishSpinner()

	writeMidstreamOptions := midstream.WriteOptions{
		MidstreamDir: filepath.Join(b.GetOverlaysDir(writeBaseOptions), "midstream"),
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
			DownstreamDir: filepath.Join(b.GetOverlaysDir(writeBaseOptions), "downstreams", downstreamName),
			MidstreamDir:  writeMidstreamOptions.MidstreamDir,
		}

		if err := d.WriteDownstream(writeDownstreamOptions); err != nil {
			return "", errors.Wrap(err, "failed to write downstream")
		}

		log.FinishSpinner()
	}

	if includeAdminConsole {
		if err := writeArchiveAsConfigMap(pullOptions, u, u.GetBaseDir(writeUpstreamOptions)); err != nil {
			return "", errors.Wrap(err, "failed to write archive as config map")
		}
	}

	return filepath.Join(pullOptions.RootDir, u.Name), nil
}

func parseLicenseFromFile(filename string) (*kotsv1beta1.License, error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read license file")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	decoded, gvk, err := decode(contents, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode license file")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "License" {
		return nil, errors.New("not an application license")
	}

	license := decoded.(*kotsv1beta1.License)
	verifiedLicense, err := VerifySignature(license)
	if err != nil {
		return nil, errors.Wrap(err, "failed to verify signature")
	}

	return verifiedLicense, nil
}

func parseConfigValuesFromFile(filename string) (*kotsv1beta1.ConfigValues, error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to read config values file")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	decoded, gvk, err := decode(contents, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode config values file")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "ConfigValues" {
		return nil, errors.New("not config values")
	}

	config := decoded.(*kotsv1beta1.ConfigValues)

	return config, nil
}

func parseInstallationFromFile(filename string) (*kotsv1beta1.Installation, error) {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to read installation file")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	decoded, gvk, err := decode(contents, nil, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode installation file")
	}

	if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Installation" {
		return nil, errors.New("not installation file")
	}

	installation := decoded.(*kotsv1beta1.Installation)

	return installation, nil
}

func findAirgapMetaInDir(root string) (*kotsv1beta1.Airgap, error) {
	files, err := ioutil.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to read airgap directory content")
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		contents, err := ioutil.ReadFile(filepath.Join(root, file.Name()))
		if err != nil {
			// TODO: log
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		decoded, gvk, err := decode(contents, nil, nil)
		if err != nil {
			// TODO: log
			continue
		}

		if gvk.Group != "kots.io" || gvk.Version != "v1beta1" || gvk.Kind != "Airgap" {
			continue
		}

		airgap := decoded.(*kotsv1beta1.Airgap)
		return airgap, nil
	}

	return nil, nil
}

func imagesDirFromOptions(upstream *upstreamtypes.Upstream, pullOptions PullOptions) string {
	if pullOptions.RewriteImageOptions.ImageFiles != "" {
		return pullOptions.RewriteImageOptions.ImageFiles
	}

	if pullOptions.CreateAppDir {
		return filepath.Join(pullOptions.RootDir, upstream.Name, "images")
	}

	return filepath.Join(pullOptions.RootDir, "images")
}

func publicKeysMatch(license *kotsv1beta1.License, airgap *kotsv1beta1.Airgap) error {
	if license == nil || airgap == nil {
		// not sure when this would happen, but earlier logic allows this combinaion
		return nil
	}

	publicKey, err := GetAppPublicKey(license)
	if err != nil {
		return errors.Wrap(err, "failed to get public key from license")
	}

	if err := verify([]byte(license.Spec.AppSlug), []byte(airgap.Spec.Signature), publicKey); err != nil {
		return errors.Wrap(err, "failed to verify bundle signature")
	}

	return nil
}
