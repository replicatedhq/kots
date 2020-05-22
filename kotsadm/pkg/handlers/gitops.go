package handlers

import (
	"net/http"
	"os"
	"sort"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/kotsadm/pkg/app"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstream"
	"github.com/replicatedhq/kots/kotsadm/pkg/downstreamversion"
	"github.com/replicatedhq/kots/kotsadm/pkg/gitops"
	"github.com/replicatedhq/kots/kotsadm/pkg/logger"
	"github.com/replicatedhq/kots/kotsadm/pkg/session"
	"github.com/replicatedhq/kots/kotsadm/pkg/version"

	"gopkg.in/src-d/go-git.v4/plumbing/transport"
)

type InitGitOpsConnectionRequest struct {
}

type InitGitOpsConnectionResponse struct {
	Error string `json:"error"`
}

func InitGitOpsConnection(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "content-type, origin, accept, authorization")

	if r.Method == "OPTIONS" {
		w.WriteHeader(200)
		return
	}

	initGitOpsConnectionResponse := &InitGitOpsConnectionResponse{}

	sess, err := session.Parse(r.Header.Get("Authorization"))
	if err != nil {
		logger.Error(err)
		initGitOpsConnectionResponse.Error = "failed to parse authorization header"
		JSON(w, 401, initGitOpsConnectionResponse)
		return
	}

	// we don't currently have roles, all valid tokens are valid sessions
	if sess == nil || sess.ID == "" {
		initGitOpsConnectionResponse.Error = "failed to parse authorization header"
		JSON(w, 401, initGitOpsConnectionResponse)
		return
	}

	appID := mux.Vars(r)["appId"]
	clusterID := mux.Vars(r)["clusterId"]

	gitOpsConfig, err := gitops.GetDownstreamGitOps(appID, clusterID)
	if err != nil {
		logger.Error(err)
		initGitOpsConnectionResponse.Error = "failed to get downstream gitops"
		JSON(w, 500, initGitOpsConnectionResponse)
		return
	}
	if gitOpsConfig == nil {
		initGitOpsConnectionResponse.Error = "gitops config not found"
		JSON(w, 500, initGitOpsConnectionResponse)
		return
	}

	err = gitops.TestGitOpsConnection(gitOpsConfig)
	if errors.Cause(err) == transport.ErrEmptyRemoteRepository {
		_, err = gitops.InitializeGitRepo(gitOpsConfig)
		if err != nil {
			logger.Error(err)
			initGitOpsConnectionResponse.Error = "failed to initialize repo"
			JSON(w, 500, initGitOpsConnectionResponse)
			return
		}
	} else if err != nil {
		// update gitops error in configmap
		if err := gitops.SetGitOpsError(appID, clusterID, err.Error()); err != nil {
			logger.Error(err)
		}
		logger.Error(err)
		initGitOpsConnectionResponse.Error = "failed to test gitops connection"
		JSON(w, 500, initGitOpsConnectionResponse)
		return
	}

	// clear gitops error in configmap
	if err := gitops.SetGitOpsError(appID, clusterID, ""); err != nil {
		logger.Error(err)
		initGitOpsConnectionResponse.Error = "failed to clear gitops error"
		JSON(w, 500, initGitOpsConnectionResponse)
		return
	}

	if err := sendInitialGitCommits(gitOpsConfig, appID, clusterID); err != nil {
		logger.Error(err)
		initGitOpsConnectionResponse.Error = "failed to send initial commits"
		JSON(w, 500, initGitOpsConnectionResponse)
		return
	}

	JSON(w, 204, "")
}

func sendInitialGitCommits(gitOpsConfig *gitops.GitOpsConfig, appID string, clusterID string) error {
	a, err := app.Get(appID)
	if err != nil {
		return errors.Wrap(err, "failed to get app")
	}

	d, err := downstream.Get(a.ID, clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get downstream")
	}

	// create commit for current version
	currentVersion, err := downstreamversion.GetCurrentVersion(a.ID, d.ClusterID)
	if err != nil {
		return errors.Wrap(err, "failed to get current downstream version")
	}
	if currentVersion != nil {
		archiveDir, err := version.GetAppVersionArchive(a.ID, currentVersion.ParentSequence)
		if err != nil {
			return errors.Wrap(err, "failed to get app version dir")
		}
		defer os.RemoveAll(archiveDir)

		_, err = gitops.CreateGitOpsCommit(gitOpsConfig, a.Slug, a.Name, int(currentVersion.Sequence), archiveDir, d.Name)
		if err != nil {
			return errors.Wrap(err, "failed to create gitops commit for current version")
		}
	}

	// create commits for pending versions
	pendingVersions, err := downstreamversion.GetPendingVersions(a.ID, clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to list downstream pending versions")
	}
	sort.Slice(pendingVersions, func(i, j int) bool { return pendingVersions[i].Sequence < pendingVersions[j].Sequence }) // sort ascending

	for _, v := range pendingVersions {
		archiveDir, err := version.GetAppVersionArchive(a.ID, v.ParentSequence)
		if err != nil {
			return errors.Wrap(err, "failed to get app version dir")
		}
		defer os.RemoveAll(archiveDir)

		_, err = gitops.CreateGitOpsCommit(gitOpsConfig, a.Slug, a.Name, int(v.Sequence), archiveDir, d.Name)
		if err != nil {
			return errors.Wrapf(err, "failed to create gitops commit for version %d", v.Sequence)
		}
	}

	return nil
}
