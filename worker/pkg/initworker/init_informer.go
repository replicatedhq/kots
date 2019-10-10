package initworker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"github.com/replicatedhq/kotsadm/worker/pkg/pullrequest"
	"github.com/replicatedhq/kotsadm/worker/pkg/ship"
	"github.com/replicatedhq/kotsadm/worker/pkg/types"
	shipstate "github.com/replicatedhq/ship/pkg/state"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

func (w *Worker) runInformer(ctx context.Context) error {
	restClient := w.K8sClient.CoreV1().RESTClient()
	watchlist := cache.NewListWatchFromClient(restClient, "pods", "", fields.Everything())

	resyncPeriod := 30 * time.Second

	_, controller := cache.NewInformer(watchlist, &corev1.Pod{}, resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				err := w.updateFunc(oldObj, newObj)
				if err != nil {
					w.Logger.Errorw("error in initworker informer update", zap.Error(err))
				}
			},
		},
	)

	controller.Run(ctx.Done())
	return ctx.Err()
}

func (w *Worker) updateFunc(oldObj interface{}, newObj interface{}) error {
	oldPod, ok := oldObj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("unexpected type %T", oldObj)
	}

	newPod, ok := newObj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("unexpected type %T", newObj)
	}

	shipCloudRole, ok := newPod.ObjectMeta.Labels["shipcloud-role"]
	if !ok || (shipCloudRole != "init" && shipCloudRole != "unfork") {
		return nil
	}

	id := ""

	initID, ok := newPod.ObjectMeta.Labels["ship-init"]
	if ok {
		id = initID
	} else {
		unforkID, ok := newPod.ObjectMeta.Labels["ship-unfork"]
		if ok {
			id = unforkID
		}
	}

	if id == "" {
		w.Logger.Errorw("initworker informer, no ship init/unfork id label", zap.String("pod.name", newPod.Name))
		return nil
	}

	if oldPod.Status.Phase == newPod.Status.Phase {
		return nil
	}

	shipState, err := ship.NewStateManager(w.Config)
	if err != nil {
		return errors.Wrap(err, "initialize state manager")
	}
	stateID := newPod.ObjectMeta.Labels["state-id"]
	deleteState := func() {
		if stateID == "" {
			return
		}
		if err := shipState.DeleteState(stateID); err != nil {
			w.Logger.Errorw("failed to delete state from S3", zap.String("state-id", stateID), zap.Error(err))
		}
	}

	if newPod.Status.Phase == corev1.PodFailed {
		defer deleteState()

		if shipCloudRole == "init" {
			if err := w.Store.SetInitStatus(context.TODO(), id, "failed"); err != nil {
				return errors.Wrap(err, "set init status to failed")
			}
		} else if shipCloudRole == "unfork" {
			if err := w.Store.SetUnforkStatus(context.TODO(), id, "failed"); err != nil {
				return errors.Wrap(err, "set unfork status to failed")
			}
		}

		// Leaving these sitting around for now...  we should
		// be grabbing the logs from these and writing them to
		// somewhere for analysis of failures

		// if err := w.K8sClient.CoreV1().Namespaces().Delete(newPod.Namespace, &metav1.DeleteOptions{}); err != nil {
		// 	return errors.Wrap(err, "delete namespace")
		// }

	} else if newPod.Status.Phase == corev1.PodSucceeded {
		defer deleteState()

		stateJSON, err := shipState.GetState(stateID)
		if err != nil {
			return errors.Wrap(err, "get state")
		}

		if shipCloudRole == "init" {
			if err := w.initSessionToWatch(id, newPod, stateJSON); err != nil {
				return errors.Wrap(err, "init session to watch")
			}
		} else if shipCloudRole == "unfork" {
			return w.unforkSessionToWatch(id, newPod, stateJSON)
		}
	}

	return nil
}

