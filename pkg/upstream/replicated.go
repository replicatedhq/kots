package upstream

import (
	"archive/tar"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
	kotsscheme "github.com/replicatedhq/kotsadm/kotskinds/client/kotsclientset/scheme"
	"k8s.io/client-go/kubernetes/scheme"
)

type ReplicatedUpstream struct {
	Host         string
	Channel      *string
	AppSlug      string
	VersionLabel *string
	Sequence     *int
}

type App struct {
	Name string
}

type Release struct {
	Manifests map[string][]byte
}

func downloadReplicated(u *url.URL) (*Upstream, error) {
	replicatedUpstream, err := parseReplicatedURL(u)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse replicated upstream")
	}

	licenseID, err := getSuccessfulHeadResponse(replicatedUpstream)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get successful head response")
	}

	release, err := downloadReplicatedApp(replicatedUpstream, licenseID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to download replicated app")
	}

	app, err := findAppInRelease(release)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app from release")
	}

	files, err := releaseToFiles(release)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get files from release")
	}
	upstream := &Upstream{
		URI:   u.RequestURI(),
		Name:  app.Name,
		Files: files,
		Type:  "replicated",
	}

	return upstream, nil
}

func (r *ReplicatedUpstream) getRequest(method string, licenseID string) (*http.Request, error) {
	proto := "https"

	// quick and dirty hack to ensure we always have https, except don't require it when running completely local (dev)
	if strings.HasPrefix(r.Host, "localhost") {
		proto = "http"
	}

	url := fmt.Sprintf("%s://%s/release/%s", proto, r.Host, r.AppSlug)
	if r.Channel != nil {
		url = fmt.Sprintf("%s/%s", url, *r.Channel)
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to call newrequest")
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", licenseID, licenseID)))))

	return req, nil
}

func parseReplicatedURL(u *url.URL) (*ReplicatedUpstream, error) {
	replicatedUpstream := ReplicatedUpstream{
		Host: "replicated.app",
	}

	if u.User != nil {
		if u.User.Username() != "" {
			replicatedUpstream.AppSlug = u.User.Username()
			host := u.Hostname()
			replicatedUpstream.VersionLabel = &host
		}
	}

	if replicatedUpstream.AppSlug == "" {
		replicatedUpstream.AppSlug = u.Hostname()
		if u.Path != "" {
			channel := strings.TrimPrefix(u.Path, "/")
			replicatedUpstream.Channel = &channel
		}
	}

	if u.Query().Get("host") != "" {
		replicatedUpstream.Host = u.Query().Get("host")
	}

	return &replicatedUpstream, nil
}

func getSuccessfulHeadResponse(replicatedUpstream *ReplicatedUpstream) (string, error) {
	templates := &promptui.PromptTemplates{
		Prompt:  "{{ . | bold }} ",
		Valid:   "{{ . | green }} ",
		Invalid: "{{ . | red }} ",
		Success: "{{ . | bold }} ",
	}

	prompt := promptui.Prompt{
		Label:     "License ID",
		Templates: templates,
		Validate: func(input string) error {
			if len(input) < 24 {
				return errors.New("invalid license")
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

		headReq, err := replicatedUpstream.getRequest("HEAD", result)
		if err != nil {
			return "", errors.Wrap(err, "failed to create http request")
		}
		headResp, err := http.DefaultClient.Do(headReq)
		if err != nil {
			return "", errors.Wrap(err, "failed to execute head request")
		}

		if headResp.StatusCode == 401 {
			continue
		}
		if headResp.StatusCode >= 400 {
			return "", errors.Errorf("expected result from head request: %d", headResp.StatusCode)
		}

		return result, nil
	}

}

func downloadReplicatedApp(replicatedUpstream *ReplicatedUpstream, licenseID string) (*Release, error) {
	getReq, err := replicatedUpstream.getRequest("GET", licenseID)
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

	gzf, err := gzip.NewReader(getResp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new gzip reader")
	}

	release := Release{
		Manifests: make(map[string][]byte),
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

func findAppInRelease(release *Release) (*App, error) {
	kotsscheme.AddToScheme(scheme.Scheme)
	for _, content := range release.Manifests {
		decode := scheme.Codecs.UniversalDeserializer().Decode
		_, gvk, err := decode(content, nil, nil)
		if err != nil {
			continue
		}

		if gvk.Group == "kots.io" {
			if gvk.Version == "v1beta1" {
				if gvk.Kind == "App" {
					return nil, nil
				}
			}
		}
	}

	// Using Ship apps for now, so let's create an app manifest on the fly
	app := &App{
		Name: "Ship App",
	}
	return app, nil
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
