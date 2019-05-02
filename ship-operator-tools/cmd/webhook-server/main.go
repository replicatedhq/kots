// This is a dev server to test webhook delivery.
package main

import (
	"net/http"

	"github.com/replicatedhq/ship-operator-tools/pkg/webhook"
	"github.com/replicatedhq/ship-operator/pkg/logger"
)

func main() {
	h := webhook.NewHandler("/tmp", logger.FromEnv())
	http.ListenAndServe(":5419", h)
}
