package updateworker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/replicatedhq/kotsadm/worker/pkg/ship"
	"github.com/replicatedhq/kotsadm/worker/pkg/store"
	"github.com/replicatedhq/kotsadm/worker/pkg/version"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type UpdateServer struct {
	Logger *zap.SugaredLogger
	Viper  *viper.Viper
	Store  store.Store
	Worker *Worker
}

type CreateUpdateRequest struct {
	ID      string `json:"id"`
	WatchID string `json:"watchId"`
}

func (s *UpdateServer) Serve(ctx context.Context, address string) error {
	g := gin.New()

	s.configureRoutes(g)

	server := http.Server{Addr: address, Handler: g}
	errChan := make(chan error)

	go func() {
		s.Logger.Infow("starting updateworker", zap.String("address", address))
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
	var createUpdateRequest CreateUpdateRequest
	if err := c.BindJSON(&createUpdateRequest); err != nil {
		s.Logger.Warnw("updateserver failed to read json request", zap.Error(err))
		return
	}

	shipUpdate, err := s.Store.GetUpdate(context.TODO(), createUpdateRequest.ID)
	if err != nil {
		s.Logger.Errorw("updateserver failed to get edit object", zap.Error(err))
		return
	}

	if err := s.Worker.deployUpdate(shipUpdate); err != nil {
		s.Logger.Errorw("updateserver failed to get uploadURL", zap.Error(err))
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
		if err == nil && response.StatusCode == http.StatusOK {
			c.Status(http.StatusCreated)
			return
		}
		if time.Now().Sub(start) > time.Duration(time.Second*30) {
			s.Logger.Errorw("update timeout creating edit worker", zap.Error(err))
			c.AbortWithStatus(http.StatusGatewayTimeout)
			return
		}

		time.Sleep(time.Millisecond * 100)
	}
}
