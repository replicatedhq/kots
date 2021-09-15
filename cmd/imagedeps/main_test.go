package main

import (
	"io/ioutil"
	"path"
	"testing"
	"time"

	"github.com/google/go-github/v39/github"
	"github.com/stretchr/testify/require"
)

func makeReleases() []*github.RepositoryRelease {
	var releases []*github.RepositoryRelease
	tags := []string{
		"RELEASE.2021-09-09T21-37-07Z.fips",
		"RELEASE.2021-09-09T21-37-06Z.xxx",
		"RELEASE.2021-09-09T21-37-05Z",
		"RELEASE.2021-09-09T21-37-04Z",
	}
	tm := time.Now()
	for _, t := range tags {
		s := t
		r := github.RepositoryRelease{
			TagName:     &s,
			PublishedAt: &github.Timestamp{Time: tm},
		}
		releases = append(releases, &r)
		tm = tm.Add(time.Second * -1)
	}
	return releases
}

func TestFunctional(t *testing.T) {
	tt := []struct {
		name        string
		fn          tagFinderFn
		expectError bool
	}{
		{
			name: "basic",
			fn: getTagFinder(
				withGithubReleaseTagFinder(
					func(_ string, _ string) ([]*github.RepositoryRelease, error) {
						return makeReleases(), nil
					},
				),
			),
		},
		{
			name: "with-overrides",
			fn: getTagFinder(
				withRepoGetTags(
					func(_ string) ([]string, error) {
						return []string{
							"10.16", "10.17", "10.18",
							"10.19-zippy", "10.18-alpine", "10.16-alpine",
						}, nil
					},
				),
				withGithubReleaseTagFinder(
					func(_ string, _ string) ([]*github.RepositoryRelease, error) {
						return makeReleases(), nil
					},
				),
			),
		},
		{
			name: "postgres",
			fn: getTagFinder(
				withRepoGetTags(
					func(_ string) ([]string, error) {
						return []string{
							"10.16", "10.17", "10.18",
							"10.19-zippy", "10.18-alpine", "10.16-alpine",
						}, nil
					},
				),
			),
		},
		{
			name: "filter-github",
			fn: getTagFinder(
				withGithubReleaseTagFinder(
					func(_ string, _ string) ([]*github.RepositoryRelease, error) {
						return makeReleases(), nil
					},
				),
			),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {

			rootDir := path.Join("testdata", tc.name)
			expectedConstants, err := ioutil.ReadFile(path.Join(rootDir, "constants.go"))
			require.Nil(t, err)
			expectedEnvs, err := ioutil.ReadFile(path.Join(rootDir, ".image.env"))
			require.Nil(t, err)
			tempDir := t.TempDir()
			constantFile := path.Join(tempDir, "constants.go")
			envFile := path.Join(tempDir, ".image.env")
			inputSpec := path.Join(rootDir, "input-spec")
			ctx := generationContext{
				inputFilename:          inputSpec,
				outputConstantFilename: constantFile,
				outputEnvFilename:      envFile,
				tagFinderFn:            tc.fn,
			}

			err = generateTaggedImageFiles(ctx)
			if tc.expectError {
				require.NotNil(t, err)
				return
			}

			require.Nil(t, err)

			actualConstants, err := ioutil.ReadFile(constantFile)
			require.Nil(t, err)

			actualEnv, err := ioutil.ReadFile(envFile)
			require.Nil(t, err)

			require.Equal(t, string(expectedConstants), string(actualConstants))
			require.Equal(t, string(expectedEnvs), string(actualEnv))

		})
	}
}
