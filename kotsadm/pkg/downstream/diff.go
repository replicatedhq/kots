package downstream

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/marccampbell/yaml-toolbox/pkg/splitter"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/pkg/logger"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type Diff struct {
	FilesChanged int `json:"filesChanged"`
	LinesAdded   int `json:"linesAdded"`
	LinesRemoved int `json:"linesRemoved"`
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

// DiffAppVersionsForDownstream will generate a diff of the rendered yaml between two different
// archivedirs
func DiffAppVersionsForDownstream(downstreamName string, archive string, diffBasePath string, kustomizeVersion string) (*Diff, error) {
	// kustomize build both of these archives before diffing
	archiveOutput, err := exec.Command(fmt.Sprintf("kustomize%s", kustomizeVersion), "build", filepath.Join(archive, "overlays", "downstreams", downstreamName)).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			logger.Errorf("kustomize stderr: %q", string(ee.Stderr))
		}
		return nil, errors.Wrap(err, "failed to run kustomize on archive dir")
	}
	baseOutput, err := exec.Command(fmt.Sprintf("kustomize%s", kustomizeVersion), "build", filepath.Join(diffBasePath, "overlays", "downstreams", downstreamName)).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			logger.Errorf("kustomize stderr: %q", string(ee.Stderr))
		}
		return nil, errors.Wrap(err, "failed to run kustomize on base dir")
	}

	archiveFiles, err := splitter.SplitYAML(archiveOutput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split archive yaml")
	}
	baseFiles, err := splitter.SplitYAML(baseOutput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split base yaml")
	}

	diff := Diff{}

	for archiveFilename, archiveContents := range archiveFiles {
		baseContents, ok := baseFiles[archiveFilename]
		if !ok {
			// this file was added
			scanner := bufio.NewScanner(bytes.NewReader(archiveContents))
			for scanner.Scan() {
				diff.LinesAdded++
			}
			diff.FilesChanged++
			continue
		}

		linesAdded, linesRemoved, err := diffContent(string(archiveContents), string(baseContents))
		if err != nil {
			return nil, errors.Wrap(err, "failed to diff contents")
		}

		diff.LinesAdded += linesAdded
		diff.LinesRemoved += linesRemoved

		if diff.LinesAdded > 0 || diff.LinesRemoved > 0 {
			diff.FilesChanged++
		}
	}

	for baseFilename, baseContents := range baseFiles {
		_, ok := archiveFiles[baseFilename]
		if !ok {
			scanner := bufio.NewScanner(bytes.NewReader(baseContents))
			for scanner.Scan() {
				diff.LinesRemoved++
			}
			diff.FilesChanged++
		}
	}

	return &diff, nil
}
