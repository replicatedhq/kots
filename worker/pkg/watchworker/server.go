package watchworker

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/email"
	"github.com/replicatedhq/ship-cluster/worker/pkg/store"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
	"github.com/replicatedhq/ship-cluster/worker/pkg/version"
	"github.com/spf13/viper"
)

type WatchServer struct {
	Logger log.Logger
	Viper  *viper.Viper
	Store  store.Store
	Worker *Worker
	Mailer *email.Mailer
}

type FirstPullRequestRequest struct {
	Org                  string `form:"org"`
	Repo                 string `form:"repo"`
	Branch               string `form:"branch"`
	RootPath             string `form:"rootPath"`
	GithubInstallationID string `form:"githubInstallationID"`
	WatchID              string `form:"watchID"`
	VersionLabel         string `form:"versionLabel"`
	NotificationID       string `form:"existingID"`
}

type StateLocation struct {
	Namespace string
	Name      string
	Key       string
}

func (s *WatchServer) Serve(ctx context.Context, addr string) error {
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

func (s *WatchServer) configureRoutes(g *gin.Engine) {
	root := g.Group("/")

	root.GET("/healthz", s.Healthz)
	root.GET("/metricz", s.Metricz)

	v1 := g.Group("/v1")

	// This method is called by all operators to update the state in shipcloud
	v1.POST("/updated/:watchId", s.UpdatedHandler)

	// User defined actions
	v1.POST("/webhook/:notificationId", s.WebhookRequestHandler)
	v1.POST("/email/:notificationId", s.EmailRequestHandler)
}

// Healthz returns a 200 with the version if provided
func (s *WatchServer) Healthz(c *gin.Context) {
	c.JSON(200, map[string]interface{}{
		"server":    "watch",
		"version":   version.Version(),
		"sha":       version.GitSHA(),
		"buildTime": version.BuildTime(),
	})
}

// Metricz returns (empty) metrics for this server
func (s *WatchServer) Metricz(c *gin.Context) {
	type Metric struct {
		M1  float64 `json:"m1"`
		P95 float64 `json:"p95"`
	}
	c.IndentedJSON(200, map[string]Metric{})
}

func (s *WatchServer) WebhookRequestHandler(c *gin.Context) {
	debug := level.Debug(log.With(s.Logger, "method", "watchworker.Server.WebhookRequestHandler"))
	debug.Log("event", "webhook", "id", c.Param("notificationId"))

	c.String(http.StatusOK, "")
}

func (s *WatchServer) uploadTarGZ(ctx context.Context, watchID string, file multipart.File) (int, error) {
	nextSequence, err := s.Store.GetNextUploadSequence(ctx, watchID)
	if err != nil {
		return 0, errors.Wrap(err, "findNextFileSequenceNum")
	}

	outputSession := types.Notification{
		ID:             watchID,
		WatchID:        watchID,
		UploadSequence: nextSequence,
	}

	err = s.Store.SetOutputFilepath(ctx, &outputSession)
	if err != nil {
		return 0, errors.Wrap(err, "setOutputFilepath")
	}

	err = s.Store.UploadToS3(ctx, &outputSession, file)
	if err != nil {
		return 0, errors.Wrap(err, "uploadToS3")
	}

	return nextSequence, nil
}

func (s *WatchServer) readStateJSONFromRequest(c *gin.Context) ([]byte, error) {
	stateLocationFileHeader, err := c.FormFile("state")
	if err != nil {
		level.Error(s.Logger).Log("read form (state)", err)
		return nil, err
	}
	stateFile, err := stateLocationFileHeader.Open()
	if err != nil {
		level.Error(s.Logger).Log("openFile (state)", err)
		return nil, err
	}
	stateLocationData, err := ioutil.ReadAll(stateFile)
	if err != nil {
		level.Error(s.Logger).Log("readall (state)", err)
		return nil, err
	}
	stateLocation := StateLocation{}
	if err := json.Unmarshal(stateLocationData, &stateLocation); err != nil {
		level.Error(s.Logger).Log("unmarshal state location", err)
		return nil, err
	}
	stateJSON, err := s.Worker.GetStateJSONFromSecret(stateLocation.Namespace, stateLocation.Name, stateLocation.Key)
	if err != nil {
		level.Error(s.Logger).Log("getStateJSONFromSecret", err)
		return nil, err
	}
	if len(stateJSON) == 0 {
		level.Error(s.Logger).Log("stateJSON was empty", err)
		return nil, err
	}

	return stateJSON, nil
}

func (s *WatchServer) getWatchFromNotificationID(notificationID string) (*types.Watch, error) {
	watchID, err := s.Store.GetNotificationWatchID(context.TODO(), notificationID)
	if err != nil {
		level.Error(s.Logger).Log("getNotificationWatchID", err)
		return nil, errors.Wrap(err, "getNotificationWatchID")
	}

	watch, err := s.Store.GetWatch(context.TODO(), watchID)
	if err != nil {
		level.Error(s.Logger).Log("getWatch", err, "id", watchID)
		return nil, errors.Wrap(err, "getWatch")
	}

	return watch, nil
}

func (s *WatchServer) readArchiveFromRequest(c *gin.Context) (multipart.File, error) {
	fileHeader, err := c.FormFile("output")
	if err != nil {
		level.Error(s.Logger).Log("read form", err)
		return nil, errors.Wrap(err, "read form")
	}

	file, err := fileHeader.Open()
	if err != nil {
		level.Error(s.Logger).Log("openFile", err)
		return nil, errors.Wrap(err, "openFile")
	}

	return file, nil
}
