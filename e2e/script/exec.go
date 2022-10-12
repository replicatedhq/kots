package script

import (
	"net/http"
	"os"
	"os/exec"
	"time"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/e2e/util"
)

func Execute(scriptFilename string, isAync bool, url string, httpStatusCode int) {
	_, err := os.Stat(scriptFilename)
	if os.IsNotExist(err) {
		return
	}

	Expect(err).WithOffset(1).Should(Succeed(), "check prerun script exists")

	exec := func() {
		util.RunCommand(exec.Command("/bin/bash", "-c", scriptFilename))
	}

	if !isAync {
		exec()
		return
	}

	go exec()
	if url == "" {
		return
	}

	for x := 0; x < 60; x++ {
		resp, e := http.DefaultClient.Get(url)
		if e != nil {
			err = e
		} else if resp.StatusCode != httpStatusCode {
			err = errors.Errorf("unexpected status code %v", resp.StatusCode)
		} else {
			return
		}
		time.Sleep(1 * time.Second)
	}
	Expect(err).WithOffset(1).Should(Succeed(), "prerun status check")
}
