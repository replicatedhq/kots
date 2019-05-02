package watchworker

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// PullRequestHandler is called from operator when there's a pull reuqest action
// set on the spec. This will be called with the standard payload (archive file and state json).
// This handler is responsible for creating the PR and any history items related to the PR,
// but should not create a new sequence or update the state in the database.
func (s *WatchServer) PullRequestHandler(c *gin.Context) {
	debug := level.Debug(log.With(s.Logger, "method", "watchworker.Server.PullRequestHandler"))
	debug.Log("event", "pullrequest", "id", c.Param("notificationId"))




	// sourceBranch := ""
	// if pullrequest.ShouldUsePreviousBranch(*notification, previousPR) {
	// 	sourceBranch = previousPR.SourceBranch
	// }

	// // build the Pull Request from the request
	// prRequest := pullrequest.NewPullRequestRequest(watch, notification, watchState, file, "", sourceBranch)

	// shouldCreate, err := pullrequest.ShouldCreatePullRequest(s.Worker.Logger, s.Worker.Config.GithubPrivateKey, s.Worker.Config.GithubIntegrationID, prRequest)
	// if err != nil {
	// 	level.Error(s.Logger).Log("shouldCreatePR", err)
	// 	c.AbortWithStatus(http.StatusInternalServerError)
	// 	return
	// }

	// if shouldCreate {
	// 	// Create the PR
	// 	pullRequestNumber, sourceBranch, err := pullrequest.CreatePullRequest(s.Worker.Logger, s.Worker.Config.GithubPrivateKey, s.Worker.Config.GithubIntegrationID, prRequest)
	// 	if err != nil {
	// 		level.Error(s.Logger).Log("createPullRequest", err)
	// 		c.AbortWithStatus(http.StatusInternalServerError)
	// 		return
	// 	}

	// 	sequenceNumber, err := s.Store.GetSequenceNumberForNotificationID(context.TODO(), c.Param("notificationId"))
	// 	if err != nil {
	// 		level.Error(s.Logger).Log("getSequenceNumber", err)
	// 		c.AbortWithStatus(http.StatusInternalServerError)
	// 		return
	// 	}

	// 	if err := s.Store.SavePullRequestCreated(context.TODO(), c.Param("notificationId"), prRequest.NewVersionString, pullRequestNumber, *notification, sequenceNumber, sourceBranch); err != nil {
	// 		level.Error(s.Logger).Log("savePullRequestCreated", err)
	// 		c.AbortWithStatus(http.StatusInternalServerError)
	// 		return
	// 	}
	// }

	c.String(http.StatusOK, "")
}
