package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	semver "github.com/Masterminds/semver/v3"
	"github.com/google/go-github/v39/github"
	"github.com/heroku/docker-registry-client/registry"
	"golang.org/x/oauth2"
)

type ImageRef struct {
	name      string
	reference string
	tag       string
}

// GetDeclarationLine produces a line of text intended for use in a Go constant declaration file.
func (ir ImageRef) GetDeclarationLine() string {
	return fmt.Sprintf("\t%s = \"%s:%s\"", getConstantName(ir.name), ir.reference, ir.tag)
}

// GetEnvironmentLine generates a line of text intended for use in an .env file.
func (ir ImageRef) GetEnvironmentLine() string {
	return fmt.Sprintf("%s='%s'", getEnvironmentName(ir.name), ir.tag)
}

// GetMakefileLine generates a line of text intended for use in a Makefile file.
func (ir ImageRef) GetMakefileLine() string {
	return fmt.Sprintf("%s ?= %s", getMakefileVarName(ir.name), ir.tag)
}

// GetDockerfileLine generates a line of text intended for use in a Dockerfile file.
func (ir ImageRef) GetDockerfileLine() string {
	return fmt.Sprintf("ARG %s=%s", getDockerfileVarName(ir.name), ir.tag)
}

type getTagsFn func(string) ([]string, error)
type getReleaseFn func(string, string) ([]*github.RepositoryRelease, error)
type tagFinderFn func(inputLine string) (*ImageRef, error)

type filterFn func(tag string) bool

func getFilter(expression string) (filterFn, error) {
	expr, err := regexp.Compile(expression)
	if err != nil {
		return nil, err
	}
	return func(tag string) bool {
		if expr.MatchString(tag) {
			return true
		}
		return false
	}, err
}

// persists function pointers to external resources, github, registries etc.
type configuration struct {
	repositoryTagsFinder getTagsFn
	releaseFinder        getReleaseFn
}

// pass to getTagFinder to override the repository tag finder
func withRepoGetTags(fn getTagsFn) func(c *configuration) {
	return func(c *configuration) {
		c.repositoryTagsFinder = fn
	}
}

// pass to getTagFinder to override the github release tag finder.
func withGithubReleaseTagFinder(fn getReleaseFn) func(c *configuration) {
	return func(c *configuration) {
		c.releaseFinder = fn
	}
}

// returns a tag finder function that returns information about an image and it's latest tag.
func getTagFinder(opts ...func(c *configuration)) tagFinderFn {
	// set defaults
	config := configuration{
		repositoryTagsFinder: getRegistryTags,
		releaseFinder:        getReleases,
	}
	// apply options
	for _, opt := range opts {
		opt(&config)
	}

	return func(inputLine string) (*ImageRef, error) {
		parts := strings.Split(inputLine, " ")
		if len(parts) < 2 {
			return nil, fmt.Errorf("malformed input line %q", inputLine)
		}

		// everything matches by default
		matcherFn := func(_ string) bool {
			return true
		}

		var err error

		// if we have three parts, the third is regular expression for the matcher
		if len(parts) > 2 {
			matcherFn, err = getFilter(parts[2])
			if err != nil {
				return nil, fmt.Errorf("unable to parse regex %q from input file %w", parts[2], err)
			}
		}

		imageName, untaggedRef := parts[0], parts[1]
		var latestReleaseTag string

		switch imageName {
		case minioReference:
			latestReleaseTag, err = getLatestTagFromRegistry("kotsadm/minio", config.repositoryTagsFinder, matcherFn)
			if err != nil {
				return nil, fmt.Errorf("failed to get release tag for %s %w", imageName, err)
			}
		case dexReference:
			latestReleaseTag, err = getLatestTagFromRegistry("kotsadm/dex", config.repositoryTagsFinder, matcherFn)
			if err != nil {
				return nil, fmt.Errorf("failed to get release tag for %s %w", imageName, err)
			}
		case rqliteReference:
			latestReleaseTag, err = getLatestTagFromRegistry("kotsadm/rqlite", config.repositoryTagsFinder, matcherFn)
			if err != nil {
				return nil, fmt.Errorf("failed to get release tag for %s %w", imageName, err)
			}
		case schemaheroReference:
			latestReleaseTag, err = getLatestTagFromRegistry("schemahero/schemahero", config.repositoryTagsFinder, matcherFn)
			if err != nil {
				return nil, fmt.Errorf("failed to get release tag for %s %w", imageName, err)
			}
		case lvpReference:
			latestReleaseTag, err = getLatestTagFromRegistry("replicated/local-volume-provider", config.repositoryTagsFinder, matcherFn)
			if err != nil {
				return nil, fmt.Errorf("failed to get release tag for %s %w", imageName, err)
			}

		default:
			return nil, fmt.Errorf("don't know how to deal with %q image", imageName)
		}

		return &ImageRef{
			name:      imageName,
			reference: untaggedRef,
			tag:       latestReleaseTag,
		}, nil
	}

}

