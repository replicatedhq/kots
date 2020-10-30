package print

import (
	"fmt"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

func Restores(restores []velerov1.Restore) {
	w := NewTabWriter()
	defer w.Flush()

	fmtColumns := "%s\t%s\t%s\t%s\t%s\t%s\n"
	fmt.Fprintf(w, fmtColumns, "NAME", "BACKUP", "STATUS", "ERRORS", "WARNINGS", "CREATED")
	for _, r := range restores {
		fmt.Fprintf(w, fmtColumns, r.ObjectMeta.Name, r.Spec.BackupName, r.Status.Phase, fmt.Sprintf("%d", r.Status.Errors), fmt.Sprintf("%d", r.Status.Warnings), r.CreationTimestamp.Time)
	}
}
