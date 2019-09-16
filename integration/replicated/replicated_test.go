package replicated

import (
	"testing"

	"github.com/replicatedhq/kots/integration/replicated/pull"
)

const endpoint = "http://localhost:3000"

func TestMain(m *testing.M) {
	stopCh, err := pull.StartMockServer(endpoint, "integration", "integration", []byte(""))
	if err != nil {
		panic(err)
	}

	defer func() {
		stopCh <- true
	}()

	m.Run()
}
