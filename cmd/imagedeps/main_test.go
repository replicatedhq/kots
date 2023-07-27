package main

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/google/go-github/v39/github"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

var releaseTags = []string{
	"RELEASE.2022-06-11T19-55-32Z.fips",
	"RELEASE.2021-09-09T21-37-06Z.xxx",
	"RELEASE.2021-09-09T21-37-05Z",
	"RELEASE.2021-09-09T21-37-04Z",
}
var semVerTags = []string{
	"0.12.7", "0.12.6", "0.12.5",
	"0.12.4", "0.12.3", "0.12.2",
}

func makeReleases(tags []string) []*github.RepositoryRelease {
	var releases []*github.RepositoryRelease
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
		replacers   []*replacer
		expectError bool
	}{
		{
			name: "basic",
			fn: getTagFinder(
				withGithubReleaseTagFinder(
					func(_ string, _ string) ([]*github.RepositoryRelease, error) {
						return makeReleases(releaseTags), nil
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
							"0.13.2", "0.13.1",
							"0.12.7", "0.12.2",
						}, nil
					},
				),
				withGithubReleaseTagFinder(
					func(_ string, _ string) ([]*github.RepositoryRelease, error) {
						return makeReleases(releaseTags), nil
					},
				),
			),
			replacers: []*replacer{
				getMakefileReplacer("test.mk"),
				getDockerfileReplacer("test.Dockerfile"),
			},
		},
		{
			name: "rqlite",
			fn: getTagFinder(
				withRepoGetTags(
					func(_ string) ([]string, error) {
						return []string{
							"7.7.0", "7.6.1", "7.6.0",
							"6.10.2", "6.10.1", "6.8.2",
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
						return makeReleases(releaseTags), nil
					},
				),
			),
		},
		{
			name: "schemahero",
			fn: getTagFinder(
				withRepoGetTags(
					func(_ string) ([]string, error) {
						return semVerTags, nil
					},
				),
			),
		},
		{
			name: "lvp",
			fn: getTagFinder(
				withRepoGetTags(
					func(_ string) ([]string, error) {
						return []string{
							"v0.3.3",
						}, nil
					},
				),
			),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			rootDir := path.Join("testdata", tc.name)

			expectedConstants, err := os.ReadFile(path.Join(rootDir, "constants.go"))
			require.Nil(t, err)

			expectedEnvs, err := os.ReadFile(path.Join(rootDir, ".image.env"))
			require.Nil(t, err)

			tempDir := t.TempDir()
			constantFile := path.Join(tempDir, "constants.go")
			envFile := path.Join(tempDir, ".image.env")
			inputSpec := path.Join(rootDir, "input-spec")

			// since replacers will update the actual files, not create new ones, copy the files over to the tmp directory
			// and compare the results with the expected files
			if len(tc.replacers) > 0 {
				inputDir := path.Join(rootDir, "replacers", "input")
				outputDir := path.Join(tempDir, "replacers", "actual")

				err := copyDirFiles(inputDir, outputDir)
				require.Nil(t, err)

				// update replacers paths to point to the tmp dir
				for _, r := range tc.replacers {
					r.path = path.Join(outputDir, r.path)
				}
			}

			ctx := generationContext{
				inputFilename:          inputSpec,
				outputConstantFilename: constantFile,
				outputEnvFilename:      envFile,
				replacers:              tc.replacers,
				tagFinderFn:            tc.fn,
			}

			err = generateTaggedImageFiles(ctx)
			if tc.expectError {
				require.NotNil(t, err)
				return
			}

			require.Nil(t, err)

			actualConstants, err := os.ReadFile(constantFile)
			require.Nil(t, err)

			actualEnv, err := os.ReadFile(envFile)
			require.Nil(t, err)

			require.Equal(t, string(expectedConstants), string(actualConstants))
			require.Equal(t, string(expectedEnvs), string(actualEnv))

			if len(tc.replacers) > 0 {
				expectedDir := path.Join(rootDir, "replacers", "expected")
				actualDir := path.Join(tempDir, "replacers", "actual")

				files, err := os.ReadDir(expectedDir)
				require.Nil(t, err)

				for _, f := range files {
					expectedContent, err := os.ReadFile(path.Join(expectedDir, f.Name()))
					require.Nil(t, err)

					actualContent, err := os.ReadFile(path.Join(actualDir, f.Name()))
					require.Nil(t, err)

					require.Equal(t, string(expectedContent), string(actualContent))
				}
			}
		})
	}
}

func copyDirFiles(inputDir string, outputDir string) error {
	files, err := os.ReadDir(inputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to read input dir %s", inputDir)
	}

	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return errors.Wrapf(err, "failed to create output dir %s", outputDir)
	}

	for _, f := range files {
		content, err := os.ReadFile(path.Join(inputDir, f.Name()))
		if err != nil {
			return errors.Wrapf(err, "failed to read file %s", path.Join(inputDir, f.Name()))
		}

		err = os.WriteFile(path.Join(outputDir, f.Name()), []byte(content), 0644)
		if err != nil {
			return errors.Wrapf(err, "failed to write to file %s", path.Join(outputDir, f.Name()))
		}
	}

	return nil
}
