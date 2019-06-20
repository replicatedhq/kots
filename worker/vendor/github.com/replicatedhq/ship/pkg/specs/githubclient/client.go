package githubclient

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	errors2 "github.com/replicatedhq/ship/pkg/util/errors"

	// "github.com/spf13/afero"
	"golang.org/x/oauth2"
)

type GitHubFetcher interface {
	ResolveReleaseNotes(ctx context.Context, upstream string) (string, error)
	ResolveLatestRelease(ctx context.Context, upstream string) (string, error)
}

var _ GitHubFetcher = &GithubClient{}

type GithubClient struct {
	Logger log.Logger
	Client *github.Client
}

func NewGithubClient(logger log.Logger) *GithubClient {
	var httpClient *http.Client
	if accessToken := os.Getenv("GITHUB_TOKEN"); accessToken != "" {
		level.Debug(logger).Log("msg", "using github access token from environment")
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: accessToken},
		)
		httpClient = oauth2.NewClient(context.Background(), ts)
	}
	client := github.NewClient(httpClient)
	return &GithubClient{
		Client: client,
		Logger: logger,
	}
}

func (g *GithubClient) GetFiles(
	ctx context.Context,
	upstream string,
	destinationPath string,
) (string, error) {
	debug := level.Debug(log.With(g.Logger, "method", "getRepoContents"))

	debug.Log("event", "validateGithubURL")
	validatedUpstreamURL, err := validateGithubURL(upstream)
	if err != nil {
		return "", err
	}

	debug.Log("event", "decodeGithubURL")
	owner, repo, branch, repoPath, err := decodeGitHubURL(validatedUpstreamURL.Path)
	if err != nil {
		return "", err
	}

	debug.Log("event", "removeAll", "destinationPath", destinationPath)
	err = os.RemoveAll(destinationPath)
	if err != nil {
		return "", errors.Wrap(err, "remove chart clone destination")
	}

	downloadBasePath := ""
	if filepath.Ext(repoPath) != "" {
		downloadBasePath = repoPath
		repoPath = ""
	}
	err = g.downloadAndExtractFiles(ctx, owner, repo, branch, downloadBasePath, destinationPath)
	if err != nil {
		return "", errors2.FetchFilesError{Message: err.Error()}
	}

	return filepath.Join(destinationPath, repoPath), nil
}

func (g *GithubClient) downloadAndExtractFiles(
	ctx context.Context,
	owner string,
	repo string,
	branch string,
	basePath string,
	filePath string,
) error {
	debug := level.Debug(log.With(g.Logger, "method", "downloadAndExtractFiles"))

	debug.Log("event", "getContents", "path", basePath)

	archiveOpts := &github.RepositoryContentGetOptions{
		Ref: branch,
	}
	archiveLink, _, err := g.Client.Repositories.GetArchiveLink(ctx, owner, repo, github.Tarball, archiveOpts)
	if err != nil {
		return errors.Wrapf(err, "get archive link for owner - %s repo - %s", owner, repo)
	}

	resp, err := http.Get(archiveLink.String())
	if err != nil {
		return errors.Wrapf(err, "downloading archive")
	}
	defer resp.Body.Close()

	uncompressedStream, err := gzip.NewReader(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "create uncompressed stream")
	}

	tarReader := tar.NewReader(uncompressedStream)

	basePathFound := false
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			if !basePathFound {
				branchString := branch
				if branchString == "" {
					branchString = "master"
				}
				return errors.Errorf("Path %s in %s/%s on branch %s not found", basePath, owner, repo, branchString)
			}
			break
		}

		if err != nil {
			return errors.Wrapf(err, "extract tar gz, next()")
		}

		switch header.Typeflag {
		case tar.TypeReg:
			// need this in a func because defer in a loop was leaking handles
			err := func() error {
				fileName := strings.Join(strings.Split(header.Name, "/")[1:], "/")
				if !strings.HasPrefix(fileName, basePath) {
					return nil
				}
				basePathFound = true

				if fileName != basePath {
					fileName = strings.TrimPrefix(fileName, basePath)
				}
				dirPath, _ := path.Split(fileName)
				if err := os.MkdirAll(filepath.Join(filePath, dirPath), 0755); err != nil {
					return errors.Wrapf(err, "extract tar gz, mkdir")
				}
				outFile, err := os.Create(filepath.Join(filePath, fileName))
				if err != nil {
					return errors.Wrapf(err, "extract tar gz, create")
				}
				defer outFile.Close()
				if _, err := io.Copy(outFile, tarReader); err != nil {
					return errors.Wrapf(err, "extract tar gz, copy")
				}
				return nil
			}()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func decodeGitHubURL(chartPath string) (owner string, repo string, branch string, path string, err error) {
	splitPath := strings.Split(chartPath, "/")

	if len(splitPath) < 3 {
		return owner, repo, path, branch, errors.Wrapf(errors.New("unable to decode github url"), chartPath)
	}

	owner = splitPath[1]
	repo = splitPath[2]
	branch = ""
	path = ""
	if len(splitPath) > 3 {
		if splitPath[3] == "tree" || splitPath[3] == "blob" {
			branch = splitPath[4]
			path = strings.Join(splitPath[5:], "/")
		} else {
			path = strings.Join(splitPath[3:], "/")
		}
	}

	return owner, repo, branch, path, nil
}

func validateGithubURL(upstream string) (*url.URL, error) {
	if !strings.HasPrefix(upstream, "http") {

		upstream = fmt.Sprintf("http://%s", upstream)
	}

	upstreamURL, err := url.Parse(upstream)
	if err != nil {
		return nil, err
	}

	if !strings.Contains(upstreamURL.Host, "github.com") {
		return nil, errors.Errorf("%s is not a Github URL", upstream)
	}

	return upstreamURL, nil
}

func (g *GithubClient) ResolveReleaseNotes(ctx context.Context, upstream string) (string, error) {
	debug := level.Debug(log.With(g.Logger, "method", "ResolveReleaseNotes"))

	debug.Log("event", "validateGithubURL")
	validatedUpstreamURL, err := validateGithubURL(upstream)
	if err != nil {
		return "", errors.Wrap(err, "not a valid github url")
	}

	debug.Log("event", "decodeGithubURL")
	owner, repo, branch, repoPath, err := decodeGitHubURL(validatedUpstreamURL.Path)
	if err != nil {
		return "", err
	}

	commitList, _, err := g.Client.Repositories.ListCommits(ctx, owner, repo, &github.CommitsListOptions{
		SHA:  branch,
		Path: repoPath,
	})
	if err != nil {
		return "", err
	}

	if len(commitList) > 0 {
		latestRepoCommit := commitList[0]
		if latestRepoCommit != nil {
			commit := latestRepoCommit.GetCommit()
			if commit != nil {
				return commit.GetMessage(), nil
			}
		}
	}

	return "", errors.New("No commit available")
}

func (g *GithubClient) ResolveLatestRelease(ctx context.Context, upstream string) (string, error) {
	validatedUpstreamURL, err := validateGithubURL(upstream)
	if err != nil {
		return "", errors.Wrap(err, "not a valid github url")
	}

	owner, repo, _, _, err := decodeGitHubURL(validatedUpstreamURL.Path)
	if err != nil {
		return "", err
	}

	latest, _, err := g.Client.Repositories.GetLatestRelease(ctx, owner, repo)
	if err != nil {
		return "", errors.Wrap(err, "get latest release")
	}

	return latest.GetTagName(), nil
}
