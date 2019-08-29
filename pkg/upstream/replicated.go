package upstream

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	kotsscheme "github.com/replicatedhq/kots/kotskinds/client/kotsclientset/scheme"
	"github.com/replicatedhq/kots/pkg/kotsadm"
	"github.com/replicatedhq/kots/pkg/template"
	"github.com/replicatedhq/kots/pkg/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/client-go/kubernetes/scheme"
)

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
	Manifests    map[string][]byte
}

func downloadReplicated(u *url.URL, localPath string, license *kotsv1beta1.License, includeAdminConsole bool, sharedPassword string) (*Upstream, error) {
	var release *Release

	if localPath != "" {
		parsedLocalRelease, err := readReplicatedAppFromLocalPath(localPath)
		if err != nil {
			return nil, errors.Wrap(err, "failed to read replicated app from local path")
		}

		release = parsedLocalRelease
	} else {
		// A license file is required to be set for this to succeed
		if license == nil {
			return nil, errors.New("No license was provided")
		}

		replicatedUpstream, err := parseReplicatedURL(u)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse replicated upstream")
		}

		license, err := getSuccessfulHeadResponse(replicatedUpstream, license)
		if err != nil {
			return nil, errors.Wrap(err, "failed to get successful head response")
		}

		downloadedRelease, err := downloadReplicatedApp(replicatedUpstream, license)
		if err != nil {
			return nil, errors.Wrap(err, "failed to download replicated app")
		}

		release = downloadedRelease
	}

	// Find the config in the upstream and write out default values
	application := findAppInRelease(release)
	config := findConfigInRelease(release)
	if config != nil {
		configValues, err := createEmptyConfigValues(application.Name, config)
		if err != nil {
			return nil, errors.Wrap(err, "failed to create empty config values")
		}

		release.Manifests["userdata/config.yaml"] = mustMarshalConfigValues(configValues)
	}

	// Add the license to the upstream, if one was propvided
	if license != nil {
		release.Manifests["userdata/license.yaml"] = mustMarshalLicense(license)
	}

	files, err := releaseToFiles(release)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get files from release")
	}

	if includeAdminConsole {
		adminConsoleFiles, err := generateAdminConsoleFiles(sharedPassword)
		if err != nil {
			return nil, errors.Wrap(err, "failed to generate admin console files")
		}

		files = append(files, adminConsoleFiles...)
	}

	upstream := &Upstream{
		URI:          u.RequestURI(),
		Name:         application.Name,
		Files:        files,
		Type:         "replicated",
		UpdateCursor: release.UpdateCursor,
	}

	return upstream, nil
}

func (r *ReplicatedUpstream) getRequest(method string, license *kotsv1beta1.License) (*http.Request, error) {
	u, err := url.Parse(license.Spec.Endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse endpoint from license")
	}

	hostname := u.Hostname()
	if u.Port() != "" {
		hostname = fmt.Sprintf("%s:%s", u.Hostname(), u.Port())
	}

	url := fmt.Sprintf("%s://%s/release/%s", u.Scheme, hostname, license.Spec.AppSlug)

	if r.Channel != nil {
		url = fmt.Sprintf("%s/%s", url, *r.Channel)
	}

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
	headReq, err := replicatedUpstream.getRequest("HEAD", license)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	headResp, err := http.DefaultClient.Do(headReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute head request")
	}

	if headResp.StatusCode == 401 {
		return nil, errors.Wrap(err, "license was not accepted")
	}

	if headResp.StatusCode >= 400 {
		return nil, errors.Errorf("expected result from head request: %d", headResp.StatusCode)
	}

	return license, nil
}

