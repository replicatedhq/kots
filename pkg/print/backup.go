package print

import (
	"fmt"
	"time"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

func Backups(backups []velerov1.Backup) {
	w := NewTabWriter()
	defer w.Flush()

	fmt.Fprintf(w, "%s\t%s\n", "NAME", "AGE")
	for _, backup := range backups {
		age := time.Now().Sub(backup.ObjectMeta.CreationTimestamp.Time)
		age = age.Round(time.Second)
		fmt.Fprintf(w, "%s\t%s\n", backup.ObjectMeta.Name, age.String())
	}
}
