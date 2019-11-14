package upstream

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/util"
	"k8s.io/helm/cmd/helm/search"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/repo"
)

func getUpdatesHelm(u *url.URL, repoURI string) ([]Update, error) {
	repoName, chartName, _, err := parseHelmURL(u)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse helm uri")
	}

	helmHome, err := ioutil.TempDir("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temporary helm home")
	}
	defer os.RemoveAll(helmHome)

	i, err := helmLoadRepositoriesIndex(helmHome, repoName, repoURI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load helm repositories")
	}

	var updates []Update
	for _, result := range i.All() {
		if result.Chart.GetName() != chartName {
			continue
		}

		updates = append(updates, Update{Cursor: result.Chart.GetVersion()})
	}
	return updates, nil
}

func downloadHelm(u *url.URL, repoURI string) (*Upstream, error) {
	repoName, chartName, chartVersion, err := parseHelmURL(u)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse helm uri")
	}

	helmHome, err := ioutil.TempDir("", "kots")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temporary helm home")
	}
	defer os.RemoveAll(helmHome)

	i, err := helmLoadRepositoriesIndex(helmHome, repoName, repoURI)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load helm repositories")
	}

	if chartVersion == "" {
		highestChartVersion := semver.MustParse("0.0.0")
		for _, result := range i.All() {
			if result.Chart.GetName() != chartName {
				continue
			}

			v, err := semver.NewVersion(result.Chart.GetVersion())
			if err != nil {
				return nil, errors.Wrap(err, "unable to parse chart version")
			}

			if v.GreaterThan(highestChartVersion) {
				highestChartVersion = v
			}
		}

		chartVersion = highestChartVersion.String()
	}

	for _, result := range i.All() {
		if result.Chart.GetName() != chartName {
			continue
		}

		if result.Chart.GetVersion() != chartVersion {
			continue
		}

		dl := downloader.ChartDownloader{
			HelmHome: helmpath.Home(helmHome),
			Out:      os.Stdout,
			Getters:  getter.All(environment.EnvSettings{}),
		}

		archiveDir, err := ioutil.TempDir("", "archive")
		if err != nil {
			return nil, errors.Wrap(err, "failed to create archive directory for chart")
		}
		defer os.RemoveAll(archiveDir)

		chartRef, err := repo.FindChartInRepoURL(repoURI, result.Chart.GetName(), chartVersion, "", "", "", getter.All(environment.EnvSettings{}))
		if err != nil {
			return nil, errors.Wrap(err, "failed to find chart in repo url")
		}

		_, _, err = dl.DownloadTo(chartRef, result.Chart.GetVersion(), archiveDir)
		if err != nil {
			return nil, errors.Wrap(err, "failed to download chart")
		}

		files, err := readTarGz(path.Join(archiveDir, fmt.Sprintf("%s-%s.tgz", chartName, chartVersion)))
		if err != nil {
			return nil, errors.Wrap(err, "failed to read chart archive")
		}

		upstream := &Upstream{
			URI:          u.RequestURI(),
			Name:         chartName,
			Type:         "helm",
			Files:        files,
			UpdateCursor: chartVersion,
			VersionLabel: chartVersion,
		}

		return upstream, nil
	}

	return nil, errors.New("chart version not found")
}

func helmLoadRepositoriesIndex(helmHome, repoName, repoURI string) (*search.Index, error) {
	if repoURI == "" {
		repoURI = getKnownHelmRepoURI(repoName)
	}

	if repoURI == "" {
		return nil, errors.New("unknown helm repo uri, try passing the repo uri")
	}

	if err := os.MkdirAll(filepath.Join(helmHome, "repository"), 0755); err != nil {
		return nil, errors.Wrap(err, "failed to make directory for helm home")
	}
	reposFile := filepath.Join(helmHome, "repository", "repositories.yaml")

	repoIndexFile, err := ioutil.TempFile("", "index")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temporary index file")
	}
	defer os.Remove(repoIndexFile.Name())

	cacheIndexFile, err := ioutil.TempFile("", "cache")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create cache index file")
	}
	defer os.Remove(cacheIndexFile.Name())

	repoYAML := `apiVersion: v1
generated: "2019-05-29T14:31:58.906598702Z"
repositories: []`
	if err := ioutil.WriteFile(reposFile, []byte(repoYAML), 0644); err != nil {
		return nil, err
	}

	c := repo.Entry{
		Name:  repoName,
		Cache: repoIndexFile.Name(),
		URL:   repoURI,
	}
	r, err := repo.NewChartRepository(&c, getter.All(environment.EnvSettings{}))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chart repository")
	}
	if err := r.DownloadIndexFile(cacheIndexFile.Name()); err != nil {
		return nil, errors.Wrap(err, "failed to download index file")
	}

	rf, err := repo.LoadRepositoriesFile(reposFile)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load repositories file")
	}
	rf.Update(&c)

	i := search.NewIndex()
	for _, re := range rf.Repositories {
		n := re.Name
		ind, err := repo.LoadIndexFile(repoIndexFile.Name())
		if err != nil {
			return nil, errors.Wrap(err, "failed to load index file")
		}

		i.AddRepo(n, ind, true)
	}

	return i, nil
}

func parseHelmURL(u *url.URL) (string, string, string, error) {
	repo := u.Host
	chartName := strings.TrimLeft(u.Path, "/")
	chartVersion := ""

	chartAndVersion := strings.Split(chartName, "@")
	if len(chartAndVersion) > 1 {
		chartName = chartAndVersion[0]
		chartVersion = chartAndVersion[1]
	}

	return repo, chartName, chartVersion, nil
}

func getKnownHelmRepoURI(repoName string) string {
	val, ok := KnownRepos[repoName]
	if !ok {
		return ""
	}

	return val
}

func readTarGz(source string) ([]UpstreamFile, error) {
	f, err := os.Open(source)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open archive")
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create gzip reader")
	}

	tarReader := tar.NewReader(gzf)

	upstreamFiles := []UpstreamFile{}
	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "failed to advance in tar archive")
		}

		name := header.Name

		switch header.Typeflag {
		case tar.TypeReg:
			buf := new(bytes.Buffer)
			_, err = buf.ReadFrom(tarReader)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read file from tar archive")
			}
			upstreamFile := UpstreamFile{
				Path:    name,
				Content: buf.Bytes(),
			}

			upstreamFiles = append(upstreamFiles, upstreamFile)
		default:
			continue
		}
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
