package pacts

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/pact-foundation/pact-go/dsl"
)

func Test_GetDesiredState(t *testing.T) {
	authHeader := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(":ship-cluster-no-applications")))

	var test = func() (err error) {
		u := fmt.Sprintf("http://localhost:%d/api/v1/deploy/desired", pact.Server.Port)
		req, err := http.NewRequest("GET", u, strings.NewReader(``))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", authHeader)

		if _, err = http.DefaultClient.Do(req); err != nil {
			return err
		}

		return nil
	}

	pact.AddInteraction().
		Given("Cluster ship-test-pact exists").
		UponReceiving("A request to list objects in a cluster").
		WithRequest(dsl.Request{
			Method: "GET",
			Path:   dsl.String("/api/v1/deploy/desired"),
			Headers: dsl.MapMatcher{
				"Authorization": dsl.String(authHeader),
			},
		}).
		WillRespondWith(dsl.Response{
			Status: 200,
			Body: DeployDesiredResponse{
				Present: []string{},
				Missing: []string{},
			},
		})

	if err := pact.Verify(test); err != nil {
		t.Fatalf("Error on Verify: %v", err)
	}
}
