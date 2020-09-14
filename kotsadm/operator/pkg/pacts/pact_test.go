package pacts

import (
	"fmt"
	"os"
	"testing"

	"github.com/pact-foundation/pact-go/dsl"
)

var (
	dir, _  = os.Getwd()
	pactDir = fmt.Sprintf("%s/../../pacts", dir)
	logDir  = fmt.Sprintf("%s/log", dir)

	pact dsl.Pact
)

type DeployDesiredResponse struct {
	Present []string `json:"present"`
	Missing []string `json:"missing"`
}

func TestMain(m *testing.M) {
	pact = createPact()
	pact.Setup(true)

	code := m.Run()

	pact.WritePact()
	pact.Teardown()

	os.Exit(code)
}

func createPact() dsl.Pact {
	dir, _ = os.Getwd()
	pactDir = fmt.Sprintf("%s/../../../../pacts", dir)
	logDir = fmt.Sprintf("%s/log", dir)

	return dsl.Pact{
		Consumer: "ship-cd",
		Provider: "kotsadm-api",
		LogDir:   logDir,
		PactDir:  pactDir,
		LogLevel: "debug",
	}
}
