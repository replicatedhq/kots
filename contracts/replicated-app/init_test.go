package replicated_app

import (
	"os"
	"path"
	"testing"

	"github.com/pact-foundation/pact-go/dsl"
)

var (
	pact dsl.Pact
)

func TestMain(m *testing.M) {
	dir, _ := os.Getwd()

	pactDir := path.Join(dir, "..", "..", "pacts", "consumer")
	logDir := path.Join(dir, "..", "..", "pact_logs")

	pact = dsl.Pact{
		Consumer: "kots",
		Provider: "replicated-app",
		LogDir:   logDir,
		PactDir:  pactDir,
		LogLevel: "debug",
	}

	pact.Setup(true)

	code := m.Run()

	pact.WritePact()
	pact.Teardown()

	os.Exit(code)
}
