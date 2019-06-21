package updateworker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

var updateSession types.UpdateSession

func (w *Worker) runInformer(ctx context.Context) error {
	restClient := w.K8sClient.CoreV1().RESTClient()
	watchlist := cache.NewListWatchFromClient(restClient, "pods", "", fields.Everything())

	resyncPeriod := 30 * time.Second

	_, controller := cache.NewInformer(watchlist, &corev1.Pod{}, resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				err := w.deleteFunc(obj)
				if err != nil {
					w.Logger.Errorw("error in updateworker informer deleteFunc", zap.Error(err))
				}
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				err := w.updateFunc(oldObj, newObj)
				if err != nil {
					w.Logger.Errorw("error in updateworker informer updateFunc", zap.Error(err))
				}
			},
		},
	)

	controller.Run(ctx.Done())
	return ctx.Err()
}

func (w *Worker) deleteFunc(obj interface{}) error {
	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("unexpected type %T", obj)
	}

	shipCloudRole, ok := pod.ObjectMeta.Labels["shipcloud-role"]
	if !ok || shipCloudRole != updateSession.GetRole() {
		return nil
	}

	id, ok := pod.ObjectMeta.Labels[updateSession.GetType()]
	if !ok {
		w.Logger.Errorw("updateworker delete informer, no id found in pod labels", zap.String("id", id), zap.String("pod.name", pod.Name))
		return nil
	}

	return nil
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
	if !ok || shipCloudRole != updateSession.GetRole() {
		return nil
	}

	id, ok := newPod.ObjectMeta.Labels[updateSession.GetType()]
	if !ok {
		w.Logger.Errorw("updateworker informer, no id found in pod labels", zap.String("pod.name", newPod.Name))
		return nil
	}

	if oldPod.Status.Phase != newPod.Status.Phase {
		if newPod.Status.Phase == corev1.PodFailed {
			if err := w.Store.SetUpdateStatus(context.TODO(), id, "failed"); err != nil {
				return errors.Wrap(err, "set update status to failed")
			}

			// Leaving these sitting around for now...  we should
			// be grabbing the logs from these and writing them to
			// somewhere for analysis of failures

			// if err := w.K8sClient.CoreV1().Namespaces().Delete(newPod.Namespace, &metav1.DeleteOptions{}); err != nil {
			// 	return errors.Wrap(err, "delete namespace")
			// }

		} else if newPod.Status.Phase == corev1.PodSucceeded {
			updateSession, err := w.Store.GetUpdate(context.TODO(), id)
			if err != nil {
				return errors.Wrap(err, "get update session")
			}

			// read the secret, put the state in the database
			secret, err := w.K8sClient.CoreV1().Secrets(newPod.Namespace).Get(newPod.Name, metav1.GetOptions{})
			if err != nil {
				return errors.Wrap(err, "get secret")
			}

			var stateMetadata types.ShipStateMetadata
			err = json.Unmarshal(secret.Data["state.json"], &stateMetadata)
			if err != nil {
				return errors.Wrap(err, "unmarshal state json")
			}

			if err := w.Store.UpdateWatchFromState(context.TODO(), updateSession.WatchID, secret.Data["state.json"]); err != nil {
				return errors.Wrap(err, "update watch from state")
			}

			if err := w.Store.SetUpdateStatus(context.TODO(), id, "completed"); err != nil {
				return errors.Wrap(err, "set update status to completed")
			}

			if err := w.K8sClient.CoreV1().Namespaces().Delete(newPod.Namespace, &metav1.DeleteOptions{}); err != nil {
				return errors.Wrap(err, "delete namespace")
			}

			s3Filepath, ok := newPod.ObjectMeta.Labels["s3-filepath"]
			if !ok {
				w.Logger.Errorw("updateworker informer, no s3filepath found in pod labels", zap.String("pod.name", newPod.Name))
				return nil
			}
			decodedS3Filepath, err := base64.RawStdEncoding.DecodeString(s3Filepath)
			if err != nil {
				return errors.Wrap(err, "decode filepath")
			}

			updateSequenceStr, ok := newPod.ObjectMeta.Labels["update-sequence"]
			if !ok {
				w.Logger.Errorw("updateworker informer, no updatesequence found in pod labels", zap.String("pod.name", newPod.Name))
				return nil
			}
			updateSequence, err := strconv.Atoi(string(updateSequenceStr))
			if err != nil {
				w.Logger.Errorw("updateworker informer, unable to convert update sequence from string", zap.String("updateSequenceStr", updateSequenceStr), zap.String("pod.name", newPod.Name))
				return nil
			}

			if err := w.postUpdateActions(updateSession.WatchID, updateSequence, string(decodedS3Filepath)); err != nil {
				return errors.Wrap(err, "postUpdateActions")
			}
		}
	}

	return nil
}