func (w *Worker) unforkSessionToWatch(id string, newPod *corev1.Pod, stateJSON []byte) error {
	unforkSession, err := w.Store.GetUnfork(context.TODO(), id)
	if err != nil {
		return errors.Wrap(err, "get unfork session")
	}

	title := ship.WatchNameFromState(stateJSON)
	watchSlug := fmt.Sprintf("%s/%s", unforkSession.Username, slug.Make(title))

	watches, err := w.Store.GetWatches(context.TODO(), unforkSession.UserID)
	if err != nil {
		return errors.Wrap(err, "get user watches")
	}

	var matchingWatchSlugs []string
	for _, watch := range watches {
		if strings.Contains(watch.Slug, watchSlug) {
			matchingWatchSlugs = append(matchingWatchSlugs, watch.Slug)
		}
	}

	if len(matchingWatchSlugs) > 0 {
		watchSlug = fmt.Sprintf("%s-%d", watchSlug, len(matchingWatchSlugs))
	}

	icon := ship.WatchIconFromState(stateJSON)

	if err := w.Store.CreateWatchFromState(context.TODO(), stateJSON, ship.ShipClusterMetadataFromState(stateJSON), title, icon, watchSlug, unforkSession.UserID, unforkSession.ID, "", "", ""); err != nil {
		return errors.Wrap(err, "create watch from state")
	}

	license := ship.LicenseJsonFromStateJson(stateJSON)
	if err := w.Store.SetWatchLicense(context.TODO(), unforkSession.ID, license); err != nil {
		return errors.Wrap(err, "set watch license")
	}

	if err := w.Store.SetUnforkStatus(context.TODO(), id, "completed"); err != nil {
		return errors.Wrap(err, "set init status to completed")
	}

	if err := w.K8sClient.CoreV1().Namespaces().Delete(newPod.Namespace, &metav1.DeleteOptions{}); err != nil {
		return errors.Wrap(err, "delete namespace")
	}

	return nil
}

func (w *Worker) initSessionToWatch(id string, newPod *corev1.Pod, stateJSON []byte) error {
	parentWatchID, hasParent := newPod.ObjectMeta.Labels["parent-watch-id"]
	parentSequence, _ := newPod.ObjectMeta.Labels["parent-sequence"]

	initSession, err := w.Store.GetInit(context.TODO(), id)
	if err != nil {
		return errors.Wrap(err, "get init session")
	}

	shipState := shipstate.State{}
	if err := json.Unmarshal(stateJSON, &shipState); err != nil {
		return errors.Wrap(err, "unmarshal state")
	}

	// title is the parent's title, if there is a parent
	title := ""
	var parentWatch *types.Watch
	if hasParent {
		parentWatch, err = w.Store.GetWatch(context.TODO(), parentWatchID)
		if err != nil {
			return errors.Wrap(err, "get parent watch")
		}

		title = parentWatch.Title
	} else {
		title = ship.WatchNameFromState(stateJSON)
	}

	// Slug
	watches, err := w.Store.GetWatches(context.TODO(), initSession.UserID)
	if err != nil {
		return errors.Wrap(err, "get user watches")
	}

	existingSlugs := make([]string, 0, 0)
	for _, watch := range watches {
		existingSlugs = append(existingSlugs, watch.Slug)
	}

	attemptedSlug := fmt.Sprintf("%s/%s", initSession.Username, slug.Make(title))
	watchSlug := ""
	count := 0
	for watchSlug == "" {
		if !stringInSlice(attemptedSlug, existingSlugs) {
			watchSlug = attemptedSlug
		} else {
			count++
			attemptedSlug = fmt.Sprintf("%s/%s-%d", initSession.Username, slug.Make(title), count)

		}
	}

	icon := ship.WatchIconFromState(stateJSON)

	if err := w.Store.CreateWatchFromState(context.TODO(), stateJSON, ship.ShipClusterMetadataFromState(stateJSON), title, icon, watchSlug, initSession.UserID, initSession.ID, initSession.ClusterID, initSession.GitHubPath, parentWatchID); err != nil {
		return errors.Wrap(err, "create watch from state")
	}

	license := ship.LicenseJsonFromStateJson(stateJSON)
	if err := w.Store.SetWatchLicense(context.TODO(), initSession.ID, license); err != nil {
		return errors.Wrap(err, "set watch license")
	}

	prNumber, commitSHA, versionStatus, branchName, err := w.maybeCreatePullRequest(initSession.ID, initSession.ClusterID)
	if err != nil {
		return errors.Wrap(err, "maybe create pull request")
	}

	versionLabel := ""
	if parentWatch == nil {
		versionLabel = ship.WatchVersionFromState(stateJSON)
	} else {
		seq, err := strconv.Atoi(parentSequence)
		if err != nil {
			return errors.Wrap(err, "convert parent sequence")
		}

		parentWatchVersion, err := w.Store.GetOneWatchVersion(context.TODO(), parentWatchID, seq)
		if err != nil {
			return errors.Wrap(err, "get parent watch version")
		}
		versionLabel = parentWatchVersion.VersionLabel
	}

	setActive := prNumber == 0 // This isn't obvious and a pretty odd implementation. only set non-gitops clusters to active
	var parentSeq *int
	if parentWatch != nil {
		seq, err := strconv.Atoi(parentSequence)
		if err != nil {
			return errors.Wrap(err, "convert parent sequence")
		}

		parentSeq = &seq
	}
	if err := w.Store.CreateWatchVersion(context.TODO(), initSession.ID, versionLabel, versionStatus, branchName, 0, prNumber, commitSHA, setActive, parentSeq); err != nil {
		return errors.Wrap(err, "create watch version")
	}

	if err := w.Store.SetInitStatus(context.TODO(), id, "completed"); err != nil {
		return errors.Wrap(err, "set init status to completed")
	}

	if err := w.K8sClient.CoreV1().Namespaces().Delete(newPod.Namespace, &metav1.DeleteOptions{}); err != nil {
		return errors.Wrap(err, "delete namespace")
	}

	return nil
}

