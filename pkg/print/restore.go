package print

import (
	"encoding/json"
	"fmt"
	"time"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

func Restores(restores []velerov1.Restore, format string) {
	switch format {
	case "json":
		printRestoresJSON(restores)
	default:
		printRestoresTable(restores)
	}
}

func printRestoresJSON(restores []velerov1.Restore) {
	str, _ := json.MarshalIndent(restores, "", "    ")
	fmt.Println(string(str))
}

func printRestoresTable(restores []velerov1.Restore) {
	w := NewTabWriter()
	defer w.Flush()

	fmtColumns := "%s\t%s\t%s\t%s\t%s\t%s\t%s\n"
	fmt.Fprintf(w, fmtColumns, "NAME", "BACKUP", "STATUS", "STARTED", "COMPLETED", "ERRORS", "WARNINGS")
	for _, r := range restores {
		var startedAt *time.Time
		if r.Status.StartTimestamp != nil && !r.Status.StartTimestamp.Time.IsZero() {
			startedAt = &r.Status.StartTimestamp.Time
		}

		var completedAt *time.Time
		if r.Status.CompletionTimestamp != nil && !r.Status.CompletionTimestamp.Time.IsZero() {
			completedAt = &r.Status.CompletionTimestamp.Time
		}

		phase := r.Status.Phase
		if phase == "" {
			phase = "New"
		}

		fmt.Fprintf(w, fmtColumns, r.ObjectMeta.Name, r.Spec.BackupName, phase, startedAt, completedAt, fmt.Sprintf("%d", r.Status.Errors), fmt.Sprintf("%d", r.Status.Warnings))
	}
}
