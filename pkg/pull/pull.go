package pull

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/archives"
	"github.com/replicatedhq/kots/pkg/base"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/downstream"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/midstream"
	"github.com/replicatedhq/kots/pkg/upstream"
	upstreamtypes "github.com/replicatedhq/kots/pkg/upstream/types"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	k8sjson "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

type PullOptions struct {
	HelmRepoURI         string
	RootDir             string
	Namespace           string
	Downstreams         []string
	LocalPath           string
	LicenseObj          *kotsv1beta1.License
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
	HelmVersion         string
	HelmOptions         []string
	ReportWriter        io.Writer
	AppSlug             string
	AppSequence         int64
	IsGitOps            bool
	HTTPProxyEnvValue   string
	HTTPSProxyEnvValue  string
	NoProxyEnvValue     string
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

// Pull will download the application specified in upstreamURI using the options
// specified in pullOptions. It returns the directory that the app was pulled to
func Pull(upstreamURI string, pullOptions PullOptions) (string, error) {
	log := logger.NewLogger()

	if pullOptions.Silent {
		log.Silence()
	}

	log.Initialize()

	if pullOptions.ReportWriter == nil {
		pullOptions.ReportWriter = ioutil.Discard
	}

	uri, err := url.ParseRequestURI(upstreamURI)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse uri")
	}

	fetchOptions := upstreamtypes.FetchOptions{
		HelmRepoURI:   pullOptions.HelmRepoURI,
		RootDir:       pullOptions.RootDir,
		UseAppDir:     pullOptions.CreateAppDir,
		LocalPath:     pullOptions.LocalPath,
		CurrentCursor: pullOptions.UpdateCursor,
		AppSequence:   pullOptions.AppSequence,
		LocalRegistry: upstreamtypes.LocalRegistry{
			Host:      pullOptions.RewriteImageOptions.Host,
			Namespace: pullOptions.RewriteImageOptions.Namespace,
			Username:  pullOptions.RewriteImageOptions.Username,
			Password:  pullOptions.RewriteImageOptions.Password,
		},
	}

	var installation *kotsv1beta1.Installation

	_, localConfigValues, localLicense, localInstallation, err := findConfig(pullOptions.LocalPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to find config files in local path")
	}

	if pullOptions.LicenseObj != nil {
		fetchOptions.License = pullOptions.LicenseObj
	} else if pullOptions.LicenseFile != "" {
		license, err := ParseLicenseFromFile(pullOptions.LicenseFile)
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
	} else {
		fetchOptions.License = localLicense
	}

	encryptConfig := false
	if pullOptions.ConfigFile != "" {
		config, err := ParseConfigValuesFromFile(pullOptions.ConfigFile)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse config values from file")
		}
		fetchOptions.ConfigValues = config
		encryptConfig = true
	} else {
		fetchOptions.ConfigValues = localConfigValues
	}

	if pullOptions.InstallationFile != "" {
		i, err := parseInstallationFromFile(pullOptions.InstallationFile)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse installation from file")
		}
		installation = i
	} else {
		installation = localInstallation
	}

	if installation != nil {
		fetchOptions.EncryptionKey = installation.Spec.EncryptionKey
		fetchOptions.CurrentVersionLabel = installation.Spec.VersionLabel
		fetchOptions.CurrentChannelID = installation.Spec.ChannelID
		fetchOptions.CurrentChannelName = installation.Spec.ChannelName
		if fetchOptions.CurrentCursor == "" {
			fetchOptions.CurrentCursor = installation.Spec.UpdateCursor
		}
	}

	if pullOptions.AirgapRoot != "" {
		if expired, err := LicenseIsExpired(fetchOptions.License); err != nil {
			return "", errors.Wrap(err, "failed to check license expiration")
		} else if expired {
			return "", util.ActionableError{Message: "License is expired"}
		}

		airgap, err := findAirgapMetaInDir(pullOptions.AirgapRoot)
		if err != nil {
			return "", errors.Wrap(err, "failed to parse license from file")
		}

		if err := publicKeysMatch(fetchOptions.License, airgap); err != nil {
			return "", errors.Wrap(err, "failed to validate app key")
		}

		airgapAppFiles, err := ioutil.TempDir("", "airgap-kots")
		if err != nil {
			return "", errors.Wrap(err, "failed to create temp airgap dir")
		}
		defer os.RemoveAll(airgapAppFiles)

		err = util.ExtractTGZArchive(filepath.Join(pullOptions.AirgapRoot, "app.tar.gz"), airgapAppFiles)
		if err != nil {
			return "", errors.Wrap(err, "failed to extract app files")
		}

		fetchOptions.Airgap = airgap
		fetchOptions.LocalPath = airgapAppFiles
	}

	log.ActionWithSpinner("Pulling upstream")
	io.WriteString(pullOptions.ReportWriter, "Pulling upstream\n")
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
		EncryptConfig:       encryptConfig,
		HTTPProxyEnvValue:   pullOptions.HTTPProxyEnvValue,
		HTTPSProxyEnvValue:  pullOptions.HTTPSProxyEnvValue,
		NoProxyEnvValue:     pullOptions.NoProxyEnvValue,
	}
	if err := upstream.WriteUpstream(u, writeUpstreamOptions); err != nil {
		log.FinishSpinnerWithError()
		return "", errors.Wrap(err, "failed to write upstream")
	}
	log.FinishSpinner()

	replicatedRegistryInfo := registry.ProxyEndpointFromLicense(fetchOptions.License)

	renderOptions := base.RenderOptions{
		SplitMultiDocYAML:      true,
		Namespace:              pullOptions.Namespace,
		HelmVersion:            pullOptions.HelmVersion,
		HelmOptions:            pullOptions.HelmOptions,
		LocalRegistryHost:      pullOptions.RewriteImageOptions.Host,
		LocalRegistryNamespace: pullOptions.RewriteImageOptions.Namespace,
		LocalRegistryUsername:  pullOptions.RewriteImageOptions.Username,
		LocalRegistryPassword:  pullOptions.RewriteImageOptions.Password,
		ExcludeKotsKinds:       pullOptions.ExcludeKotsKinds,
		Log:                    log,
		Sequence:               pullOptions.AppSequence,
		IsAirgap:               pullOptions.AirgapRoot != "",
	}
	log.ActionWithSpinner("Creating base")
	io.WriteString(pullOptions.ReportWriter, "Creating base\n")

	b, err := base.RenderUpstream(u, &renderOptions)
	if err != nil {
		return "", errors.Wrap(err, "failed to render upstream")
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
			return "", errors.Wrap(err, "failed to load installation")
		}
		newInstallation.Spec.YAMLErrors = files

		err = upstream.SaveInstallation(newInstallation, u.GetUpstreamDir(writeUpstreamOptions))
		if err != nil {
			return "", errors.Wrap(err, "failed to save installation")
		}
	}

	log.FinishSpinner()

	writeBaseOptions := base.WriteOptions{
		BaseDir:          u.GetBaseDir(writeUpstreamOptions),
		SkippedDir:       u.GetSkippedDir(writeUpstreamOptions),
		Overwrite:        true,
		ExcludeKotsKinds: pullOptions.ExcludeKotsKinds,
	}
	if err := b.WriteBase(writeBaseOptions); err != nil {
		return "", errors.Wrap(err, "failed to write base")
	}

	var pullSecret *corev1.Secret
	var images []kustomizetypes.Image
	var objects []k8sdoc.K8sDoc
	if pullOptions.RewriteImages {

		log.ActionWithSpinner("Copying private images")
		io.WriteString(pullOptions.ReportWriter, "Copying private images\n")

		// Rewrite all images
		if pullOptions.RewriteImageOptions.ImageFiles == "" {
			newInstallation, err := upstream.LoadInstallation(u.GetUpstreamDir(writeUpstreamOptions))
			if err != nil {
				return "", errors.Wrap(err, "failed to load installation")
			}
			newApplication, err := upstream.LoadApplication(u.GetUpstreamDir(writeUpstreamOptions))
			if err != nil {
				return "", errors.Wrap(err, "failed to load application")
			}

			writeUpstreamImageOptions := base.WriteUpstreamImageOptions{
				BaseDir: writeBaseOptions.BaseDir,
				Log:     log,
				SourceRegistry: registry.RegistryOptions{
					Endpoint:      replicatedRegistryInfo.Registry,
					ProxyEndpoint: replicatedRegistryInfo.Proxy,
				},
				ReportWriter: pullOptions.ReportWriter,
				Installation: newInstallation,
				Application:  newApplication,
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

			copyResult, err := base.CopyUpstreamImages(writeUpstreamImageOptions)
			if err != nil {
				return "", errors.Wrap(err, "failed to write upstream images")
			}
			images = copyResult.Images

			newInstallation.Spec.KnownImages = copyResult.CheckedImages

			err = upstream.SaveInstallation(newInstallation, u.GetUpstreamDir(writeUpstreamOptions))
			if err != nil {
				return "", errors.Wrap(err, "failed to save installation")
			}
		}

		// If the request includes a rewrite image options host name, then also
		// push the images
		if pullOptions.RewriteImageOptions.Host != "" {
			processUpstreamImageOptions := upstream.ProcessUpstreamImagesOptions{
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
				processUpstreamImageOptions.ReplicatedRegistry.Username = fetchOptions.License.Spec.LicenseID
				processUpstreamImageOptions.ReplicatedRegistry.Password = fetchOptions.License.Spec.LicenseID
			}

			var rewrittenImages []kustomizetypes.Image
			if images == nil { // don't do ProcessUpstreamImages if we already copied them
				imagesData, err := ioutil.ReadFile(filepath.Join(pullOptions.AirgapRoot, "images.json"))
				if err != nil && !os.IsNotExist(err) {
					return "", errors.Wrap(err, "failed to load images file")
				}

				if err == nil {
					err := json.Unmarshal(imagesData, &images)
					if err != nil && !os.IsNotExist(err) {
						return "", errors.Wrap(err, "failed to unmarshal images data")
					}
					processUpstreamImageOptions.SkipImagePush = true
					processUpstreamImageOptions.KnownImages = images
				}

				rewrittenImages, err = upstream.ProcessUpstreamImages(u, processUpstreamImageOptions)
				if err != nil {
					return "", errors.Wrap(err, "failed to push upstream images")
				}
			}

			findObjectsOptions := base.FindObjectsWithImagesOptions{
				BaseDir: writeBaseOptions.BaseDir,
			}
			affectedObjects, err := base.FindObjectsWithImages(findObjectsOptions)
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

		newInstallation, err := upstream.LoadInstallation(u.GetUpstreamDir(writeUpstreamOptions))
		if err != nil {
			return "", errors.Wrap(err, "failed to load installation")
		}

		application, err := upstream.LoadApplication(u.GetUpstreamDir(writeUpstreamOptions))
		if err != nil {
			return "", errors.Wrap(err, "failed to load application")
		}

		allPrivate := false
		if application != nil {
			allPrivate = application.Spec.ProxyPublicImages
		}

		// Rewrite private images
		findPrivateImagesOptions := base.FindPrivateImagesOptions{
			BaseDir: writeBaseOptions.BaseDir,
			AppSlug: fetchOptions.License.Spec.AppSlug,
			ReplicatedRegistry: registry.RegistryOptions{
				Endpoint:      replicatedRegistryInfo.Registry,
				ProxyEndpoint: replicatedRegistryInfo.Proxy,
			},
			Installation:     newInstallation,
			AllImagesPrivate: allPrivate,
		}
		findResult, err := base.FindPrivateImages(findPrivateImagesOptions)
		if err != nil {
			return "", errors.Wrap(err, "failed to find private images")
		}

		newInstallation.Spec.KnownImages = findResult.CheckedImages
		err = upstream.SaveInstallation(newInstallation, u.GetUpstreamDir(writeUpstreamOptions))
		if err != nil {
			return "", errors.Wrap(err, "failed to save installation")
		}

		// Note that there maybe no rewritten images if only replicated private images are being used.
		// We still need to create the secret in that case.
		if len(findResult.Docs) > 0 {
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
		images = findResult.Images
		objects = findResult.Docs
	}

	log.ActionWithSpinner("Creating midstream")
	io.WriteString(pullOptions.ReportWriter, "Creating midstream\n")

	m, err := midstream.CreateMidstream(b, images, objects, pullSecret)
	if err != nil {
		return "", errors.Wrap(err, "failed to create midstream")
	}
	log.FinishSpinner()

	writeMidstreamOptions := midstream.WriteOptions{
		MidstreamDir: filepath.Join(b.GetOverlaysDir(writeBaseOptions), "midstream"),
		BaseDir:      u.GetBaseDir(writeUpstreamOptions),
		AppSlug:      pullOptions.AppSlug,
		IsGitOps:     pullOptions.IsGitOps,
	}
	if err := m.WriteMidstream(writeMidstreamOptions); err != nil {
		return "", errors.Wrap(err, "failed to write midstream")
	}

	for _, downstreamName := range pullOptions.Downstreams {
		log.ActionWithSpinner("Creating downstream %q", downstreamName)
		io.WriteString(pullOptions.ReportWriter, fmt.Sprintf("Creating downstream %q\n", downstreamName))
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

func ParseLicenseFromFile(filename string) (*kotsv1beta1.License, error) {
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

func ParseConfigValuesFromFile(filename string) (*kotsv1beta1.ConfigValues, error) {
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

func GetAppMetadataFromAirgap(airgapArchive string) ([]byte, error) {
	appArchive, err := archives.GetFileFromAirgap("app.tar.gz", airgapArchive)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract app archive")
	}

	tempDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(tempDir)

	err = archives.ExtractTGZArchiveFromReader(bytes.NewReader(appArchive), tempDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to extract app archive")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(tempDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read kots kinds")
	}

	s := k8sjson.NewYAMLSerializer(k8sjson.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(&kotsKinds.KotsApplication, &b); err != nil {
		errors.Wrap(err, "failed to encode metadata")
	}

	return b.Bytes(), nil
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
		// not sure when this would happen, but earlier logic allows this combination
		return nil
	}

	publicKey, err := GetAppPublicKey(license)
	if err != nil {
		return errors.Wrap(err, "failed to get public key from license")
	}

	if err := verify([]byte(license.Spec.AppSlug), []byte(airgap.Spec.Signature), publicKey); err != nil {
		if airgap.Spec.AppSlug != "" {
			return util.ActionableError{Message: fmt.Sprintf("Failed to verify bundle signature - license is for app %q, airgap package for app %q", license.Spec.AppSlug, airgap.Spec.AppSlug)}
		} else {
			return util.ActionableError{Message: fmt.Sprintf("Failed to verify bundle signature - airgap package does not match license app %q", license.Spec.AppSlug)}
		}
	}

	return nil
}

func LicenseIsExpired(license *kotsv1beta1.License) (bool, error) {
	val, found := license.Spec.Entitlements["expires_at"]
	if !found {
		return false, nil
	}
	if val.ValueType != "" && val.ValueType != "String" {
		return false, errors.Errorf("expires_at must be type String: %s", val.ValueType)
	}
	if val.Value.StrVal == "" {
		return false, nil
	}

	partsed, err := time.Parse(time.RFC3339, val.Value.StrVal)
	if err != nil {
		return false, errors.Wrap(err, "failed to parse expiration time")
	}
	return partsed.Before(time.Now()), nil
}

func findConfig(localPath string) (*kotsv1beta1.Config, *kotsv1beta1.ConfigValues, *kotsv1beta1.License, *kotsv1beta1.Installation, error) {
	if localPath == "" {
		return nil, nil, nil, nil, nil
	}

	var config *kotsv1beta1.Config
	var values *kotsv1beta1.ConfigValues
	var license *kotsv1beta1.License
	var installation *kotsv1beta1.Installation

	err := filepath.Walk(localPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			content, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			decode := scheme.Codecs.UniversalDeserializer().Decode
			obj, gvk, err := decode(content, nil, nil)
			if err != nil {
				return nil
			}

			if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Config" {
				config = obj.(*kotsv1beta1.Config)
			} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "ConfigValues" {
				values = obj.(*kotsv1beta1.ConfigValues)
			} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "License" {
				license = obj.(*kotsv1beta1.License)
			} else if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "Installation" {
				installation = obj.(*kotsv1beta1.Installation)
			}

			return nil
		})

	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "failed to walk local dir")
	}

	return config, values, license, installation, nil
}
