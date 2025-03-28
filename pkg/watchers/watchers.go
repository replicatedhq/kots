package watchers

import (
	"context"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/reporting"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/ptr"
)

func Start(clusterID string) error {
	cfg, err := k8sutil.GetClusterConfig()
	if err != nil {
		return errors.Wrap(err, "get cluster config")
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return errors.Wrap(err, "create clientset")
	}

	if err := watchECNodes(clientset, clusterID); err != nil {
		return errors.Wrap(err, "watch embedded cluster nodes")
	}

	return nil
}

func watchECNodes(clientset kubernetes.Interface, clusterID string) error {
	if !util.IsEmbeddedCluster() {
		return nil
	}

	logger.Info("starting embedded cluster nodes watcher")

	factory := informers.NewSharedInformerFactory(clientset, 0)
	nodeInformer := factory.Core().V1().Nodes().Informer()

	// by default, add func gets called for existing nodes, and
	// we don't want to report N times every time kotsadm restarts,
	// specially if there are a lot of nodes.
	hasSynced := ptr.To(false)

	handler, err := nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			if !*hasSynced {
				return
			}
			node := obj.(*corev1.Node)
			logger.Infof("Node added: %s", node.Name)
			if err := submitAppInfo(clusterID); err != nil {
				logger.Warnf("failed to submit app info: %v", err)
			}
		},
		DeleteFunc: func(obj interface{}) {
			node := obj.(*corev1.Node)
			logger.Infof("Node deleted: %s", node.Name)
			if err := submitAppInfo(clusterID); err != nil {
				logger.Warnf("failed to submit app info: %v", err)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldNode := oldObj.(*corev1.Node)
			newNode := newObj.(*corev1.Node)

			// Check if ready condition changed
			oldReady := getNodeReadyStatus(oldNode)
			newReady := getNodeReadyStatus(newNode)

			if oldReady != newReady {
				logger.Infof("Node %s ready status changed from %v to %v", newNode.Name, oldReady, newReady)
				if err := submitAppInfo(clusterID); err != nil {
					logger.Warnf("failed to submit app info: %v", err)
				}
			}
		},
	})
	if err != nil {
		return errors.Wrap(err, "add event handler")
	}

	ctx := context.Background()
	go nodeInformer.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), handler.HasSynced) {
		return errors.New("sync node cache")
	}

	*hasSynced = true

	return nil
}

func getNodeReadyStatus(node *corev1.Node) corev1.ConditionStatus {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status
		}
	}
	return corev1.ConditionUnknown
}

func submitAppInfo(clusterID string) error {
	apps, err := store.GetStore().ListAppsForDownstream(clusterID)
	if err != nil {
		return errors.Wrap(err, "list installed apps for downstream")
	}

	if len(apps) == 0 {
		return nil
	}

	// embedded cluster only supports one app
	return reporting.GetReporter().SubmitAppInfo(apps[0].ID)
}
