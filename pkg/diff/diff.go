package diff

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kustomize"
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
	// diff kubernetes manifests
	_, archiveFiles, err := kustomize.GetRenderedApp(archive, downstreamName, kustomizeBinPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rendered app")
	}

	_, baseFiles, err := kustomize.GetRenderedApp(diffBasePath, downstreamName, kustomizeBinPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get base rendered app")
	}

	manifestsDiff, err := diffAppFiles(archiveFiles, baseFiles)
	if err != nil {
		return nil, errors.Wrap(err, "failed to diff app files")
	}

	// diff v1beta1 charts
	_, archiveV1Beta1ChartFiles, err := kustomize.GetRenderedChartsArchive(archive, downstreamName, kustomizeBinPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rendered charts files")
	}

	_, baseV1Beta1ChartFiles, err := kustomize.GetRenderedChartsArchive(diffBasePath, downstreamName, kustomizeBinPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get base rendered charts files")
	}

	v1Beta1ChartsDiff, err := diffAppFiles(archiveV1Beta1ChartFiles, baseV1Beta1ChartFiles)
	if err != nil {
		return nil, errors.Wrap(err, "failed to diff charts files")
	}

	// diff v1beta2 charts
	archiveV1Beta2ChartFiles, err := GetRenderedV1Beta2FileMap(archive, downstreamName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get rendered charts files")
	}

	baseV1Beta2ChartFiles, err := GetRenderedV1Beta2FileMap(diffBasePath, downstreamName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get base rendered charts files")
	}

	v1Beta2ChartsDiff, err := diffAppFiles(archiveV1Beta2ChartFiles, baseV1Beta2ChartFiles)
	if err != nil {
		return nil, errors.Wrap(err, "failed to diff charts files")
	}

	totalDiff := &Diff{
		FilesChanged: manifestsDiff.FilesChanged + v1Beta1ChartsDiff.FilesChanged + v1Beta2ChartsDiff.FilesChanged,
		LinesAdded:   manifestsDiff.LinesAdded + v1Beta1ChartsDiff.LinesAdded + v1Beta2ChartsDiff.LinesAdded,
		LinesRemoved: manifestsDiff.LinesRemoved + v1Beta1ChartsDiff.LinesRemoved + v1Beta2ChartsDiff.LinesRemoved,
	}

	return totalDiff, nil
}

func diffAppFiles(archive map[string][]byte, base map[string][]byte) (*Diff, error) {
	diff := Diff{}
	for archiveFilename, archiveContents := range archive {
		baseContents, ok := base[archiveFilename]
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

	for baseFilename, baseContents := range base {
		_, ok := archive[baseFilename]
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
