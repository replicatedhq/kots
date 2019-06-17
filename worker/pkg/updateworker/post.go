package updateworker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"

	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/pullrequest"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
	"github.com/replicatedhq/ship/pkg/state"
)

func (w *Worker) postUpdateActions(watchID string, sequence int, s3Filepath string) error {
	watch, err := w.Store.GetWatch(context.TODO(), watchID)
	if err != nil {
		return errors.Wrap(err, "get watch")
	}

	downstreamWatchIDs, err := w.Store.ListDownstreamWatchIDs(context.TODO(), watchID)
	if err != nil {
		level.Error(w.Logger).Log("event", "get downstream watch ids", "err", err)
	}
	for _, downstreamWatchID := range downstreamWatchIDs {
		if err := w.Store.CreateWatchUpdate(context.TODO(), downstreamWatchID); err != nil {
			level.Error(w.Logger).Log("event", "create downstream watch update", "err", err)
		}
	}
	archive, err := w.fetchArchiveFromS3(watchID, s3Filepath)
	if err != nil {
		return errors.Wrap(err, "fetch archive")
	}
	defer os.Remove(archive.Name())

	if err := w.triggerIntegrations(watch, sequence, archive); err != nil {
		return errors.Wrap(err, "trigger integraitons")
	}

	return nil
}

func (w *Worker) fetchArchiveFromS3(watchID string, s3Filepath string) (*os.File, error) {
	filename, err := w.Store.DownloadFromS3(context.TODO(), s3Filepath)
	if err != nil {
		return nil, errors.Wrap(err, "download")
	}

	archive, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "open")
	}

	return archive, nil
}

func (w *Worker) triggerIntegrations(watch *types.Watch, sequence int, archive *os.File) error {
	cluster, err := w.Store.GetClusterForWatch(context.TODO(), watch.ID)
	if err != nil {
		return errors.Wrap(err, "get cluster for watch")
	}

	prNumber, versionStatus, branchName, err := w.maybeCreatePullRequest(watch, cluster, archive)
	if err != nil {
		return errors.Wrap(err, "maybe create pull request")
	}

	isCurrent := cluster == nil

	if err := w.createVersion(watch, sequence, archive, versionStatus, branchName, prNumber, isCurrent); err != nil {
		return errors.Wrap(err, "create version")
	}

	return nil
}

func (w *Worker) maybeCreatePullRequest(watch *types.Watch, cluster *types.Cluster, file multipart.File) (int, string, string, error) {
	file.Seek(0, io.SeekStart)

	// midstreams won't have a cluster
	if cluster == nil {
		return 0, "deployed", "", nil
	}

	// ship clusters don't need PRs
	if cluster.Type != "gitops" {
		return 0, "pending", "", nil
	}

	watchState := state.State{}
	if err := json.Unmarshal([]byte(watch.StateJSON), &watchState); err != nil {
		return 0, "", "", errors.Wrap(err, "unmarshal watch state")
	}

	previousWatchVersion, err := w.Store.GetMostRecentWatchVersion(context.TODO(), watch.ID)
	if err != nil {
		return 0, "", "", errors.Wrap(err, "getMostRecentPullRequestCreated")
	}

	sourceBranch := ""
	if pullrequest.ShouldUsePreviousBranch(previousWatchVersion) {
		sourceBranch = previousWatchVersion.SourceBranch
	}

	updatePRTitle := fmt.Sprintf("Update %s with edits made in Replicated Ship Cloud", watch.Title)

	githubPath, err := w.Store.GetGitHubPathForClusterWatch(context.TODO(), cluster.ID, watch.ID)
	if err != nil {
		return 0, "", "", err
	}

	prRequest, err := pullrequest.NewPullRequestRequest(watch, file, cluster.GitHubOwner, cluster.GitHubRepo, cluster.GitHubBranch, githubPath, cluster.GitHubInstallationID, watchState, updatePRTitle, sourceBranch)
	if err != nil {
		return 0, "", "", errors.Wrap(err, "new pull request request")
	}

	prNumber, branchName, err := pullrequest.CreatePullRequest(w.Logger, w.Config.GithubPrivateKey, w.Config.GithubIntegrationID, prRequest)
	if err != nil {
		return 0, "", "", errors.Wrap(err, "create pull request")
	}

	return prNumber, "pending", branchName, nil
}

func (w *Worker) createVersion(watch *types.Watch, sequence int, file multipart.File, versionStatus string, branchName string, prNumber int, isCurrent bool) error {
	watchState := state.State{}
	if err := json.Unmarshal([]byte(watch.StateJSON), &watchState); err != nil {
		return errors.Wrap(err, "unmarshal watch state")
	}

	versionLabel := "Unknown"
	if watchState.V1 != nil && watchState.V1.Metadata != nil && watchState.V1.Metadata.Version != "" {
		versionLabel = watchState.V1.Metadata.Version
	} else {
		// Hmmm...
		previousWatchVersion, err := w.Store.GetMostRecentWatchVersion(context.TODO(), watch.ID)
		if err != nil {
			return errors.Wrap(err, "get most recent watch version")
		}

		versionLabel = previousWatchVersion.VersionLabel
	}

	err := w.Store.CreateWatchVersion(context.TODO(), watch.ID, versionLabel, versionStatus, branchName, sequence, prNumber, isCurrent)
	if err != nil {
		return errors.Wrap(err, "create watch version")
	}

	return nil
}
