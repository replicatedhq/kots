package testim

import (
	"regexp"
	"time"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/ginkgo/v2"
	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var (
	TestimURLRegexp = regexp.MustCompile(`https://app.testim.io/[^\s]+`)
)

type Run struct {
	session *gexec.Session
}

func (r *Run) ShouldSucceed() {
	Eventually(r.session).WithOffset(1).WithTimeout(60*time.Minute).Should(gexec.Exit(), "Run testim tests timed out")
	Expect(r.session.ExitCode()).Should(Equal(0), "Run testim tests failed with non-zero exit code")
}

func (r *Run) PrintDebugInfo() {
	url := r.URL()
	if url == "" {
		return
	}
	GinkgoWriter.Printf("Testim run URL:\n  %s\n", url)
}

func (r *Run) URL() string {
	return TestimURLRegexp.FindString(string(r.session.Out.Contents()))
}
