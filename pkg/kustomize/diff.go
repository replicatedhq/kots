package kustomize

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/pkg/errors"
	"github.com/sergi/go-diff/diffmatchpatch"
)

type Diff struct {
	FilesChanged int `json:"filesChanged"`
	LinesAdded   int `json:"linesAdded"`
	LinesRemoved int `json:"linesRemoved"`
}

func diffContent(baseContent string, updatedContent string) (int, int, error) {
	dmp := diffmatchpatch.New()

	charsA, charsB, lines := dmp.DiffLinesToChars(baseContent, updatedContent)

	diffs := dmp.DiffMain(charsA, charsB, false)
	diffs = dmp.DiffCharsToLines(diffs, lines)

	additions := 0
	deletions := 0

	for _, diff := range diffs {
		scanner := bufio.NewScanner(strings.NewReader(diff.Text))
		for scanner.Scan() {
			if diff.Type == diffmatchpatch.DiffDelete {
				deletions++
			} else if diff.Type == diffmatchpatch.DiffInsert {
				additions++
			}
		}
	}

	return additions, deletions, nil
}

// DiffAppVersionsForDownstream will generate a diff of the rendered yaml between two different archive dirs
func DiffAppVersionsForDownstream(downstreamName string, archive string, diffBasePath string, kustomizeBinPath string) (*Diff, error) {
	_, archiveFiles, err := GetRenderedApp(archive, downstreamName, kustomizeBinPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rendered app")
	}

	_, baseFiles, err := GetRenderedApp(diffBasePath, downstreamName, kustomizeBinPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get base rendered app")
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

		linesAdded, linesRemoved, err := diffContent(string(baseContents), string(archiveContents))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff base and archive contents %s", archiveFilename)
		}

		diff.LinesAdded += linesAdded
		diff.LinesRemoved += linesRemoved

		if linesAdded > 0 || linesRemoved > 0 {
			diff.FilesChanged++
		}
	}

	for baseFilename, baseContents := range baseFiles {
		_, ok := archiveFiles[baseFilename]
		if !ok {
			// this file was removed
			scanner := bufio.NewScanner(bytes.NewReader(baseContents))
			for scanner.Scan() {
				diff.LinesRemoved++
			}
			diff.FilesChanged++
		}
	}

	_, archiveChartFiles, err := GetRenderedChartsArchive(archive, downstreamName, kustomizeBinPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rendered charts files")
	}

	_, baseChartFiles, err := GetRenderedChartsArchive(diffBasePath, downstreamName, kustomizeBinPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get base rendered charts files")
	}

	for archiveFilename, archiveContents := range archiveChartFiles {
		baseContents, ok := baseChartFiles[archiveFilename]
		if !ok {
			// this file was added
			scanner := bufio.NewScanner(bytes.NewReader(archiveContents))
			for scanner.Scan() {
				diff.LinesAdded++
			}
			diff.FilesChanged++
			continue
		}

		linesAdded, linesRemoved, err := diffContent(string(baseContents), string(archiveContents))
		if err != nil {
			return nil, errors.Wrapf(err, "failed to diff base and archive chart contents %s", archiveFilename)
		}

		diff.LinesAdded += linesAdded
		diff.LinesRemoved += linesRemoved

		if linesAdded > 0 || linesRemoved > 0 {
			diff.FilesChanged++
		}
	}

	for baseFilename, baseContents := range baseChartFiles {
		_, ok := archiveChartFiles[baseFilename]
		if !ok {
			// this file was removed
			scanner := bufio.NewScanner(bytes.NewReader(baseContents))
			for scanner.Scan() {
				diff.LinesRemoved++
			}
			diff.FilesChanged++
		}
	}
	return &diff, nil
}
