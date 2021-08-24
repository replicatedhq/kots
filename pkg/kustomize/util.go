package kustomize

import (
	"fmt"
	"os"

	"github.com/replicatedhq/kots/pkg/persistence"
)

func GetKustomizePath(version string) string {
	if persistence.IsSQlite() {
		// in the kots run workflow, binaries exist under {kotsdatadir}/binaries
		return fmt.Sprintf("%s/binaries/kustomize%s", os.Getenv("KOTS_DATA_DIR"), version)
	}

	return fmt.Sprintf("kustomize%s", version)
}
