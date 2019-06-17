package initworker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gosimple/slug"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/pullrequest"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
	"github.com/replicatedhq/ship/pkg/state"
	shipstate "github.com/replicatedhq/ship/pkg/state"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

func (w *Worker) runInformer(ctx context.Context) error {
	debug := level.Debug(log.With(w.Logger, "method", "initworker.Worker.runInformer"))

	debug.Log("event", "runInformer")

	restClient := w.K8sClient.CoreV1().RESTClient()
	watchlist := cache.NewListWatchFromClient(restClient, "pods", "", fields.Everything())

	resyncPeriod := 30 * time.Second

	_, controller := cache.NewInformer(watchlist, &corev1.Pod{}, resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				err := w.deleteFunc(obj)
				if err != nil {
					level.Error(w.Logger).Log("event", "init.session.informer.pod.delete", "err", err)
				}
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				err := w.updateFunc(oldObj, newObj)
				if err != nil {
					level.Error(w.Logger).Log("event", "init.session.informer.pod.update", "err", err)
				}
			},
		},
	)

	controller.Run(ctx.Done())
	return ctx.Err()
}

func (w *Worker) deleteFunc(obj interface{}) error {
	// debug := level.Debug(log.With(w.Logger, "method", "initworker.Worker.deleteFunc"))

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("unexpected type %T", obj)
	}

	// debug.Log("event", "deleteFunc", "pod", pod.Name)

	shipCloudRole, ok := pod.ObjectMeta.Labels["shipcloud-role"]
	if !ok || shipCloudRole != "init" {
		// debug.Log("has role", ok, "ignoring role", shipCloudRole)
		return nil
	}

	return nil
}

func (w *Worker) updateFunc(oldObj interface{}, newObj interface{}) error {
	// debug := level.Debug(log.With(w.Logger, "method", "initworker.Worker.updateFunc"))

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
		level.Error(w.Logger).Log("event", "no ship init/unfork id label")
		return nil
	}

	if oldPod.Status.Phase != newPod.Status.Phase {
		if newPod.Status.Phase == corev1.PodFailed {
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
			if shipCloudRole == "init" {
				return w.initSessionToWatch(id, newPod)
			} else if shipCloudRole == "unfork" {
				return w.unforkSessionToWatch(id, newPod)
			}
		}
	}

	return nil
}

func (w *Worker) unforkSessionToWatch(id string, newPod *corev1.Pod) error {
	unforkSession, err := w.Store.GetUnfork(context.TODO(), id)
	if err != nil {
		return errors.Wrap(err, "get unfork session")
	}

	// read the secret, put the state in the database
	secret, err := w.K8sClient.CoreV1().Secrets(newPod.Namespace).Get(newPod.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "get secret")
	}

	shipState := shipstate.State{}
	if err := json.Unmarshal(secret.Data["state.json"], &shipState); err != nil {
		return errors.Wrap(err, "unmarshal state")
	}

	title := createWatchName(shipState, unforkSession.UpstreamURI)

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

	marshaledMetadata, err := json.Marshal(shipState.V1.Metadata)
	if err != nil {
		return errors.Wrap(err, "marshal state metadata")
	}

	icon := ""
	if shipState.V1 != nil && shipState.V1.Metadata != nil {
		icon = shipState.V1.Metadata.Icon
		if shipState.V1.Metadata.ApplicationType == "replicated.app" {
			if shipState.V1.UpstreamContents != nil {
				if shipState.V1.UpstreamContents.AppRelease != nil {
					icon = shipState.V1.UpstreamContents.AppRelease.ChannelIcon

				}
			}
		}
	}

	if err := w.Store.CreateWatchFromState(context.TODO(), secret.Data["state.json"], marshaledMetadata, title, icon, watchSlug, unforkSession.UserID, unforkSession.ID, "", "", ""); err != nil {
		return errors.Wrap(err, "create watch from state")
	}

	if err := w.Store.SetUnforkStatus(context.TODO(), id, "completed"); err != nil {
		return errors.Wrap(err, "set init status to completed")
	}

	if err := w.K8sClient.CoreV1().Namespaces().Delete(newPod.Namespace, &metav1.DeleteOptions{}); err != nil {
		return errors.Wrap(err, "delete namespace")
	}

	return nil
}

