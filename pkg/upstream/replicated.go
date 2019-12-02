package upstream

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	imagedocker "github.com/containers/image/docker"
	dockerref "github.com/containers/image/docker/reference"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/docker/registry"
	"github.com/replicatedhq/kots/pkg/image"
	"github.com/replicatedhq/kots/pkg/k8sdoc"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	serializer "k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
	kustomizeimage "sigs.k8s.io/kustomize/v3/pkg/image"
)

const DefaultMetadata = `apiVersion: kots.io/v1beta1
kind: Application
metadata:
  name: "default-application"
spec:
  title: "the application"
  icon: https://cdn2.iconfinder.com/data/icons/mixd/512/16_kubernetes-512.png
  releaseNotes: |
    release notes`

type ReplicatedUpstream struct {
	Channel      *string
	AppSlug      string
	VersionLabel *string
	Sequence     *int
}

type App struct {
	Name string
}

type Release struct {
	UpdateCursor string
	VersionLabel string
	ReleaseNotes string
	Manifests    map[string][]byte
}

type ChannelRelease struct {
	ChannelSequence int    `json:"channelSequence"`
	ReleaseSequence int    `json:"releaseSequence"`
	VersionLabel    string `json:"versionLabel"`
	CreatedAt       string `json:"createdAt"`
}

func getUpdatesReplicated(u *url.URL, localPath string, currentCursor, versionLabel string, license *kotsv1beta1.License, channelSequence string) ([]Update, error) {
	if localPath != "" {
		parsedLocalRelease, err := readReplicatedAppFromLocalPath(localPath, currentCursor, versionLabel)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read replicated app from local path")
		}

		return []Update{{Cursor: parsedLocalRelease.UpdateCursor, VersionLabel: versionLabel}}, nil
	}

	// A license file is required to be set for this to succeed
	if license == nil {
		return nil, errors.New("No license was provided")
	}

	replicatedUpstream, err := parseReplicatedURL(u)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse replicated upstream")
	}

	remoteLicense, err := getSuccessfulHeadResponse(replicatedUpstream, license)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get successful head response")
	}

	pendingReleases, err := listPendingChannelReleases(replicatedUpstream, remoteLicense, channelSequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list replicated app releases")
	}

	updates := []Update{}
	for _, pendingRelease := range pendingReleases {
		updates = append(updates, Update{
			Cursor:       strconv.Itoa(pendingRelease.ChannelSequence),
			VersionLabel: pendingRelease.VersionLabel,
		})
	}
	return updates, nil
}

func downloadReplicated(u *url.URL, localPath string, rootDir string, useAppDir bool, license *kotsv1beta1.License, existingConfigValues *kotsv1beta1.ConfigValues, updateCursor, versionLabel string) (*Upstream, error) {
	var release *Release

	if localPath != "" {
		parsedLocalRelease, err := readReplicatedAppFromLocalPath(localPath, updateCursor, versionLabel)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read replicated app from local path")
		}

		release = parsedLocalRelease
		if updateCursor != "" && release.UpdateCursor != updateCursor {
			return nil, errors.Wrap(err, "release in local path does not match update cursor")
		}
	} else {
		// A license file is required to be set for this to succeed
		if license == nil {
			return nil, errors.New("No license was provided")
		}

		replicatedUpstream, err := parseReplicatedURL(u)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse replicated upstream")
		}

		remoteLicense, err := getSuccessfulHeadResponse(replicatedUpstream, license)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get successful head response")
		}

		downloadedRelease, err := downloadReplicatedApp(replicatedUpstream, remoteLicense, updateCursor)
		if err != nil {
			return nil, errors.Wrap(err, "failed to download replicated app")
		}

		release = downloadedRelease
	}

	// Find the config in the upstream and write out default values

	application := findAppInRelease(release) // this function never returns nil

	// NOTE: this currently comes from the application spec and not the channel release meta
	if release.ReleaseNotes == "" {
		release.ReleaseNotes = application.Spec.ReleaseNotes
	}

	if existingConfigValues == nil {
		var prevConfigFile string
		if useAppDir {
			prevConfigFile = filepath.Join(rootDir, application.Name, "upstream", "userdata", "config.yaml")
		} else {
			prevConfigFile = filepath.Join(rootDir, "upstream", "userdata", "config.yaml")
		}
		var err error
		existingConfigValues, err = findConfigValuesInFile(prevConfigFile)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load existing config values")
		}
	}

	config := findConfigInRelease(release)
	if config != nil || existingConfigValues != nil {
		// If config existed and was removed from the app,
		// values will be carried over to the new version anyway.
		configValues, err := createConfigValues(application.Name, config, existingConfigValues)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create empty config values")
		}

		release.Manifests["userdata/config.yaml"] = mustMarshalConfigValues(configValues)
	}

	// Add the license to the upstream, if one was propvided
	if license != nil {
		release.Manifests["userdata/license.yaml"] = MustMarshalLicense(license)
	}

	files, err := releaseToFiles(release)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get files from release")
	}

	upstream := &Upstream{
		URI:          u.RequestURI(),
		Name:         application.Name,
		Files:        files,
		Type:         "replicated",
		UpdateCursor: release.UpdateCursor,
		VersionLabel: release.VersionLabel,
		ReleaseNotes: release.ReleaseNotes,
	}

	return upstream, nil
}

