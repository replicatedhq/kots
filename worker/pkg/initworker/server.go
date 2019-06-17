package initworker

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

type InitServer struct {
	Logger log.Logger
	Viper  *viper.Viper
	Store  store.Store
	Worker *Worker
}

type CreateSessionRequest struct {
	ID          string `json:"id"`
	UpstreamURI string `json:"upstreamUri"`
	ForkURI     string `json:"forkUri"`
}

func (s *InitServer) Serve(ctx context.Context, addr string) error {
	debug := level.Debug(log.With(s.Logger, "method", "serve"))

	g := gin.New()

	debug.Log("event", "routes.configure")
	s.configureRoutes(g)

	server := http.Server{Addr: addr, Handler: g}
	errChan := make(chan error)

	go func() {
		debug.Log("event", "server.listen", "server.addr", addr)
		errChan <- server.ListenAndServe()
	}()

	return nil
}

func (s *InitServer) configureRoutes(g *gin.Engine) {
	root := g.Group("/")

	root.GET("/healthz", s.Healthz)
	root.GET("/metricz", s.Metricz)

	v1 := g.Group("/v1")
	v1.POST("/init", s.CreateInitHandler)
	v1.POST("/unfork", s.CreateUnforkHandler)
}

// Healthz returns a 200 with the version if provided
func (s *InitServer) Healthz(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"server":    "init",
		"version":   version.Version(),
		"sha":       version.GitSHA(),
		"buildTime": version.BuildTime(),
	})
}

// Metricz returns (empty) metrics for this server
func (s *InitServer) Metricz(c *gin.Context) {
	type Metric struct {
		M1  float64 `json:"m1"`
		P95 float64 `json:"p95"`
	}
	c.IndentedJSON(200, map[string]Metric{})
}

func (s *InitServer) CreateUnforkHandler(c *gin.Context) {
	debug := level.Debug(log.With(s.Logger, "method", "initworker.Server.CreateInitHandler"))

	var createUnforkRequest CreateSessionRequest
	if err := c.BindJSON(&createUnforkRequest); err != nil {
		level.Warn(s.Logger).Log("bindJSON", err)
		return
	}

	debug.Log("event", "getunfork", "id", createUnforkRequest.ID)

	shipUnfork, err := s.Store.GetUnfork(context.TODO(), createUnforkRequest.ID)
	if err != nil {
		level.Error(s.Logger).Log("getInit", err)
		return
	}

	debug.Log("event", "set upload url", "id", shipUnfork.ID)
	uploadURL, err := s.Store.GetS3StoreURL(shipUnfork)
	if err != nil {
		level.Error(s.Logger).Log("getInitUploadURL", err)
		return
	}
	shipUnfork.UploadURL = uploadURL

	debug.Log("event", "set output filepath", "id", shipUnfork.ID)
	err = s.Store.SetOutputFilepath(context.TODO(), shipUnfork)
	if err != nil {
		level.Error(s.Logger).Log("setUnforkFilePath", err)
		return
	}

	debug.Log("event", "get namespace", "id", shipUnfork.ID)
	namespace := ship.GetNamespace(context.TODO(), shipUnfork)
	if err := s.Worker.ensureNamespace(context.TODO(), namespace); err != nil {
		level.Error(s.Logger).Log("ensureNamespace", err)
		return
	}

	networkPolicy := ship.GetNetworkPolicySpec(context.TODO(), shipUnfork)
	if err := s.Worker.ensureNetworkPolicy(context.TODO(), networkPolicy); err != nil {
		level.Error(s.Logger).Log("ensureNetworkPolicy", err)
		return
	}

	secret := ship.GetSecretSpec(context.TODO(), shipUnfork, []byte(""))
	if err := s.Worker.ensureSecret(context.TODO(), secret); err != nil {
		level.Error(s.Logger).Log("ensureSecret", err)
		return
	}

	serviceAccount := ship.GetServiceAccountSpec(context.TODO(), shipUnfork)
	if err := s.Worker.ensureServiceAccount(context.TODO(), serviceAccount); err != nil {
		level.Error(s.Logger).Log("ensureSecret", err)
		return
	}

	role := ship.GetRoleSpec(context.TODO(), shipUnfork)
	if err := s.Worker.ensureRole(context.TODO(), role); err != nil {
		level.Error(s.Logger).Log("ensureRole", err)
		return
	}

	rolebinding := ship.GetRoleBindingSpec(context.TODO(), shipUnfork)
	if err := s.Worker.ensureRoleBinding(context.TODO(), rolebinding); err != nil {
		level.Error(s.Logger).Log("ensureRoleBinding", err)
		return
	}

	pod := ship.GetPodSpec(context.TODO(), s.Worker.Config.LogLevel, s.Worker.Config.ShipImage, s.Worker.Config.ShipTag, s.Worker.Config.ShipPullPolicy, secret.Name, serviceAccount.Name, shipUnfork, s.Worker.Config.GithubToken)
	if err := s.Worker.ensurePod(context.TODO(), pod); err != nil {
		level.Error(s.Logger).Log("ensurePod", err)
		return
	}

	// Wait for the pod to be ready here, or clean up and return an error

	service := ship.GetServiceSpec(context.TODO(), shipUnfork)
	if err := s.Worker.ensureService(context.TODO(), service); err != nil {
		level.Error(s.Logger).Log("ensureService", err)
		return
	}

	// Unfork runs in headless mode, so there's no port in ship we can use
	// to monitor success.

	// We are relying on informers to alert when this has completed
}

