package updateworker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship-cluster/worker/pkg/ship"
	"github.com/replicatedhq/ship-cluster/worker/pkg/store"
	"github.com/replicatedhq/ship-cluster/worker/pkg/version"
	"github.com/spf13/viper"
)

type UpdateServer struct {
	Logger log.Logger
	Viper  *viper.Viper
	Store  store.Store
	Worker *Worker
}

type CreateUpdateRequest struct {
	ID      string `json:"id"`
	WatchID string `json:"watchId"`
}

func (s *UpdateServer) Serve(ctx context.Context, address string) error {
	debug := level.Debug(log.With(s.Logger, "method", "serve"))

	g := gin.New()

	debug.Log("event", "routes.configure")
	s.configureRoutes(g)

	server := http.Server{Addr: address, Handler: g}
	errChan := make(chan error)

	go func() {
		debug.Log("event", "server.listen", "server.address", address)
		errChan <- server.ListenAndServe()
	}()

	return nil
}

func (s *UpdateServer) configureRoutes(g *gin.Engine) {
	root := g.Group("/")

	root.GET("/healthz", s.Healthz)
	root.GET("/metricz", s.Metricz)

	v1 := g.Group("/v1")
	v1.POST("/update", s.CreateUpdateHandler)
}

// Healthz returns a 200 with the version if provided
func (s *UpdateServer) Healthz(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"server":    "update",
		"version":   version.Version(),
		"sha":       version.GitSHA(),
		"buildTime": version.BuildTime(),
	})
}

// Metricz returns (empty) metrics for this server
func (s *UpdateServer) Metricz(c *gin.Context) {
	type Metric struct {
		M1  float64 `json:"m1"`
		P95 float64 `json:"p95"`
	}
	c.IndentedJSON(200, map[string]Metric{})
}

func (s *UpdateServer) CreateUpdateHandler(c *gin.Context) {
	debug := level.Debug(log.With(s.Logger, "method", "updateworker.Server.CreateUpdateHandler"))

	var createUpdateRequest CreateUpdateRequest
	if err := c.BindJSON(&createUpdateRequest); err != nil {
		level.Warn(s.Logger).Log("bindJSON", err)
		return
	}

	debug.Log("event", "getupdate", "id", createUpdateRequest.ID)

	shipUpdate, err := s.Store.GetUpdate(context.TODO(), createUpdateRequest.ID)
	if err != nil {
		level.Error(s.Logger).Log("getUpdate", err)
		return
	}

	if err := s.Worker.deployUpdate(shipUpdate); err != nil {
		level.Error(s.Logger).Log("deployUpdate", err)
		return
	}

	// Block until the new service is responding
	quickClient := &http.Client{
		Timeout: time.Millisecond * 200,
	}

	namespace := ship.GetNamespace(context.TODO(), shipUpdate)
	service := ship.GetServiceSpec(context.TODO(), shipUpdate)

	start := time.Now()
	for {
		response, err := quickClient.Get(fmt.Sprintf("http://%s.%s.svc.cluster.local:8800/healthz", namespace.Name, service.Name))
		debug.Log("update health err", err)
		if err == nil && response.StatusCode == http.StatusOK {
			debug.Log("update health", response.StatusCode)
			c.Status(http.StatusCreated)
			return
		}
		if time.Now().Sub(start) > time.Duration(time.Second*30) {
			level.Error(s.Logger).Log("timeout creating update worker", shipUpdate.ID)
			c.AbortWithStatus(http.StatusGatewayTimeout)
			return
		}

		time.Sleep(time.Millisecond * 100)
	}
}
