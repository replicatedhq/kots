package replicated_app

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/pact-foundation/pact-go/dsl"
	"github.com/pkg/errors"
)

func Test_CheckConnectivity(t *testing.T) {
	var test = func() (err error) {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/", pact.Server.Port), nil)
		if err != nil {
			return errors.Wrap(err, "create request")
		}

		_, err = http.DefaultClient.Do(req)
		if err != nil {
			return errors.Wrap(err, "execute request")
		}

		return nil
	}

	pact.AddInteraction().
		Given("Empty state").
		UponReceiving("Check connectivity").
		WithRequest(dsl.Request{
			Method: "GET",
			Path:   dsl.String("/"),
		}).
		WillRespondWith(dsl.Response{
			Status: 200,
		})

	if err := pact.Verify(test); err != nil {
		t.Fatalf("Error on Verify: %v", err)
	}
}
