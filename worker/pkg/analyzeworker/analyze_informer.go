package analyzeworker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

func (w *Worker) runInformer(ctx context.Context) error {
	w.Logger.Infow("starting analyze informer")

	restClient := w.K8sClient.CoreV1().RESTClient()
	watchlist := cache.NewListWatchFromClient(restClient, "pods", "", fields.Everything())

	resyncPeriod := 30 * time.Second

	_, controller := cache.NewInformer(watchlist, &corev1.Pod{}, resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				err := w.updateFunc(oldObj, newObj)
				if err != nil {
					w.Logger.Errorw("error in analyzeworker informer updateFunc", zap.Error(err))
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
	if !ok || shipCloudRole != "analyze" {
		return nil
	}

	supportBundleID, ok := newPod.ObjectMeta.Labels["supportbundle-id"]
	if !ok {
		w.Logger.Errorw("analyzeworker informer, no id found in pod labels", zap.String("pod.name", newPod.Name))
		return nil
	}

	if oldPod.Status.Phase == newPod.Status.Phase {
		return nil
	}

	if newPod.Status.Phase == corev1.PodFailed {
		if err := w.Store.SetUpdateStatus(context.TODO(), supportBundleID, "failed"); err != nil {
			return errors.Wrap(err, "set update status to failed")
		}

		// Leaving these sitting around for now...  we should
		// be grabbing the logs from these and writing them to
		// somewhere for analysis of failures

		// if err := w.K8sClient.CoreV1().Namespaces().Delete(newPod.Namespace, &metav1.DeleteOptions{}); err != nil {
		// 	return errors.Wrap(err, "delete namespace")
		// }

	} else if newPod.Status.Phase == corev1.PodSucceeded {
		podLogOpts := corev1.PodLogOptions{}

		req := w.K8sClient.CoreV1().Pods(newPod.Namespace).GetLogs(newPod.Name, &podLogOpts)
		podLogs, err := req.Stream()
		if err != nil {
			return errors.Wrap(err, "open pod stream")
		}
		defer podLogs.Close()

		buf := new(bytes.Buffer)
		_, err = io.Copy(buf, podLogs)
		if err != nil {
			return errors.Wrap(err, "copy logs too buffer")
		}

		analysisResult := buf.String()
		if err := w.Store.SetAnalysisSucceeded(context.TODO(), supportBundleID, analysisResult); err != nil {
			return errors.Wrap(err, "save analysis results")
		}

		if err := w.K8sClient.CoreV1().Namespaces().Delete(newPod.Namespace, &metav1.DeleteOptions{}); err != nil {
			return errors.Wrap(err, "delete namespace")
		}
	}

	return nil
}
