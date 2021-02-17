package snapshot

import (
	"fmt"
	"io"
	"regexp"
	"time"

	"github.com/go-logfmt/logfmt"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/api/snapshot/types"
	"github.com/replicatedhq/kots/pkg/logger"
)

var (
	stdoutPrefix = regexp.MustCompile(`^stdout: `)
	stderrPrefix = regexp.MustCompile(`^stderr: `)
)

func parseLogs(reader io.Reader) ([]types.SnapshotError, []types.SnapshotError, []*types.SnapshotHook, error) {
	errs := []types.SnapshotError{}
	warnings := []types.SnapshotError{}
	execs := []*types.SnapshotHook{}
	openExecs := map[string]*types.SnapshotHook{}

	d := logfmt.NewDecoder(reader)
	for d.ScanRecord() {
		line := map[string]string{}
		for d.ScanKeyval() {
			line[string(d.Key())] = string(d.Value())
		}

		if isExecBegin(line) {
			key := execKey(line)
			if open, ok := openExecs[key]; ok {
				// close out the existing exec with the same key
				execs = append(execs, open)
				delete(openExecs, key)
			}

			open := types.SnapshotHook{
				Name:          line["hookName"],
				Namespace:     line["namespace"],
				Phase:         line["hookPhase"],
				PodName:       line["name"],
				ContainerName: line["hookContainer"],
				Command:       line["hookCommand"],
			}

			if startedAt, err := time.Parse(time.RFC3339, line["time"]); err == nil {
				open.StartedAt = &startedAt
			} else {
				logger.Error(errors.Wrap(err, "failed to parse time"))
			}

			openExecs[key] = &open
			continue
		}

		if isExecStdout(line) {
			key := execKey(line)
			open, ok := openExecs[key]
			if !ok {
				// Dropping stdout from backup logs
				continue
			}

			open.Stdout = stdoutPrefix.ReplaceAllString(line["msg"], "")
			if finishedAt, err := time.Parse(time.RFC3339, line["time"]); err == nil {
				open.FinishedAt = &finishedAt
			} else {
				logger.Error(errors.Wrap(err, "failed to parse time"))
			}
			continue
		}

		if isExecStderr(line) {
			key := execKey(line)
			open, ok := openExecs[key]
			if !ok {
				// Dropping stderr from backup logs
				continue
			}

			open.Stderr = stderrPrefix.ReplaceAllString(line["msg"], "")
			if finishedAt, err := time.Parse(time.RFC3339, line["time"]); err == nil {
				open.FinishedAt = &finishedAt
			} else {
				logger.Error(errors.Wrap(err, "failed to parse time"))
			}
			continue
		}

		if isError(line) && isExec(line) {
			key := execKey(line)
			open, ok := openExecs[key]
			if !ok {
				// Dropping exec error from backup logs
				continue
			}

			open.Errors = append(open.Errors, types.SnapshotError{Title: line["msg"], Message: line["error"]})
		}

		if isWarning(line) && isExec(line) {
			key := execKey(line)
			open, ok := openExecs[key]
			if !ok {
				// Dropping exec error from backup logs
				continue
			}

			open.Warnings = append(open.Warnings, types.SnapshotError{Title: line["msg"], Message: line["error"]})
		}

		if isError(line) {
			errs = append(errs, types.SnapshotError{Title: line["msg"], Message: line["error"]})
		}

		if isWarning(line) {
			warnings = append(warnings, types.SnapshotError{Title: line["msg"], Message: line["error"]})
		}
	}

	for _, exec := range openExecs {
		execs = append(execs, exec)
	}

	return errs, warnings, execs, errors.Wrap(d.Err(), "failed to scan logs")
}

func execKey(line map[string]string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", line["namespace"], line["name"], line["hookPhase"], line["hookSource"], line["hookType"])
}

func isExecBegin(line map[string]string) bool {
	return line["msg"] == "running exec hook"
}

func isExecStdout(line map[string]string) bool {
	return stdoutPrefix.MatchString(line["msg"])
}

func isExecStderr(line map[string]string) bool {
	return stderrPrefix.MatchString(line["msg"])
}

func isError(line map[string]string) bool {
	return line["level"] == "error"
}

func isWarning(line map[string]string) bool {
	return line["level"] == "warning"
}

func isExec(line map[string]string) bool {
	return line["hookName"] != ""
}