func (r *ReplicatedUpstream) getRequest(method string, license *kotsv1beta1.License, channelSequence string) (*http.Request, error) {
	u, err := url.Parse(license.Spec.Endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse endpoint from license")
	}

	hostname := u.Hostname()
	if u.Port() != "" {
		hostname = fmt.Sprintf("%s:%s", u.Hostname(), u.Port())
	}

	urlPath := path.Join(hostname, "release", license.Spec.AppSlug)
	if r.Channel != nil {
		urlPath = path.Join(urlPath, *r.Channel)
	}
	url := fmt.Sprintf("%s://%s?channelSequence=%s", u.Scheme, urlPath, channelSequence)

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", license.Spec.LicenseID, license.Spec.LicenseID)))))

	return req, nil
}

func parseReplicatedURL(u *url.URL) (*ReplicatedUpstream, error) {
	replicatedUpstream := ReplicatedUpstream{}

	if u.User != nil {
		if u.User.Username() != "" {
			replicatedUpstream.AppSlug = u.User.Username()
			versionLabel := u.Hostname()
			replicatedUpstream.VersionLabel = &versionLabel
		}
	}

	if replicatedUpstream.AppSlug == "" {
		replicatedUpstream.AppSlug = u.Hostname()
		if u.Path != "" {
			channel := strings.TrimPrefix(u.Path, "/")
			replicatedUpstream.Channel = &channel
		}
	}

	return &replicatedUpstream, nil
}

func getSuccessfulHeadResponse(replicatedUpstream *ReplicatedUpstream, license *kotsv1beta1.License) (*kotsv1beta1.License, error) {
	headReq, err := replicatedUpstream.getRequest("HEAD", license, "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	headResp, err := http.DefaultClient.Do(headReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute head request")
	}
	defer headResp.Body.Close()

	if headResp.StatusCode == 401 {
		return nil, errors.New("license was not accepted")
	}

	if headResp.StatusCode >= 400 {
		return nil, errors.Errorf("unexpected result from head request: %d", headResp.StatusCode)
	}

	return license, nil
}

func readReplicatedAppFromLocalPath(localPath, localCursor, versionLabel string) (*Release, error) {
	release := Release{
		Manifests:    make(map[string][]byte),
		UpdateCursor: localCursor,
		VersionLabel: versionLabel,
	}

	err := filepath.Walk(localPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			contents, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			// remove localpath prefix
			appPath := strings.TrimPrefix(path, localPath)
			appPath = strings.TrimLeft(appPath, string(os.PathSeparator))

			release.Manifests[appPath] = contents

			return nil
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to walk local path")
	}

	return &release, nil
}