type GithubReleaseSorter []*github.RepositoryRelease

func (r GithubReleaseSorter) Len() int { return len(r) }

func (r GithubReleaseSorter) Less(i, j int) bool {
	return r[i].PublishedAt.Before(r[j].PublishedAt.Time)
}

func (r GithubReleaseSorter) Swap(i, j int) {
	tmp := r[i]
	r[i] = r[j]
	r[j] = tmp
}

func getLatestTagFromRegistry(imageUri string, getTags getTagsFn, match filterFn) (string, error) {
	tags, err := getTags(imageUri)
	if err != nil {
		return "", err
	}

	var versions []*semver.Version

	for _, tag := range tags {
		if match(tag) {
			v, err := semver.NewVersion(tag)
			if err != nil {
				return "", err
			}
			versions = append(versions, v)
		}
	}
	sort.Sort(semver.Collection(versions))
	if len(versions) == 0 {
		return "", fmt.Errorf("no versions found")
	}

	return versions[len(versions)-1].Original(), nil
}

func getLatestTagFromGithub(getReleases getReleaseFn, owner, repo string, match filterFn) (string, error) {
	releases, err := getReleases(owner, repo)
	if err != nil {
		return "", err
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("no releases found for %s/%s", owner, repo)
	}

	var matches []*github.RepositoryRelease
	for _, release := range releases {
		if release.TagName != nil && match(*release.TagName) {
			// exclude pre-releases
			if release.Prerelease != nil && *release.Prerelease {
				continue
			}
			matches = append(matches, release)
		}
	}

	s := GithubReleaseSorter(matches)
	sort.Sort(s)
	return *s[len(s)-1].TagName, nil
}

func getReleases(owner, repo string) ([]*github.RepositoryRelease, error) {
	var httpClient *http.Client
	if token, ok := os.LookupEnv(githubAuthTokenEnvironmentVarName); ok {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		httpClient = oauth2.NewClient(context.Background(), ts)
	}

	client := github.NewClient(httpClient)
	var releases []*github.RepositoryRelease
	listOptions := github.ListOptions{
		Page:    0,
		PerPage: githubPageSize,
	}
	for {
		page, response, err := client.Repositories.ListReleases(context.Background(), owner, repo, &listOptions)
		if err != nil {
			return nil, err
		}
		if len(page) > 0 {
			releases = append(releases, page...)
		}
		if response.NextPage == 0 {
			break
		}
		listOptions.Page = response.NextPage
	}

	return releases, nil
}

// getRegistryTags queries a Docker Registry HTTP API V2 compliant registry to get the tags for an image.
func getRegistryTags(untaggedRef string) ([]string, error) {
	registryUri := dockerRegistryUrl
	imageRef := untaggedRef
	userName, password := "", ""
	parts := strings.Split(untaggedRef, "/")
	if len(parts) > 2 {
		registryUri = fmt.Sprintf("https://%s", parts[0])
		imageRef = path.Join(parts[1:]...)
	}
	hub, err := registry.New(registryUri, userName, password)
	if err != nil {
		return nil, fmt.Errorf("could not connect to registry %q %w", registryUri, err)
	}
	tags, err := hub.Tags(imageRef)
	if err != nil {
		return nil, fmt.Errorf("could not fetch tags for image %q %w", imageRef, err)
	}
	return tags, nil
}
