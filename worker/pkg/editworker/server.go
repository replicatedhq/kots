package editworker

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

type EditServer struct {
	Logger log.Logger
	Viper  *viper.Viper
	Store  store.Store
	Worker *Worker
}

type CreateEditRequest struct {
	ID      string `json:"id"`
	WatchID string `json:"watchId"`
}

func (s *EditServer) Serve(ctx context.Context, address string) error {
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

func (s *EditServer) configureRoutes(g *gin.Engine) {
	root := g.Group("/")

	root.GET("/healthz", s.Healthz)
	root.GET("/metricz", s.Metricz)

	v1 := g.Group("/v1")
	v1.POST("/edit", s.CreateEditHandler)
}

// Healthz returns a 200 with the version if provided
func (s *EditServer) Healthz(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"server":    "edit",
		"version":   version.Version(),
		"sha":       version.GitSHA(),
		"buildTime": version.BuildTime(),
	})
}

// Metricz returns (empty) metrics for this server
func (s *EditServer) Metricz(c *gin.Context) {
	type Metric struct {
		M1  float64 `json:"m1"`
		P95 float64 `json:"p95"`
	}
	c.IndentedJSON(200, map[string]Metric{})
}

func (s *EditServer) CreateEditHandler(c *gin.Context) {
	debug := level.Debug(log.With(s.Logger, "method", "editworker.Server.CreateEditHandler"))

	var createEditRequest CreateEditRequest
	if err := c.BindJSON(&createEditRequest); err != nil {
		level.Warn(s.Logger).Log("bindJSON", err)
		return
	}

	debug.Log("event", "getedit", "id", createEditRequest.ID)

	shipEdit, err := s.Store.GetEdit(context.TODO(), createEditRequest.ID)
	if err != nil {
		level.Error(s.Logger).Log("getEdit", err)
		return
	}

	debug.Log("event", "set upload url", "id", shipEdit.ID)
	uploadURL, err := s.Store.GetS3StoreURL(shipEdit)
	if err != nil {
		level.Error(s.Logger).Log("getEditUploadURL", err)
		return
	}
	shipEdit.UploadURL = uploadURL

	debug.Log("event", "set output filepath", "watchId", shipEdit.WatchID, "sequence", shipEdit.UploadSequence)
	err = s.Store.SetOutputFilepath(context.TODO(), shipEdit)
	if err != nil {
		level.Error(s.Logger).Log("setEditOutputFilepath", err)
		return
	}

	debug.Log("event", "get namespace", "id", shipEdit.ID)
	namespace := ship.GetNamespace(context.TODO(), shipEdit)
	if err := s.Worker.ensureNamespace(context.TODO(), namespace); err != nil {
		level.Error(s.Logger).Log("ensureNamespace", err)
		return
	}

	networkPolicy := ship.GetNetworkPolicySpec(context.TODO(), shipEdit)
	if err := s.Worker.ensureNetworkPolicy(context.TODO(), networkPolicy); err != nil {
		level.Error(s.Logger).Log("networkPolicy", err)
		return
	}

	secret := ship.GetSecretSpec(context.TODO(), shipEdit, shipEdit.StateJSON)
	if err := s.Worker.ensureSecret(context.TODO(), secret); err != nil {
		level.Error(s.Logger).Log("ensureSecret", err)
		return
	}

	serviceAccount := ship.GetServiceAccountSpec(context.TODO(), shipEdit)
	if err := s.Worker.ensureServiceAccount(context.TODO(), serviceAccount); err != nil {
		level.Error(s.Logger).Log("ensureSecret", err)
		return
	}

	role := ship.GetRoleSpec(context.TODO(), shipEdit)
	if err := s.Worker.ensureRole(context.TODO(), role); err != nil {
		level.Error(s.Logger).Log("ensureRole", err)
		return
	}

	rolebinding := ship.GetRoleBindingSpec(context.TODO(), shipEdit)
	if err := s.Worker.ensureRoleBinding(context.TODO(), rolebinding); err != nil {
		level.Error(s.Logger).Log("ensureRoleBinding", err)
		return
	}

	pod := ship.GetPodSpec(context.TODO(), s.Worker.Config.LogLevel, s.Worker.Config.ShipImage, s.Worker.Config.ShipTag, s.Worker.Config.ShipPullPolicy, secret.Name, serviceAccount.Name, shipEdit, s.Worker.Config.GithubToken)
	if err := s.Worker.ensurePod(context.TODO(), pod); err != nil {
		level.Error(s.Logger).Log("ensurePod", err)
		return
	}

	// Wait for the pod to be ready here, or clean up and return an error

	service := ship.GetServiceSpec(context.TODO(), shipEdit)
	if err := s.Worker.ensureService(context.TODO(), service); err != nil {
		level.Error(s.Logger).Log("ensureService", err)
		return
	}

	if shipEdit.IsHeadless {
		c.Status(http.StatusCreated)
		return
	}

	// Block until the new service is responding
	quickClient := &http.Client{
		Timeout: time.Millisecond * 200,
	}

	start := time.Now()
	for {
		response, err := quickClient.Get(fmt.Sprintf("http://%s.%s.svc.cluster.local:8800/healthz", namespace.Name, service.Name))
		debug.Log("edit health err", err)
		if err == nil && response.StatusCode == http.StatusOK {
			debug.Log("edit health", response.StatusCode)
			c.Status(http.StatusCreated)
			return
		}
		if time.Now().Sub(start) > time.Duration(time.Second*30) {
			level.Error(s.Logger).Log("timeout creating edit worker", shipEdit.ID)
			c.AbortWithStatus(http.StatusGatewayTimeout)
			return
		}

		time.Sleep(time.Millisecond * 100)
	}
}
