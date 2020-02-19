package app

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type Diff struct {
	FilesChanged int `json:"filesChanged"`
	LinesAdded   int `json:"linesAdded"`
	LinedRemoved int `json:"linesRemoved"`
}

func diffContent(updatedContent string, baseContent string) (int, int, error) {
	dmp := diffmatchpatch.New()

	charsA, charsB, lines := dmp.DiffLinesToChars(updatedContent, baseContent)

	diffs := dmp.DiffMain(charsA, charsB, false)
	diffs = dmp.DiffCharsToLines(diffs, lines)

	additions := 0
	deletions := 0

	for _, diff := range diffs {
		if diff.Type == diffmatchpatch.DiffDelete {
			deletions++
		} else if diff.Type == diffmatchpatch.DiffInsert {
			additions++
		}
	}

	return additions, deletions, nil
}

func diffAppVersionsForDownstreams(downstreamName string, archive string, diffBasePath string) (*Diff, error) {
	rootPathsToInclude := []string{
		"base",
		filepath.Join("overlays", "midstream"),
		filepath.Join("overlays", "downstreams", downstreamName),
	}

	diff := Diff{}

	err := filepath.Walk(archive,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if info.IsDir() {
				return nil
			}

			// we only diff files in base and the specific downstream
			isInSupportedRoot := false
			pathWithoutRoot := path[len(archive)+1:]
			for _, rootPathToInclude := range rootPathsToInclude {
				if strings.HasPrefix(pathWithoutRoot, rootPathToInclude) {
					isInSupportedRoot = true
				}
			}

			if !isInSupportedRoot {
				return nil
			}

			contentA, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}

			pathInDiffBase := filepath.Join(diffBasePath, path[len(archive):])
			contentB, err := ioutil.ReadFile(pathInDiffBase)
			if err != nil {
				return err
			}

			linedAdded, linesRemoved, err := diffContent(string(contentA), string(contentB))
			if err != nil {
				return err
			}

			diff.LinesAdded += linedAdded
			diff.LinedRemoved += linesRemoved

			if linedAdded > 0 || linesRemoved > 0 {
				diff.FilesChanged++
			}

			return nil
		})

	if err != nil {
		return nil, errors.Wrap(err, "failed to diff")
	}

	return &diff, nil
}
