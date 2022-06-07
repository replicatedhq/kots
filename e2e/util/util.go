package util

import (
	"os"
	"os/exec"

	//lint:ignore ST1001 since Ginkgo and Gomega are DSLs this makes the tests more natural to read
	. "github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/gexec"
)

func EnvOrDefault(key string, defaultVal string) string {
	val := os.Getenv(key)
	if val != "" {
		return val
	}
	return defaultVal
}

func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func RunCommand(cmd *exec.Cmd) (*gexec.Session, error) {
	return gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
}