func downloadReplicatedApp(replicatedUpstream *ReplicatedUpstream, license *kotsv1beta1.License, channelSequence string) (*Release, error) {
	getReq, err := replicatedUpstream.getRequest("GET", license, channelSequence)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get request")
	}
	defer getResp.Body.Close()

	if getResp.StatusCode >= 400 {
		return nil, errors.Errorf("unexpected result from get request: %d", getResp.StatusCode)
	}

	updateCursor := getResp.Header.Get("X-Replicated-ChannelSequence")
	versionLabel := getResp.Header.Get("X-Replicated-VersionLabel")

	gzf, err := gzip.NewReader(getResp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new gzip reader")
	}

	release := Release{
		Manifests:    make(map[string][]byte),
		UpdateCursor: updateCursor,
		VersionLabel: versionLabel,
		// NOTE: release notes come from Application spec
	}
	tarReader := tar.NewReader(gzf)
	i := 0
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "failed to get next file from reader")
		}

		name := header.Name

		switch header.Typeflag {
		case tar.TypeDir:
			continue
		case tar.TypeReg:
			content, err := ioutil.ReadAll(tarReader)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read file from tar")
			}

			release.Manifests[name] = content
		}

		i++
	}

	return &release, nil
}

func listPendingChannelReleases(replicatedUpstream *ReplicatedUpstream, license *kotsv1beta1.License, channelSequence string) ([]ChannelRelease, error) {
	u, err := url.Parse(license.Spec.Endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse endpoint from license")
	}

	hostname := u.Hostname()
	if u.Port() != "" {
		hostname = fmt.Sprintf("%s:%s", u.Hostname(), u.Port())
	}

	url := fmt.Sprintf("%s://%s/release/%s/pending?channelSequence=%s", u.Scheme, hostname, license.Spec.AppSlug, channelSequence)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", license.Spec.LicenseID, license.Spec.LicenseID)))))

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get request")
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, errors.Errorf("unexpected result from get request: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	var channelReleases struct {
		ChannelReleases []ChannelRelease `json:"channelReleases"`
	}
	if err := json.Unmarshal(body, &channelReleases); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response")
	}

	return channelReleases.ChannelReleases, nil
}

func MustMarshalLicense(license *kotsv1beta1.License) []byte {
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(license, &b); err != nil {
		panic(err)
	}

	return b.Bytes()
}

func mustMarshalConfigValues(configValues *kotsv1beta1.ConfigValues) []byte {
	s := serializer.NewYAMLSerializer(serializer.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(configValues, &b); err != nil {
		panic(err)
	}

	return b.Bytes()
}

func createConfigValues(applicationName string, config *kotsv1beta1.Config, existingConfigValues *kotsv1beta1.ConfigValues) (*kotsv1beta1.ConfigValues, error) {
	templateContextValues := make(map[string]template.ItemValue)

	var newValues kotsv1beta1.ConfigValuesSpec
	if existingConfigValues != nil {
		for k, v := range existingConfigValues.Spec.Values {
			templateContextValues[k] = template.ItemValue{
				Value:   v.Value,
				Default: v.Default,
			}
		}
		newValues = kotsv1beta1.ConfigValuesSpec{
			Values: existingConfigValues.Spec.Values,
		}
	} else {
		newValues = kotsv1beta1.ConfigValuesSpec{
			Values: map[string]kotsv1beta1.ConfigValue{},
		}
	}

	if config == nil {
		return &kotsv1beta1.ConfigValues{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "kots.io/v1beta1",
				Kind:       "ConfigValues",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: applicationName,
			},
			Spec: newValues,
		}, nil
	}

	builder := template.Builder{}
	builder.AddCtx(template.StaticCtx{})

	configCtx, err := builder.NewConfigContext(config.Spec.Groups, templateContextValues)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create config context")
	}
	builder.AddCtx(configCtx)

	for _, group := range config.Spec.Groups {
		for _, item := range group.Items {
			var foundValue string
			prevValue, ok := newValues.Values[item.Name]
			if ok && prevValue.Value != "" {
				foundValue = prevValue.Value
			}

			renderedValue, err := builder.RenderTemplate(item.Name, item.Value)
			if err != nil {
				return nil, errors.Wrap(err, "failed to render config item value")
			}

			renderedDefault, err := builder.RenderTemplate(item.Name, item.Default)
			if err != nil {
				return nil, errors.Wrap(err, "failed to render config item default")
			}

			if renderedValue == "" && renderedDefault == "" && foundValue == "" {
				continue
			}

			if foundValue != "" {
				newValues.Values[item.Name] = kotsv1beta1.ConfigValue{
					Value:   foundValue,
					Default: renderedDefault,
				}
			} else {
				newValues.Values[item.Name] = kotsv1beta1.ConfigValue{
					Value:   renderedValue,
					Default: renderedDefault,
				}
			}
		}
	}

	configValues := kotsv1beta1.ConfigValues{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "ConfigValues",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: applicationName,
		},
		Spec: newValues,
	}

	return &configValues, nil
}

