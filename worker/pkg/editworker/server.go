package editworker

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/replicatedhq/kotsadm/worker/pkg/ship"
	"github.com/replicatedhq/kotsadm/worker/pkg/store"
	"github.com/replicatedhq/kotsadm/worker/pkg/version"
	"github.com/spf13/viper"
)

type EditServer struct {
	Logger *zap.SugaredLogger
	Viper  *viper.Viper
	Store  store.Store
	Worker *Worker
}

type CreateEditRequest struct {
	ID      string `json:"id"`
	WatchID string `json:"watchId"`
}

func (s *EditServer) Serve(ctx context.Context, address string) error {
	g := gin.New()

	s.configureRoutes(g)

	server := http.Server{Addr: address, Handler: g}
	errChan := make(chan error)

	go func() {
		s.Logger.Infow("starting editserver", zap.String("address", address))
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
	var createEditRequest CreateEditRequest
	if err := c.BindJSON(&createEditRequest); err != nil {
		s.Logger.Warnw("editserver failed to read json request", zap.Error(err))
		return
	}

	shipEdit, err := s.Store.GetEdit(context.TODO(), createEditRequest.ID)
	if err != nil {
		s.Logger.Errorw("editserver failed to get edit object", zap.Error(err))
		return
	}

	uploadURL, err := s.Store.GetS3StoreURL(shipEdit)
	if err != nil {
		s.Logger.Errorw("editserver failed to get uploadURL", zap.Error(err))
		return
	}
	shipEdit.UploadURL = uploadURL

	err = s.Store.SetOutputFilepath(context.TODO(), shipEdit)
	if err != nil {
		s.Logger.Errorw("editserver failed to setOutputFilepath", zap.Error(err))
		return
	}

	namespace := ship.GetNamespace(context.TODO(), shipEdit)
	if err := s.Worker.ensureNamespace(context.TODO(), namespace); err != nil {
		s.Logger.Errorw("editserver failed to create namespace", zap.Error(err))
		return
	}

	networkPolicy := ship.GetNetworkPolicySpec(context.TODO(), shipEdit)
	if err := s.Worker.ensureNetworkPolicy(context.TODO(), networkPolicy); err != nil {
		s.Logger.Errorw("editserver failed to create network policy", zap.Error(err))
		return
	}

	shipState, err := ship.NewStateManager(s.Worker.Config)
	if err != nil {
		s.Logger.Errorw("editserver failed to initialize state manager", zap.Error(err))
		return
	}
	s3State, err := shipState.CreateS3State(shipEdit.StateJSON)
	if err != nil {
		s.Logger.Errorw("editserver failed to upload state to S3", zap.Error(err))
		return
	}

	serviceAccount := ship.GetServiceAccountSpec(context.TODO(), shipEdit)
	if err := s.Worker.ensureServiceAccount(context.TODO(), serviceAccount); err != nil {
		s.Logger.Errorw("editserver failed to create serviceaccount", zap.Error(err))
		return
	}

	role := ship.GetRoleSpec(context.TODO(), shipEdit)
	if err := s.Worker.ensureRole(context.TODO(), role); err != nil {
		s.Logger.Errorw("editserver failed to create role", zap.Error(err))
		return
	}

	rolebinding := ship.GetRoleBindingSpec(context.TODO(), shipEdit)
	if err := s.Worker.ensureRoleBinding(context.TODO(), rolebinding); err != nil {
		s.Logger.Errorw("editserver failed to create rolebinding", zap.Error(err))
		return
	}

	pod := ship.GetPodSpec(context.TODO(), s.Worker.Config.LogLevel, s.Worker.Config.ShipImage, s.Worker.Config.ShipTag, s.Worker.Config.ShipPullPolicy, s3State, serviceAccount.Name, shipEdit, s.Worker.Config.GithubToken)
	if err := s.Worker.ensurePod(context.TODO(), pod); err != nil {
		s.Logger.Errorw("editserver failed to create pod", zap.Error(err))
		return
	}

	// Wait for the pod to be ready here, or clean up and return an error

	service := ship.GetServiceSpec(context.TODO(), shipEdit)
	if err := s.Worker.ensureService(context.TODO(), service); err != nil {
		s.Logger.Errorw("editserver failed to create service", zap.Error(err))
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
		if err == nil && response.StatusCode == http.StatusOK {
			c.Status(http.StatusCreated)
			return
		}
		if time.Now().Sub(start) > time.Duration(time.Minute*30) {
			s.Logger.Errorw("editserver timeout creating edit worker", zap.Error(err))
			c.AbortWithStatus(http.StatusGatewayTimeout)
			return
		}

		time.Sleep(time.Millisecond * 100)
	}
}
