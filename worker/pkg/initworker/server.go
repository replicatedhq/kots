package initworker

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

type InitServer struct {
	Logger *zap.SugaredLogger
	Viper  *viper.Viper
	Store  store.Store
	Worker *Worker
}

type CreateSessionRequest struct {
	ID          string `json:"id"`
	UpstreamURI string `json:"upstreamUri"`
	ForkURI     string `json:"forkUri"`
}

func (s *InitServer) Serve(ctx context.Context, address string) error {
	g := gin.New()

	s.configureRoutes(g)

	server := http.Server{Addr: address, Handler: g}
	errChan := make(chan error)

	go func() {
		s.Logger.Infow("starting initserver", zap.String("address", address))
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
	var createUnforkRequest CreateSessionRequest
	if err := c.BindJSON(&createUnforkRequest); err != nil {
		s.Logger.Warnw("initserver failed to read json request", zap.Error(err))
		return
	}

	shipUnfork, err := s.Store.GetUnfork(context.TODO(), createUnforkRequest.ID)
	if err != nil {
		s.Logger.Errorw("initserver failed to get init object", zap.Error(err))
		return
	}

	uploadURL, err := s.Store.GetS3StoreURL(shipUnfork)
	if err != nil {
		s.Logger.Errorw("initserver failed to get uploadURL", zap.Error(err))
		return
	}
	shipUnfork.UploadURL = uploadURL

	err = s.Store.SetOutputFilepath(context.TODO(), shipUnfork)
	if err != nil {
		s.Logger.Errorw("initserver failed to set unfork filepath", zap.Error(err))
		return
	}

	namespace := ship.GetNamespace(context.TODO(), shipUnfork)
	if err := s.Worker.ensureNamespace(context.TODO(), namespace); err != nil {
		s.Logger.Errorw("initserver failed to create namespace", zap.Error(err))
		return
	}

	networkPolicy := ship.GetNetworkPolicySpec(context.TODO(), shipUnfork)
	if err := s.Worker.ensureNetworkPolicy(context.TODO(), networkPolicy); err != nil {
		s.Logger.Errorw("initserver failed to create network policy", zap.Error(err))
		return
	}

	shipState, err := ship.NewStateManager(s.Worker.Config)
	if err != nil {
		s.Logger.Errorw("initserver failed to initialize state manager", zap.Error(err))
		return
	}
	s3State, err := shipState.CreateS3State([]byte(""))
	if err != nil {
		s.Logger.Errorw("initserver failed to upload state to S3", zap.Error(err))
		return
	}

	serviceAccount := ship.GetServiceAccountSpec(context.TODO(), shipUnfork)
	if err := s.Worker.ensureServiceAccount(context.TODO(), serviceAccount); err != nil {
		s.Logger.Errorw("initserver failed to create secret", zap.Error(err))
		return
	}

	role := ship.GetRoleSpec(context.TODO(), shipUnfork)
	if err := s.Worker.ensureRole(context.TODO(), role); err != nil {
		s.Logger.Errorw("initserver failed to create role", zap.Error(err))
		return
	}

	rolebinding := ship.GetRoleBindingSpec(context.TODO(), shipUnfork)
	if err := s.Worker.ensureRoleBinding(context.TODO(), rolebinding); err != nil {
		s.Logger.Errorw("initserver failed to create rolebinding", zap.Error(err))
		return
	}

	pod := ship.GetPodSpec(context.TODO(), s.Worker.Config.LogLevel, s.Worker.Config.ShipImage, s.Worker.Config.ShipTag, s.Worker.Config.ShipPullPolicy, s3State, serviceAccount.Name, shipUnfork, s.Worker.Config.GithubToken)
	if err := s.Worker.ensurePod(context.TODO(), pod); err != nil {
		s.Logger.Errorw("initserver failed to create pod", zap.Error(err))
		return
	}

	// Wait for the pod to be ready here, or clean up and return an error

	service := ship.GetServiceSpec(context.TODO(), shipUnfork)
	if err := s.Worker.ensureService(context.TODO(), service); err != nil {
		s.Logger.Errorw("initserver failed to create service", zap.Error(err))
		return
	}

	// Unfork runs in headless mode, so there's no port in ship we can use
	// to monitor success.

	// We are relying on informers to alert when this has completed
}

func (s *InitServer) CreateInitHandler(c *gin.Context) {
	var createInitRequest CreateSessionRequest
	if err := c.BindJSON(&createInitRequest); err != nil {
		s.Logger.Warnw("initserver failed to read json request", zap.Error(err))
		return
	}

	shipInit, err := s.Store.GetInit(context.TODO(), createInitRequest.ID)
	if err != nil {
		s.Logger.Errorw("initserver failed to get init", zap.Error(err))
		return
	}

	uploadURL, err := s.Store.GetS3StoreURL(shipInit)
	if err != nil {
		s.Logger.Errorw("initserver failed to get upload url", zap.Error(err))
		return
	}
	shipInit.UploadURL = uploadURL

	err = s.Store.SetOutputFilepath(context.TODO(), shipInit)
	if err != nil {
		s.Logger.Errorw("initserver failed to set output filepath", zap.Error(err))
		return
	}

	namespace := ship.GetNamespace(context.TODO(), shipInit)
	if err := s.Worker.ensureNamespace(context.TODO(), namespace); err != nil {
		s.Logger.Errorw("initserver failed to get create namespace", zap.Error(err))
		return
	}

	networkPolicy := ship.GetNetworkPolicySpec(context.TODO(), shipInit)
	if err := s.Worker.ensureNetworkPolicy(context.TODO(), networkPolicy); err != nil {
		s.Logger.Errorw("initserver failed to get create network policy", zap.Error(err))
		return
	}

	shipState, err := ship.NewStateManager(s.Worker.Config)
	if err != nil {
		s.Logger.Errorw("initserver failed to initialize state manager", zap.Error(err))
		return
	}
	s3State, err := shipState.CreateS3State([]byte("{}"))
	if err != nil {
		s.Logger.Errorw("initserver failed to upload state to S3", zap.Error(err))
		return
	}

	serviceAccount := ship.GetServiceAccountSpec(context.TODO(), shipInit)
	if err := s.Worker.ensureServiceAccount(context.TODO(), serviceAccount); err != nil {
		s.Logger.Errorw("initserver failed to get create serviceaccount", zap.Error(err))
		return
	}

	role := ship.GetRoleSpec(context.TODO(), shipInit)
	if err := s.Worker.ensureRole(context.TODO(), role); err != nil {
		s.Logger.Errorw("initserver failed to get create role", zap.Error(err))
		return
	}

	rolebinding := ship.GetRoleBindingSpec(context.TODO(), shipInit)
	if err := s.Worker.ensureRoleBinding(context.TODO(), rolebinding); err != nil {
		s.Logger.Errorw("initserver failed to get create rolebinding", zap.Error(err))
		return
	}

	pod := ship.GetPodSpec(context.TODO(), s.Worker.Config.LogLevel, s.Worker.Config.ShipImage, s.Worker.Config.ShipTag, s.Worker.Config.ShipPullPolicy, s3State, serviceAccount.Name, shipInit, s.Worker.Config.GithubToken)
	if err := s.Worker.ensurePod(context.TODO(), pod); err != nil {
		s.Logger.Errorw("initserver failed to get create pod", zap.Error(err))
		return
	}

	// Wait for the pod to be ready here, or clean up and return an error

	service := ship.GetServiceSpec(context.TODO(), shipInit)
	if err := s.Worker.ensureService(context.TODO(), service); err != nil {
		s.Logger.Errorw("initserver failed to get create service", zap.Error(err))
		return
	}

	// Block until the new service is responding, limited to (math) seconds
	quickClient := &http.Client{
		Timeout: time.Millisecond * 200,
	}

	start := time.Now()
	for {
		response, err := quickClient.Get(fmt.Sprintf("http://%s.%s.svc.cluster.local:8800/healthz", namespace.Name, service.Name))
		if err == nil && response.StatusCode == http.StatusOK {
			c.Status(http.StatusCreated)
			return
		}
		if time.Now().Sub(start) > time.Duration(time.Second*60) {
			s.Logger.Errorw("editserver timeout creating init worker", zap.Error(err))
			c.AbortWithStatus(http.StatusGatewayTimeout)
			return
		}

		time.Sleep(time.Millisecond * 100)
	}
}
