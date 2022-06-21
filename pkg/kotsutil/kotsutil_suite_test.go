package kotsutil_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestKotsutil(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Kotsutil Suite")
}
