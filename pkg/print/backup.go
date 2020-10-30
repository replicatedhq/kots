package print

import (
	"fmt"
	"time"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

func Backups(backups []velerov1.Backup) {
	w := NewTabWriter()
	defer w.Flush()

	fmtColumns := "%s\t%s\t%s\t%s\t%s\t%s\n"
	fmt.Fprintf(w, fmtColumns, "NAME", "STATUS", "ERRORS", "WARNINGS", "CREATED", "EXPIRES")
	for _, b := range backups {
		expiresAt := ""
		if b.Status.Expiration != nil {
			expiresAtDuration := b.Status.Expiration.Time.Sub(time.Now())
			expiresAt = fmt.Sprintf("%dd", uint64(expiresAtDuration.Hours()/24))
		}
		fmt.Fprintf(w, fmtColumns, b.ObjectMeta.Name, b.Status.Phase, fmt.Sprintf("%d", b.Status.Errors), fmt.Sprintf("%d", b.Status.Warnings), b.CreationTimestamp.Time, expiresAt)
	}
}
