package analyzeworker

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/replicatedhq/kotsadm/worker/pkg/version"
)

func Serve(ctx context.Context, addr string) error {
	g := gin.New()

	root := g.Group("/")

	root.GET("/healthz", Healthz)
	root.GET("/metricz", Metricz)

	server := http.Server{Addr: addr, Handler: g}
	errChan := make(chan error)

	go func() {
		errChan <- server.ListenAndServe()
	}()

	return nil
}

func Healthz(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"server":    "init",
		"version":   version.Version(),
		"sha":       version.GitSHA(),
		"buildTime": version.BuildTime(),
	})
}

// Metricz returns (empty) metrics for this server
func Metricz(c *gin.Context) {
	type Metric struct {
		M1  float64 `json:"m1"`
		P95 float64 `json:"p95"`
	}
	c.IndentedJSON(200, map[string]Metric{})
}
