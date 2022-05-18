package workspace

import (
	"os"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
)

type Workspace struct {
	dir string
}

func New() Workspace {
	dir, err := os.MkdirTemp("", "kots-e2e")
	Expect(err).WithOffset(1).Should(Succeed(), "create workspace")
	return Workspace{dir: dir}
}

func (w *Workspace) GetDir() string {
	return w.dir
}

func (w *Workspace) Teardown() error {
	return os.RemoveAll(w.dir)
}
