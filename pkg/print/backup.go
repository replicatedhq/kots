package print

import (
	"encoding/json"
	"fmt"
	"time"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

func Backups(backups []velerov1.Backup, format string) {
	switch format {
	case "json":
		printBackupsJSON(backups)
	default:
		printBackupsTable(backups)
	}
}

func printBackupsJSON(backups []velerov1.Backup) {
	str, _ := json.MarshalIndent(backups, "", "    ")
	fmt.Println(string(str))
}

func printBackupsTable(backups []velerov1.Backup) {
	w := NewTabWriter()
	defer w.Flush()

	fmtColumns := "%s\t%s\t%s\t%s\t%s\t%s\t%s\n"
	fmt.Fprintf(w, fmtColumns, "NAME", "STATUS", "ERRORS", "WARNINGS", "STARTED", "COMPLETED", "EXPIRES")
	for _, b := range backups {
		expiresAt := ""
		if b.Status.Expiration != nil {
			expiresAtDuration := b.Status.Expiration.Time.Sub(time.Now())
			expiresAt = fmt.Sprintf("%dd", uint64(expiresAtDuration.Hours()/24))
		}

		var startedAt *time.Time
		if b.Status.StartTimestamp != nil && !b.Status.StartTimestamp.Time.IsZero() {
			startedAt = &b.Status.StartTimestamp.Time
		}

		var completedAt *time.Time
		if b.Status.CompletionTimestamp != nil && !b.Status.CompletionTimestamp.Time.IsZero() {
			completedAt = &b.Status.CompletionTimestamp.Time
		}

		phase := b.Status.Phase
		if phase == "" {
			phase = "New"
		}

		fmt.Fprintf(w, fmtColumns, b.ObjectMeta.Name, phase, fmt.Sprintf("%d", b.Status.Errors), fmt.Sprintf("%d", b.Status.Warnings), startedAt, completedAt, expiresAt)
	}
}
