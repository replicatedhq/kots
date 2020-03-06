package multitype

import (
	"os"
	"testing"

	"go.undefinedlabs.com/scopeagent"
)

func TestMain(m *testing.M) {
	os.Exit(scopeagent.Run(m))
}
