package upstream

import (
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/upstream/types"
	"helm.sh/helm/v3/cmd/helm/search"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/getter"
	"helm.sh/helm/v3/pkg/repo"
)

func getUpdatesHelm(u *url.URL, repoURI string) (*types.UpdateCheckResult, error) {
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

	var updates []types.Update
	for _, result := range i.All() {
		if result.Chart.Name != chartName {
			continue
		}

		updates = append(updates, types.Update{Cursor: result.Chart.Version})
	}
	return &types.UpdateCheckResult{
		Updates:         updates,
		UpdateCheckTime: time.Now(),
	}, nil
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

	repoYAML := `apiVersion: v1
generated: "2019-05-29T14:31:58.906598702Z"
repositories: []`
	if err := ioutil.WriteFile(getReposFile(helmHome), []byte(repoYAML), 0644); err != nil {
		return nil, err
	}

	c := repo.Entry{
		Name: repoName,
		URL:  repoURI,
	}
	r, err := repo.NewChartRepository(&c, getter.All(&cli.EnvSettings{}))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create chart repository")
	}
	r.CachePath = getCachePath(helmHome)

	indexFilePath, err := r.DownloadIndexFile()
	if err != nil {
		return nil, errors.Wrap(err, "failed to download index file")
	}

	ind, err := repo.LoadIndexFile(indexFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load index file")
	}

	rf, err := repo.LoadFile(getReposFile(helmHome))
	if err != nil {
		return nil, errors.Wrap(err, "failed to load repositories file")
	}
	rf.Update(&c)

	i := search.NewIndex()
	for _, re := range rf.Repositories {
		n := re.Name
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

func getReposFile(helmHome string) string {
	return filepath.Join(helmHome, "repository", "repositories.yaml")
}

func getCachePath(helmHome string) string {
	return filepath.Join(helmHome, "repository", "cache")
}
