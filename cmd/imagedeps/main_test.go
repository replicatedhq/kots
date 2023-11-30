package main

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/google/go-github/v39/github"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

var (
	minioTags = []string{
		"sha256-00428f99c05677c91ad393c3017376e800d601708baa36e51091df3b9a67b324.att",
		"latest-dev",
		"latest",
		"0.20231025.063325-r0-dev",
		"0.20231025.063325-r0",
		"0.20231025.063325-dev",
		"0.20231025.063325",
		"0.20231025-dev",
		"0.20231025",
		"0.20230904.195737-r1-dev",
		"0.20230904.195737-r1",
		"0.20230904.195737-dev",
		"0.20230904.195737",
		"0.20230904-dev",
		"0.20230904",
		"0-dev",
		"0",
	}

	schemaheroTags = []string{
		"0.13.2",
		"0.13.1",
		"0.12.7",
		"0.12.2",
	}

	rqliteTags = []string{
		"sha256-00122e405b3fa3b5105b0468f1fb72dcb32474968a971c45906a702120d55b58.att",
		"latest-dev",
		"latest",
		"7",
		"7-dev",
		"7.7.0",
		"7.7.0-dev",
		"7.7.0-r2",
		"7.7.0-r2-dev",
		"7.6.2",
		"7.6.1",
		"7.6.0",
		"6.10.2",
		"6.10.1",
		"6.8.2",
	}

	dexTags = []string{
		"sha256-002adc734b3d83bb6be291b49eb8f3f95b905c411d404c2f4b52a759140739c9.att",
		"latest-dev",
		"latest",
		"2.37.0",
		"2.37.0-r3-dev",
		"2.37.0-r3",
		"2.37.0-dev",
		"2.36.0",
		"2.35.3",
		"2.35.2",
		"2.35.1",
	}

	lvpTags = []string{
		"v0.3.3",
		"v0.3.2",
		"v0.3.1",
	}
)

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
			name: "minio",
			fn: getTagFinder(
				withRepoGetTags(
					func(_ string) ([]string, error) {
						return minioTags, nil
					},
				),
			),
		},
		{
			name: "schemahero",
			fn: getTagFinder(
				withRepoGetTags(
					func(_ string) ([]string, error) {
						return schemaheroTags, nil
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
						return rqliteTags, nil
					},
				),
			),
		},
		{
			name: "dex",
			fn: getTagFinder(
				withRepoGetTags(
					func(_ string) ([]string, error) {
						return dexTags, nil
					},
				),
			),
		},
		{
			name: "lvp",
			fn: getTagFinder(
				withRepoGetTags(
					func(_ string) ([]string, error) {
						return lvpTags, nil
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

			actualConstants, err := ioutil.ReadFile(constantFile)
			require.Nil(t, err)

			actualEnv, err := ioutil.ReadFile(envFile)
			require.Nil(t, err)

			require.Equal(t, string(expectedConstants), string(actualConstants))
			require.Equal(t, string(expectedEnvs), string(actualEnv))

			if len(tc.replacers) > 0 {
				expectedDir := path.Join(rootDir, "replacers", "expected")
				actualDir := path.Join(tempDir, "replacers", "actual")

				files, err := ioutil.ReadDir(expectedDir)
				require.Nil(t, err)

				for _, f := range files {
					expectedContent, err := ioutil.ReadFile(path.Join(expectedDir, f.Name()))
					require.Nil(t, err)

					actualContent, err := ioutil.ReadFile(path.Join(actualDir, f.Name()))
					require.Nil(t, err)

					require.Equal(t, string(expectedContent), string(actualContent))
				}
			}
		})
	}
}

func copyDirFiles(inputDir string, outputDir string) error {
	files, err := ioutil.ReadDir(inputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to read input dir %s", inputDir)
	}

	err = os.MkdirAll(outputDir, 0755)
	if err != nil {
		return errors.Wrapf(err, "failed to create output dir %s", outputDir)
	}

	for _, f := range files {
		content, err := ioutil.ReadFile(path.Join(inputDir, f.Name()))
		if err != nil {
			return errors.Wrapf(err, "failed to read file %s", path.Join(inputDir, f.Name()))
		}

		err = ioutil.WriteFile(path.Join(outputDir, f.Name()), []byte(content), 0644)
		if err != nil {
			return errors.Wrapf(err, "failed to write to file %s", path.Join(outputDir, f.Name()))
		}
	}

	return nil
}