func findConfigValuesInFile(filename string) (*kotsv1beta1.ConfigValues, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to open file")
	}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, gvk, err := decode(content, nil, nil)
	if err != nil {
		return nil, nil
	}

	if gvk.Group == "kots.io" && gvk.Version == "v1beta1" && gvk.Kind == "ConfigValues" {
		return obj.(*kotsv1beta1.ConfigValues), nil
	}

	return nil, nil
}

func findConfigInRelease(release *Release) *kotsv1beta1.Config {
	for _, content := range release.Manifests {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode(content, nil, nil)
		if err != nil {
			continue
		}

		if gvk.Group == "kots.io" {
			if gvk.Version == "v1beta1" {
				if gvk.Kind == "Config" {
					return obj.(*kotsv1beta1.Config)
				}
			}
		}
	}

	return nil
}

func findAppInRelease(release *Release) *kotsv1beta1.Application {
	for _, content := range release.Manifests {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, err := decode(content, nil, nil)
		if err != nil {
			continue
		}

		if gvk.Group == "kots.io" {
			if gvk.Version == "v1beta1" {
				if gvk.Kind == "Application" {
					return obj.(*kotsv1beta1.Application)
				}
			}
		}
	}

	// Using Ship apps for now, so let's create an app manifest on the fly
	app := &kotsv1beta1.Application{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "kots.io/v1beta1",
			Kind:       "Application",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "replicated-kots-app",
		},
		Spec: kotsv1beta1.ApplicationSpec{
			Title: "Replicated Kots App",
			Icon:  "",
		},
	}
	return app
}

func releaseToFiles(release *Release) ([]UpstreamFile, error) {
	upstreamFiles := []UpstreamFile{}

	for filename, content := range release.Manifests {
		upstreamFile := UpstreamFile{
			Path:    filename,
			Content: content,
		}

		upstreamFiles = append(upstreamFiles, upstreamFile)
	}

	// Stash the user data for this search (we will readd at the end)
	userdataFiles := []UpstreamFile{}
	withoutUserdataFiles := []UpstreamFile{}
	for _, file := range upstreamFiles {
		d, _ := path.Split(file.Path)
		dirs := strings.Split(d, string(os.PathSeparator))

		if dirs[0] == "userdata" {
			userdataFiles = append(userdataFiles, file)
		} else {
			withoutUserdataFiles = append(withoutUserdataFiles, file)
		}
	}

	// remove any common prefix from all files
	if len(withoutUserdataFiles) > 0 {
		firstFileDir, _ := path.Split(withoutUserdataFiles[0].Path)
		commonPrefix := strings.Split(firstFileDir, string(os.PathSeparator))

		for _, file := range withoutUserdataFiles {
			d, _ := path.Split(file.Path)
			dirs := strings.Split(d, string(os.PathSeparator))

			commonPrefix = util.CommonSlicePrefix(commonPrefix, dirs)

		}

		cleanedUpstreamFiles := []UpstreamFile{}
		for _, file := range withoutUserdataFiles {
			d, f := path.Split(file.Path)
			d2 := strings.Split(d, string(os.PathSeparator))

			cleanedUpstreamFile := file
			d2 = d2[len(commonPrefix):]
			cleanedUpstreamFile.Path = path.Join(path.Join(d2...), f)

			cleanedUpstreamFiles = append(cleanedUpstreamFiles, cleanedUpstreamFile)
		}

		upstreamFiles = cleanedUpstreamFiles
	}

	upstreamFiles = append(upstreamFiles, userdataFiles...)

	return upstreamFiles, nil
}

