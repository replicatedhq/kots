package kustomize

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	logs "log"
	"time"

	"github.com/marccampbell/yaml-toolbox/pkg/splitter"
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

// DiffAppVersionsForDownstream will generate a diff of the rendered yaml between two different
// archivedirs
func DiffAppVersionsForDownstream(downstreamName string, archive string, diffBasePath string, kustomizeBinPath string) (*Diff, error) {
	// kustomize build both of these archives before diffing
	logs.Printf("LG: ---------- DiffAppVersionsForDownstream -------")
	oneStart := time.Now()
	archiveOutput, err := exec.Command(kustomizeBinPath, "build", filepath.Join(archive, "overlays", "downstreams", downstreamName)).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("kustomize stderr: %q", string(ee.Stderr))
		}
		return nil, errors.Wrap(err, "failed to run kustomize on archive dir")
	}
	oneDuration := time.Since(oneStart)
	logs.Printf("LG: duration one: %v", oneDuration)
	twoStart := time.Now()
	baseOutput, err := exec.Command(kustomizeBinPath, "build", filepath.Join(diffBasePath, "overlays", "downstreams", downstreamName)).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			err = fmt.Errorf("kustomize stderr: %q", string(ee.Stderr))
		}
		return nil, errors.Wrap(err, "failed to run kustomize on base dir")
	}
	twoDuration := time.Since(twoStart)
	logs.Printf("LG: duration two: %v", twoDuration)
	
	threeStart := time.Now()
	archiveFiles, err := splitter.SplitYAML(archiveOutput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split archive yaml")
	}

	baseFiles, err := splitter.SplitYAML(baseOutput)
	if err != nil {
		return nil, errors.Wrap(err, "failed to split base yaml")
	}
	threeDuration := time.Since(threeStart)
	logs.Printf("LG: duration three: %v", threeDuration)

	fourStart := time.Now()
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
	fourDuration := time.Since(fourStart)
	logs.Printf("LG: duration four: %v", fourDuration)

	fiveStart := time.Now()
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
	fiveDuration := time.Since(fiveStart)
	logs.Printf("LG: duration five: %v", fiveDuration)

	sixStart := time.Now()
	archiveStart := time.Now()
	_, archiveChartFiles, err := RenderChartsArchive(archive, downstreamName, kustomizeBinPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to kustomize archive charts dir")
	}
	archiveDuration := time.Since(archiveStart)
	logs.Printf("LG: archive duration: %v", archiveDuration)

	baseStart := time.Now()
	_, baseChartFiles, err := RenderChartsArchive(diffBasePath, downstreamName, kustomizeBinPath)
	if err != nil {
		return nil, errors.Wrap(err, "failed to kustomize base charts dir")
	}
	baseDuration := time.Since(baseStart)
	logs.Printf("LG: base duration: %v", baseDuration)
	
	sixDuration := time.Since(sixStart)
	logs.Printf("LG: duration six: %v", sixDuration)

	sevenStart := time.Now()
	for archiveFilename, archiveContents := range archiveChartFiles {
		baseContents, ok := baseChartFiles[archiveFilename]
		if !ok {
			// this file was added
			scanner := bufio.NewScanner(strings.NewReader(archiveContents))
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
	sevenDuration := time.Since(sevenStart)
	logs.Printf("LG: duration seven: %v", sevenDuration)

	eightStart := time.Now()
	for baseFilename, baseContents := range baseChartFiles {
		_, ok := archiveChartFiles[baseFilename]
		if !ok {
			// this file was removed
			scanner := bufio.NewScanner(strings.NewReader(baseContents))
			for scanner.Scan() {
				diff.LinesRemoved++
			}
			diff.FilesChanged++
		}
	}
	eightDuration := time.Since(eightStart)
	logs.Printf("LG: duration eight: %v", eightDuration)
	logs.Printf("LG: ---------------------------")
	return &diff, nil
}