func readReplicatedAppFromLocalPath(localPath string) (*Release, error) {
	release := Release{
		Manifests:    make(map[string][]byte),
		UpdateCursor: "-1", // TODO
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

func downloadReplicatedApp(replicatedUpstream *ReplicatedUpstream, license *kotsv1beta1.License) (*Release, error) {
	getReq, err := replicatedUpstream.getRequest("GET", license)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute get request")
	}

	if getResp.StatusCode >= 400 {
		return nil, errors.Errorf("expected result from get request: %d", getResp.StatusCode)
	}

	defer getResp.Body.Close()

	updateCursor := getResp.Header.Get("X-Replicated-Sequence")

	gzf, err := gzip.NewReader(getResp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new gzip reader")
	}

	release := Release{
		Manifests:    make(map[string][]byte),
		UpdateCursor: updateCursor,
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

func mustMarshalLicense(license *kotsv1beta1.License) []byte {
	kotsscheme.AddToScheme(scheme.Scheme)

	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(license, &b); err != nil {
		panic(err)
	}

	return b.Bytes()
}

func mustMarshalConfigValues(configValues *kotsv1beta1.ConfigValues) []byte {
	kotsscheme.AddToScheme(scheme.Scheme)

	s := json.NewYAMLSerializer(json.DefaultMetaFactory, scheme.Scheme, scheme.Scheme)

	var b bytes.Buffer
	if err := s.Encode(configValues, &b); err != nil {
		panic(err)
	}

	return b.Bytes()
}

func createEmptyConfigValues(applicationName string, config *kotsv1beta1.Config) (*kotsv1beta1.ConfigValues, error) {
	emptyValues := kotsv1beta1.ConfigValuesSpec{
		Values: map[string]string{},
	}

	builder := template.Builder{}
	builder.AddCtx(template.StaticCtx{})

	for _, group := range config.Spec.Groups {
		for _, item := range group.Items {
			if item.Value != "" {
				rendered, err := builder.RenderTemplate(item.Name, item.Value)
				if err != nil {
					return nil, errors.Wrap(err, "failed to render config item value")
				}

				emptyValues.Values[item.Name] = rendered
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
		Spec: emptyValues,
	}

	return &configValues, nil
}

func findConfigInRelease(release *Release) *kotsv1beta1.Config {
	kotsscheme.AddToScheme(scheme.Scheme)
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
	kotsscheme.AddToScheme(scheme.Scheme)
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

	// remove any common prefix from all files
	if len(upstreamFiles) > 0 {
		firstFileDir, _ := path.Split(upstreamFiles[0].Path)
		commonPrefix := strings.Split(firstFileDir, string(os.PathSeparator))

		for _, file := range upstreamFiles {
			d, _ := path.Split(file.Path)
			dirs := strings.Split(d, string(os.PathSeparator))

			commonPrefix = util.CommonSlicePrefix(commonPrefix, dirs)

		}

		cleanedUpstreamFiles := []UpstreamFile{}
		for _, file := range upstreamFiles {
			d, f := path.Split(file.Path)
			d2 := strings.Split(d, string(os.PathSeparator))

			cleanedUpstreamFile := file
			d2 = d2[len(commonPrefix):]
			cleanedUpstreamFile.Path = path.Join(path.Join(d2...), f)

			cleanedUpstreamFiles = append(cleanedUpstreamFiles, cleanedUpstreamFile)
		}

		upstreamFiles = cleanedUpstreamFiles
	}
	return upstreamFiles, nil
}

func generateAdminConsoleFiles(sharedPassword string) ([]UpstreamFile, error) {
	upstreamFiles := []UpstreamFile{}

	deployOptions := kotsadm.DeployOptions{
		Namespace: "default",
	}

	if sharedPassword == "" {
		p, err := promptForSharedPassword()
		if err != nil {
			return nil, errors.Wrap(err, "failed to prompt for shared password")
		}

		sharedPassword = p
	}

	deployOptions.SharedPassword = sharedPassword

	adminConsoleDocs, err := kotsadm.YAML(deployOptions)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get minio yaml")
	}
	for n, v := range adminConsoleDocs {
		upstreamFile := UpstreamFile{
			Path:    path.Join("admin-console", n),
			Content: v,
		}
		upstreamFiles = append(upstreamFiles, upstreamFile)
	}

	return upstreamFiles, nil
}

func promptForSharedPassword() (string, error) {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	prompt := promptui.Prompt{
		Label:     "Enter a new password to be used for the Admin Console:",
		Templates: templates,
		Mask:      rune('•'),
		Validate: func(input string) error {
			if len(input) < 6 {
				return errors.New("please enter a longer password")
			}

			return nil
		},
	}

	for {
		result, err := prompt.Run()
		if err != nil {
			if err == promptui.ErrInterrupt {
				os.Exit(-1)
			}
			continue
		}

		return result, nil
	}

}
