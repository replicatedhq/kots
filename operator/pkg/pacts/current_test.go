package pacts

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/pact-foundation/pact-go/dsl"
	_ "go.undefinedlabs.com/scopeagent/autoinstrument"
)

func Test_PostCurrentState(t *testing.T) {
	authHeader := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(":ship-cluster-with-helm-charts")))
	parsedTime, _ := time.Parse(time.RFC3339, "2019-01-01T01:01:01+00:00")
	helmApplication := struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
		Version   int32  `json:"version"`

		FirstDeployedAt time.Time `json:"firstDeployedAt"`
		LastDeployedAt  time.Time `json:"lastDeployedAt"`
		IsDeleted       bool      `json:"isDeleted"`

		ChartVersion string   `json:"chartVersion"`
		AppVersion   string   `json:"appVersion"`
		Sources      []string `json:"sources"`

		Values map[string]interface{} `json:"values"`
	}{
		Name:            "helm-chart-name",
		Namespace:       "default",
		Version:         1,
		FirstDeployedAt: parsedTime,
		LastDeployedAt:  parsedTime,
		IsDeleted:       false,
		ChartVersion:    "0.0.1",
		AppVersion:      "1.0.0",
		Sources: []string{
			"githhub.com/helm/charts/stable/fake",
		},
		Values: map[string]interface{}{},
	}
	b, _ := json.Marshal(helmApplication)

	var test = func() (err error) {

		u := fmt.Sprintf("http://localhost:%d/api/v1/current", pact.Server.Port)
		req, err := http.NewRequest("POST", u, strings.NewReader(string(b)))
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
		Given("A detected helm chart").
		UponReceiving("A serialized helm application").
		WithRequest(dsl.Request{
			Method: "POST",
			Path:   dsl.String("/api/v1/current"),
			Headers: dsl.MapMatcher{
				"Authorization": dsl.String(authHeader),
			},
			Body: helmApplication,
		}).
		WillRespondWith(dsl.Response{
			Status: 200,
			Body:   `{}`,
		})

	if err := pact.Verify(test); err != nil {
		t.Fatalf("Error on Verify: %v", err)
	}
}