func (w *Worker) maybeCreatePullRequest(watchID string, clusterID string) (int, string, string, string, error) {
	// If there isn't a cluster, just mark it as deployed. This is commonly seen
	// in "midstream" watches
	if clusterID == "" {
		return 0, "", "deployed", "", nil
	}

	cluster, err := w.Store.GetCluster(context.TODO(), clusterID)
	if err != nil {
		return 0, "", "", "", err
	}

	// For watches that have a ship cluster, mark it as "pending", the user will
	// take an action to make it deployed
	if cluster.Type != "gitops" {
		return 0, "", "pending", "", nil
	}

	watch, err := w.Store.GetWatch(context.TODO(), watchID)
	if err != nil {
		return 0, "", "", "", err
	}
	watchState := shipstate.State{}
	if err := json.Unmarshal([]byte(watch.StateJSON), &watchState); err != nil {
		return 0, "", "", "", errors.Wrap(err, "unmarshal watch state")
	}

	s3Filepath := fmt.Sprintf("%s/%d.tar.gz", watchID, 0)
	filename, err := w.Store.DownloadFromS3(context.TODO(), s3Filepath)
	if err != nil {
		return 0, "", "", "", err
	}
	defer os.Remove(filename)

	file, err := os.Open(filename)
	if err != nil {
		return 0, "", "", "", err
	}
	defer file.Close()

	// And now the real part of this function, for gitops clusters, make a PR and return that status
	// this is a init, so there's no previous PR
	firstPRTitle := fmt.Sprintf("Add %s", watch.Title)
	if watchState.V1 != nil && watchState.V1.Metadata != nil && watchState.V1.Metadata.Version != "" {
		newVersionString := watchState.V1.Metadata.Version
		firstPRTitle = fmt.Sprintf("Add %s version %s", watch.Title, newVersionString)
	}

	githubPath, err := w.Store.GetGitHubPathForClusterWatch(context.TODO(), clusterID, watchID)
	if err != nil {
		return 0, "", "", "", err
	}

	prRequest, err := pullrequest.NewPullRequestRequest(w.Store, watch, file, cluster.GitHubOwner, cluster.GitHubRepo, cluster.GitHubBranch, githubPath, cluster.GitHubInstallationID, watchState, firstPRTitle, "")
	if err != nil {
		return 0, "", "", "", errors.Wrap(err, "create pull request request")
	}

	prNumber, commitSHA, branchName, err := pullrequest.CreatePullRequest(w.Logger, w.Config.GithubPrivateKey, w.Config.GithubIntegrationID, prRequest)
	if err != nil {
		return 0, "", "", "", errors.Wrap(err, "create pull request")
	}

	return prNumber, commitSHA, "pending", branchName, nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