func (w *Worker) initSessionToWatch(id string, newPod *corev1.Pod) error {
	initSession, err := w.Store.GetInit(context.TODO(), id)
	if err != nil {
		return errors.Wrap(err, "get init session")
	}

	// read the secret, put the state in the database
	secret, err := w.K8sClient.CoreV1().Secrets(newPod.Namespace).Get(newPod.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "get secret")
	}

	shipState := shipstate.State{}
	if err := json.Unmarshal(secret.Data["state.json"], &shipState); err != nil {
		return errors.Wrap(err, "unmarshal state")
	}

	// title is the parent's title, if there is a parent
	title := ""
	var parentWatch *types.Watch
	if strings.HasPrefix(initSession.RequestedUpstreamURI, "ship://") {
		parsed, err := url.Parse(initSession.RequestedUpstreamURI)
		if err != nil {
			return errors.Wrap(err, "parse init upstream")
		}

		parentWatchSlug := strings.TrimLeft(parsed.Path, "/")
		parentWatchID, err := w.Store.GetWatchIDFromSlug(context.TODO(), parentWatchSlug, initSession.UserID)
		if err != nil {
			return errors.Wrap(err, "get parent watch id from slug")
		}

		parentWatch, err = w.Store.GetWatch(context.TODO(), parentWatchID)
		if err != nil {
			return errors.Wrap(err, "get parent watch from id")
		}

		title = parentWatch.Title
	} else {
		title = createWatchName(shipState, initSession.UpstreamURI)
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

	marshaledMetadata, err := json.Marshal(shipState.V1.Metadata)
	if err != nil {
		return errors.Wrap(err, "marshal state metadata")
	}

	icon := ""
	if shipState.V1 != nil && shipState.V1.Metadata != nil {
		icon = shipState.V1.Metadata.Icon
		if shipState.V1.Metadata.ApplicationType == "replicated.app" {
			if shipState.V1.UpstreamContents != nil {
				if shipState.V1.UpstreamContents.AppRelease != nil {
					icon = shipState.V1.UpstreamContents.AppRelease.ChannelIcon

				}
			}
		}
	}

	parentWatchID := ""
	if parentWatch != nil {
		parentWatchID = parentWatch.ID
	}
	if err := w.Store.CreateWatchFromState(context.TODO(), secret.Data["state.json"], marshaledMetadata, title, icon, watchSlug, initSession.UserID, initSession.ID, initSession.ClusterID, initSession.GitHubPath, parentWatchID); err != nil {
		return errors.Wrap(err, "create watch from state")
	}

	prNumber, versionStatus, branchName, err := w.maybeCreatePullRequest(initSession.ID, initSession.ClusterID)
	if err != nil {
		return errors.Wrap(err, "maybe create pull request")
	}

	versionLabel := "Unknown"
	if shipState.V1 != nil && shipState.V1.Metadata != nil && shipState.V1.Metadata.Version != "" {
		versionLabel = shipState.V1.Metadata.Version
	} else if parentWatch != nil {
		parentWatchVersion, err := w.Store.GetMostRecentWatchVersion(context.TODO(), parentWatch.ID)
		if err != nil {
			return errors.Wrap(err, "get parent watch version")
		}
		versionLabel = parentWatchVersion.VersionLabel
	}

	setActive := prNumber == 0 // This isn't obvious and a pretty odd implementation. only set non-gitops clusters to active
	if err := w.Store.CreateWatchVersion(context.TODO(), initSession.ID, versionLabel, versionStatus, branchName, 0, prNumber, setActive); err != nil {
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

func (w *Worker) maybeCreatePullRequest(watchID string, clusterID string) (int, string, string, error) {
	// If there isn't a cluster, just mark it as deplyed. This is commonly seen
	// in "midstream" watches
	if clusterID == "" {
		return 0, "deployed", "", nil
	}

	cluster, err := w.Store.GetCluster(context.TODO(), clusterID)
	if err != nil {
		return 0, "", "", err
	}

	// For watches that have a ship cluster, mark it as "pending", the user will
	// take an action to make it deployed
	if cluster.Type != "gitops" {
		return 0, "pending", "", nil
	}

	watch, err := w.Store.GetWatch(context.TODO(), watchID)
	if err != nil {
		return 0, "", "", err
	}
	watchState := state.State{}
	if err := json.Unmarshal([]byte(watch.StateJSON), &watchState); err != nil {
		return 0, "", "", errors.Wrap(err, "unmarshal watch state")
	}

	// Get the file from s3
	s3Filepath := fmt.Sprintf("%s/%d.tar.gz", watchID, 0)
	filename, err := w.Store.DownloadFromS3(context.TODO(), s3Filepath)
	if err != nil {
		return 0, "", "", err
	}
	file, err := os.Open(filename)
	if err != nil {
		return 0, "", "", err
	}

	// And now the real part of this function, for gitops clusters, make a PR and return that status
	// this is a init, so there's no previous PR
	firstPRTitle := fmt.Sprintf("Add %s from Replicated Ship Cloud", watch.Title)
	if watchState.V1 != nil && watchState.V1.Metadata != nil && watchState.V1.Metadata.Version != "" {
		newVersionString := watchState.V1.Metadata.Version
		firstPRTitle = fmt.Sprintf("Add %s version %s from Replicated Ship Cloud", watch.Title, newVersionString)
	}

	githubPath, err := w.Store.GetGitHubPathForClusterWatch(context.TODO(), clusterID, watchID)
	if err != nil {
		return 0, "", "", err
	}

	prRequest, err := pullrequest.NewPullRequestRequest(watch, file, cluster.GitHubOwner, cluster.GitHubRepo, cluster.GitHubBranch, githubPath, cluster.GitHubInstallationID, watchState, firstPRTitle, "")
	if err != nil {
		return 0, "", "", errors.Wrap(err, "create pull request request")
	}

	prNumber, branchName, err := pullrequest.CreatePullRequest(w.Logger, w.Config.GithubPrivateKey, w.Config.GithubIntegrationID, prRequest)
	if err != nil {
		return 0, "", "", errors.Wrap(err, "create pull request")
	}

	return prNumber, "pending", branchName, nil
}

func createWatchName(shipState shipstate.State, uri string) string {
	if shipState.V1.Metadata.ApplicationType == "replicated.app" {
		return shipState.UpstreamContents().AppRelease.ChannelName
	}

	if shipState.V1.Metadata != nil {
		if shipState.V1.Metadata.Name != "" {
			return shipState.V1.Metadata.Name
		} else {
			return shipState.V1.Metadata.AppSlug
		}
	}

	repoRegex := regexp.MustCompile(`github(?:usercontent)?\.com\/([\w-]+)\/([\w-]+)(?:(?:/tree|/blob)?\/([\w-\._]+))?`)
	// attempt to extract a more human-friendly name than the uri
	if repoRegex.MatchString(uri) {
		matches := repoRegex.FindStringSubmatch(uri)

		if len(matches) >= 3 {
			var repoName, version string
			owner := matches[1]
			repo := matches[2]

			if strings.HasPrefix(repo, owner) {
				repoName = repo
			} else {
				repoName = owner + "/" + repo
			}

			if len(matches) >= 4 {
				version = matches[3]
			}

			if version != "" {
				return repoName + "@" + version
			}

			return repoName
		}
	}

	urlRegex := regexp.MustCompile(`(?:https?://)([\w\.\/\-_]+)`)
	if urlRegex.MatchString(uri) {
		matches := urlRegex.FindStringSubmatch(uri)
		if len(matches) >= 2 {
			return matches[1]
		}
	}

	return uri
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
