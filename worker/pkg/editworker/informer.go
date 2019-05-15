package editworker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

var editSession types.EditSession

func (w *Worker) runInformer(ctx context.Context) error {
	debug := level.Debug(log.With(w.Logger, "method", "editeorker.Worker.runInformer"))

	debug.Log("event", "runInformer")

	restClient := w.K8sClient.CoreV1().RESTClient()
	watchlist := cache.NewListWatchFromClient(restClient, "pods", "", fields.Everything())

	resyncPeriod := 30 * time.Second

	_, controller := cache.NewInformer(watchlist, &corev1.Pod{}, resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				err := w.deleteFunc(obj)
				if err != nil {
					level.Error(w.Logger).Log("event", "edit.session.informer.pod.delete", "err", err)
				}
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				err := w.updateFunc(oldObj, newObj)
				if err != nil {
					level.Error(w.Logger).Log("event", "edit.session.informer.pod.update", "err", err)
				}
			},
		},
	)

	controller.Run(ctx.Done())
	return ctx.Err()
}

func (w *Worker) deleteFunc(obj interface{}) error {
	debug := level.Debug(log.With(w.Logger, "method", "editworker.Worker.deleteFunc"))

	pod, ok := obj.(*corev1.Pod)
	if !ok {
		return fmt.Errorf("unexpected type %T", obj)
	}

	debug.Log("event", "deleteFunc", "pod", pod.Name)

	shipCloudRole, ok := pod.ObjectMeta.Labels["shipcloud-role"]
	if !ok || shipCloudRole != editSession.GetRole() {
		return nil
	}

	id, ok := pod.ObjectMeta.Labels[editSession.GetType()]
	if !ok {
		level.Error(w.Logger).Log("event", "no edit label")
		return nil
	}

	// Why did the pod exit?
	debug.Log("edit-id", id, "pod phase", pod.Status.Phase)

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
	if !ok || shipCloudRole != editSession.GetRole() {
		return nil
	}

	id, ok := newPod.ObjectMeta.Labels[editSession.GetType()]
	if !ok {
		level.Error(w.Logger).Log("event", "no update label")
		return nil
	}

	if oldPod.Status.Phase != newPod.Status.Phase {
		if newPod.Status.Phase == corev1.PodFailed {
			if err := w.Store.SetEditStatus(context.TODO(), id, "failed"); err != nil {
				return errors.Wrap(err, "set edit status to failed")
			}

			// Leaving these sitting around for now...  we should
			// be grabbing the logs from these and writing them to
			// somewhere for analysis of failures

			// if err := w.K8sClient.CoreV1().Namespaces().Delete(newPod.Namespace, &metav1.DeleteOptions{}); err != nil {
			// 	return errors.Wrap(err, "delete namespace")
			// }

		} else if newPod.Status.Phase == corev1.PodSucceeded {
			editSession, err := w.Store.GetEdit(context.TODO(), id)
			if err != nil {
				return errors.Wrap(err, "get edit session")
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

			if err := w.Store.UpdateWatchFromState(context.TODO(), editSession.WatchID, secret.Data["state.json"]); err != nil {
				return errors.Wrap(err, "update watch from state")
			}

			if err := w.Store.SetEditStatus(context.TODO(), id, "completed"); err != nil {
				return errors.Wrap(err, "set edit status to completed")
			}

			if err := w.K8sClient.CoreV1().Namespaces().Delete(newPod.Namespace, &metav1.DeleteOptions{}); err != nil {
				return errors.Wrap(err, "delete namespace")
			}

			s3Filepath, ok := newPod.ObjectMeta.Labels["s3-filepath"]
			if !ok {
				level.Error(w.Logger).Log("event", "no s3filepath")
				return nil
			}
			decodedS3Filepath, err := base64.RawStdEncoding.DecodeString(s3Filepath)
			if err != nil {
				return errors.Wrap(err, "decode filepath")
			}

			editSequenceStr, ok := newPod.ObjectMeta.Labels["edit-sequence"]
			if !ok {
				level.Error(w.Logger).Log("event", "no edit sequence")
				return nil
			}
			editSequence, err := strconv.Atoi(string(editSequenceStr))
			if err != nil {
				level.Error(w.Logger).Log("event", "convert edit sequence")
				return nil
			}

			if err := w.postEditActions(editSession.WatchID, editSequence, string(decodedS3Filepath)); err != nil {
				return errors.Wrap(err, "postEditActions")
			}
		}
	}

	return nil
}
