package print

import (
	"fmt"
	"time"

	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

func Restores(restores []velerov1.Restore) {
	w := NewTabWriter()
	defer w.Flush()

	fmt.Fprintf(w, "%s\t%s\n", "NAME", "AGE")
	for _, restore := range restores {
		age := time.Now().Sub(restore.ObjectMeta.CreationTimestamp.Time)
		age = age.Round(time.Second)
		fmt.Fprintf(w, "%s\t%s\n", restore.ObjectMeta.Name, age.String())
	}
}
