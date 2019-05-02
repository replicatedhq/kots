package watchworker

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/replicatedhq/ship-cluster/worker/pkg/email"
	"github.com/replicatedhq/ship/pkg/state"
)

func (s *WatchServer) EmailRequestHandler(c *gin.Context) {
	debug := level.Debug(log.With(s.Logger, "method", "watchworker.Server.EmailRequestHandler"))
	debug.Log("event", "email", "id", c.Param("notificationId"))

	stateJSON, err := s.readStateJSONFromRequest(c)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	watchState := state.VersionedState{}
	if err := json.Unmarshal([]byte(stateJSON), &watchState); err != nil {
		level.Error(s.Logger).Log("unmarshal state", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// Get the watch and notification
	watch, err := s.getWatchFromNotificationID(c.Param("notificationId"))
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	notification, err := s.Store.GetEmailNotification(context.TODO(), c.Param("notificationId"))
	if err != nil {
		level.Error(s.Logger).Log("getEmailNotification", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// Read the archive from the request
	file, err := s.readArchiveFromRequest(c)
	if err != nil {
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	emailRequest := email.NewEmailRequest(watch, notification, watchState, file, "")

	// TODO fix multiple recipients
	if err := s.Mailer.SendEmail(emailRequest); err != nil {
		level.Error(s.Logger).Log("sendEmail", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.String(http.StatusOK, "")
}
