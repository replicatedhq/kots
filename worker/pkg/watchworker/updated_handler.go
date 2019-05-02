package watchworker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/replicatedhq/ship-cluster/worker/pkg/pullrequest"
	"github.com/replicatedhq/ship/pkg/state"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
)

// UpdatedHandler is called by every watch/update operator as when an update was prepared
// This handler should not process all notifications, but instead is responsible for updateing
// the ship cloud state (for example, the watch row in the database and uploading the archive to s3)
func (s *WatchServer) UpdatedHandler(c *gin.Context) {
	debug := level.Debug(log.With(s.Logger, "method", "watchworker.Server.UpdatedHandler"))

	watchID := c.Param("watchId")
	debug.Log("event", "pullrequest", "watchid", watchID)

	if watchID == "" {
		level.Error(s.Logger).Log("missingWatchId")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// Read the state from the request
	stateJSON, err := s.readStateJSONFromRequest(c)
	if err != nil {
		level.Error(s.Logger).Log("readStateJSONFromRequest", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// Prevent deleting state json if it's posted empty
	if len(stateJSON) == 0 {
		level.Error(s.Logger).Log("stateJSON was empty")
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	// Update the state in the database
	if err := s.Store.UpdateWatchFromState(context.TODO(), watchID, stateJSON); err != nil {
		level.Error(s.Logger).Log("updateWatchFromState", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// Read the file (archive) from the request
	fileHeader, err := c.FormFile("output")
	if err != nil {
		level.Error(s.Logger).Log("read form", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		level.Error(s.Logger).Log("openFile", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// Upload the artifact to s3
	sequence, err := s.uploadTarGZ(context.TODO(), watchID, file)
	if err != nil {
		level.Error(s.Logger).Log("uploadTarGZ", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// get the cluster (returns nil if there's no cluster)
	cluster, err := s.Store.GetClusterForWatch(context.TODO(), watchID)
	if err != nil {
		level.Error(s.Logger).Log("getCluster", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	watch, err := s.Store.GetWatch(context.TODO(), watchID)
	if err != nil {
		level.Error(s.Logger).Log("getWatch", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// "maybe" create a PR (this will happen if there's a gitops cluster)
	prNumber, versionStatus, branchName, err := s.maybeCreatePullRequest(watch, cluster, file)
	if err != nil {
		level.Error(s.Logger).Log("maybeCreatePullRequest", err)
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	// This should be set as the current sequence if there's not a cluster (aka this is a midstream watch)
	isCurrent := cluster == nil

	// Create the new version
	if err := s.createVersion(watch, sequence, file, versionStatus, branchName, prNumber, isCurrent); err != nil {
		level.Error(s.Logger).Log("createVersion", err)
		c.AbortWithStatus(http.StatusInternalServerError)
	}

	c.String(http.StatusOK, "")
}

func (s *WatchServer) maybeCreatePullRequest(watch *types.Watch, cluster *types.Cluster, file multipart.File) (int, string, string, error) {
	file.Seek(0, io.SeekStart)

	// midstreams won't have a cluster
	if cluster == nil {
		return 0, "deployed", "", nil
	}

	// ship clusters don't need PRs
	if cluster.Type != "gitops" {
		return 0, "pending", "", nil
	}

	watchState := state.VersionedState{}
	if err := json.Unmarshal([]byte(watch.StateJSON), &watchState); err != nil {
		return 0, "", "", errors.Wrap(err, "unmarshal watch state")
	}

	previousWatchVersion, err := s.Store.GetMostRecentWatchVersion(context.TODO(), watch.ID)
	if err != nil {
		return 0, "", "", errors.Wrap(err, "getMostRecentPullRequestCreated")
	}

	sourceBranch := ""
	if pullrequest.ShouldUsePreviousBranch(previousWatchVersion) {
		sourceBranch = previousWatchVersion.SourceBranch
	}

	updatePRTitle := fmt.Sprintf("Update %s to a new version from Replicated Ship Cloud", watch.Title)
	if watchState.V1 != nil && watchState.V1.Metadata != nil && watchState.V1.Metadata.Version != "" {
		updatePRTitle = fmt.Sprintf("Update %s to version %s from Replicated Ship Cloud", watch.Title, watchState.V1.Metadata.Version)
	}

	githubPath, err := s.Worker.Store.GetGitHubPathForClusterWatch(context.TODO(), cluster.ID, watch.ID)
	if err != nil {
		return 0, "", "", err
	}

	prRequest, err := pullrequest.NewPullRequestRequest(watch, file, cluster.GitHubOwner, cluster.GitHubRepo, cluster.GitHubBranch, githubPath, cluster.GitHubInstallationID, watchState, updatePRTitle, sourceBranch)
	if err != nil {
		return 0, "", "", errors.Wrap(err, "new pull request request")
	}

	prNumber, branchName, err := pullrequest.CreatePullRequest(s.Logger, s.Worker.Config.GithubPrivateKey, s.Worker.Config.GithubIntegrationID, prRequest)
	if err != nil {
		return 0, "", "", errors.Wrap(err, "create pull request")
	}

	return prNumber, "pending", branchName, nil
}

func (s *WatchServer) createVersion(watch *types.Watch, sequence int, file multipart.File, versionStatus string, branchName string, prNumber int, isCurrent bool) error {
	watchState := state.VersionedState{}
	if err := json.Unmarshal([]byte(watch.StateJSON), &watchState); err != nil {
		return errors.Wrap(err, "unmarshal watch state")
	}

	versionLabel := "Unknown"
	if watchState.V1 != nil && watchState.V1.Metadata != nil && watchState.V1.Metadata.Version != "" {
		versionLabel = watchState.V1.Metadata.Version
	}

	err := s.Store.CreateWatchVersion(context.TODO(), watch.ID, versionLabel, versionStatus, branchName, sequence, prNumber, isCurrent)
	if err != nil {
		return errors.Wrap(err, "create watch version")
	}

	return nil
}