// GetApplicationMetadata will return any available application yaml from
// the upstream. If there is no application.yaml, it will return
// a placeholder one
func GetApplicationMetadata(upstream *url.URL) ([]byte, error) {
	metadata, err := getApplicationMetadataFromHost("replicated.app", upstream)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get metadata from replicated.app")
	}

	if metadata == nil {
		otherMetadata, err := getApplicationMetadataFromHost("staging.replicated.app", upstream)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get metadata from staging.replicated.app")
		}

		metadata = otherMetadata
	}

	if metadata == nil {
		metadata = []byte(DefaultMetadata)
	}

	return metadata, nil
}

func getApplicationMetadataFromHost(host string, upstream *url.URL) ([]byte, error) {
	r, err := parseReplicatedURL(upstream)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse replicated upstream")
	}

	url := fmt.Sprintf("https://%s/metadata/%s", host, r.AppSlug)

	if r.Channel != nil {
		url = fmt.Sprintf("%s/%s", url, *r.Channel)
	}

	getReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get request")
	}
	defer getResp.Body.Close()

	if getResp.StatusCode == 404 {
		// no metadata is not an error
		return nil, nil
	}

	if getResp.StatusCode >= 400 {
		return nil, errors.Errorf("unexpected result from get request: %d", getResp.StatusCode)
	}

	respBody, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	return respBody, nil
}

type FindPrivateImagesOptions struct {
	RootDir            string
	CreateAppDir       bool
	AppSlug            string
	ReplicatedRegistry registry.RegistryOptions
	Log                *logger.Logger
}

func (u *Upstream) FindPrivateImages(options FindPrivateImagesOptions) ([]kustomizeimage.Image, []*k8sdoc.Doc, error) {
	rootDir := options.RootDir
	if options.CreateAppDir {
		rootDir = path.Join(rootDir, u.Name)
	}
	upstreamDir := path.Join(rootDir, "upstream")

	upstreamImages, objects, err := image.GetPrivateImages(upstreamDir)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to list upstream images")
	}

	result := make([]kustomizeimage.Image, 0)
	for _, upstreamImage := range upstreamImages {
		// ParseReference requires the // prefix
		ref, err := imagedocker.ParseReference(fmt.Sprintf("//%s", upstreamImage))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to parse image ref:%s", upstreamImage)
		}

		registryHost := dockerref.Domain(ref.DockerReference())
		if registryHost == options.ReplicatedRegistry.Endpoint {
			// replicated images are also private, but we don't rewrite those
			continue
		}

		image := kustomizeimage.Image{
			Name:    upstreamImage,
			NewName: registry.MakeProxiedImageURL(options.ReplicatedRegistry.ProxyEndpoint, options.AppSlug, upstreamImage),
		}
		result = append(result, image)
	}

	return result, objects, nil
}

type FindObjectsWithImagesOptions struct {
	RootDir      string
	CreateAppDir bool
	Log          *logger.Logger
}

func (u *Upstream) FindObjectsWithImages(options FindObjectsWithImagesOptions) ([]*k8sdoc.Doc, error) {
	rootDir := options.RootDir
	if options.CreateAppDir {
		rootDir = path.Join(rootDir, u.Name)
	}
	upstreamDir := path.Join(rootDir, "upstream")

	objects, err := image.GetObjectsWithImages(upstreamDir)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list upstream images")
	}

	return objects, nil
}