func (s *InitServer) CreateInitHandler(c *gin.Context) {
	debug := level.Debug(log.With(s.Logger, "method", "initworker.Server.CreateInitHandler"))

	var createInitRequest CreateSessionRequest
	if err := c.BindJSON(&createInitRequest); err != nil {
		level.Warn(s.Logger).Log("bindJSON", err)
		return
	}

	debug.Log("event", "getinit", "id", createInitRequest.ID)

	shipInit, err := s.Store.GetInit(context.TODO(), createInitRequest.ID)
	if err != nil {
		level.Error(s.Logger).Log("getInit", err)
		return
	}

	debug.Log("event", "set upload url", "id", shipInit.ID)
	uploadURL, err := s.Store.GetS3StoreURL(shipInit)
	if err != nil {
		level.Error(s.Logger).Log("getInitUploadURL", err)
		return
	}
	shipInit.UploadURL = uploadURL

	debug.Log("event", "set output filepath", "id", shipInit.ID)
	err = s.Store.SetOutputFilepath(context.TODO(), shipInit)
	if err != nil {
		level.Error(s.Logger).Log("setInitOutputFilepath", err)
		return
	}

	debug.Log("event", "get namespace", "id", shipInit.ID)
	namespace := ship.GetNamespace(context.TODO(), shipInit)
	if err := s.Worker.ensureNamespace(context.TODO(), namespace); err != nil {
		level.Error(s.Logger).Log("ensureNamespace", err)
		return
	}

	networkPolicy := ship.GetNetworkPolicySpec(context.TODO(), shipInit)
	if err := s.Worker.ensureNetworkPolicy(context.TODO(), networkPolicy); err != nil {
		level.Error(s.Logger).Log("ensureNetworkPolicy", err)
		return
	}

	secret := ship.GetSecretSpec(context.TODO(), shipInit, []byte(""))
	if err := s.Worker.ensureSecret(context.TODO(), secret); err != nil {
		level.Error(s.Logger).Log("ensureSecret", err)
		return
	}

	serviceAccount := ship.GetServiceAccountSpec(context.TODO(), shipInit)
	if err := s.Worker.ensureServiceAccount(context.TODO(), serviceAccount); err != nil {
		level.Error(s.Logger).Log("ensureSecret", err)
		return
	}

	role := ship.GetRoleSpec(context.TODO(), shipInit)
	if err := s.Worker.ensureRole(context.TODO(), role); err != nil {
		level.Error(s.Logger).Log("ensureRole", err)
		return
	}

	rolebinding := ship.GetRoleBindingSpec(context.TODO(), shipInit)
	if err := s.Worker.ensureRoleBinding(context.TODO(), rolebinding); err != nil {
		level.Error(s.Logger).Log("ensureRoleBinding", err)
		return
	}

	pod := ship.GetPodSpec(context.TODO(), s.Worker.Config.LogLevel, s.Worker.Config.ShipImage, s.Worker.Config.ShipTag, s.Worker.Config.ShipPullPolicy, secret.Name, serviceAccount.Name, shipInit, s.Worker.Config.GithubToken)
	if err := s.Worker.ensurePod(context.TODO(), pod); err != nil {
		level.Error(s.Logger).Log("ensurePod", err)
		return
	}

	// Wait for the pod to be ready here, or clean up and return an error

	service := ship.GetServiceSpec(context.TODO(), shipInit)
	if err := s.Worker.ensureService(context.TODO(), service); err != nil {
		level.Error(s.Logger).Log("ensureService", err)
		return
	}

	// Block until the new service is responding, limited to 30 seconds
	quickClient := &http.Client{
		Timeout: time.Millisecond * 200,
	}

	start := time.Now()
	for {
		response, err := quickClient.Get(fmt.Sprintf("http://%s.%s.svc.cluster.local:8800/healthz", namespace.Name, service.Name))
		if err == nil && response.StatusCode == http.StatusOK {
			debug.Log("init health", response.StatusCode)
			c.Status(http.StatusCreated)
			return
		}
		if time.Now().Sub(start) > time.Duration(time.Second*60) {
			level.Error(s.Logger).Log("timeout creating init worker", shipInit.ID)
			c.AbortWithStatus(http.StatusGatewayTimeout)
			return
		}

		time.Sleep(time.Millisecond * 100)
	}
}
