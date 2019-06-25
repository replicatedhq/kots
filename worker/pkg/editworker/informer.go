package editworker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/ship-cluster/worker/pkg/ship"
	"github.com/replicatedhq/ship-cluster/worker/pkg/types"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

var editSession types.EditSession

func (w *Worker) runInformer(ctx context.Context) error {
	restClient := w.K8sClient.CoreV1().RESTClient()
	watchlist := cache.NewListWatchFromClient(restClient, "pods", "", fields.Everything())

	resyncPeriod := 30 * time.Second

	_, controller := cache.NewInformer(watchlist, &corev1.Pod{}, resyncPeriod,
		cache.ResourceEventHandlerFuncs{
			DeleteFunc: func(obj interface{}) {
				err := w.deleteFunc(obj)
				if err != nil {
					w.Logger.Errorw("error in editworker informer deleteFunc", zap.Error(err))
				}
			},
			UpdateFunc: func(oldObj interface{}, newObj interface{}) {
				err := w.updateFunc(oldObj, newObj)
				if err != nil {
					w.Logger.Errorw("error in editworker informer updateFunc", zap.Error(err))
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

	w.Logger.Debugw("editworker is starting deleteFunc for pod", zap.String("pod.name", pod.Name))

	shipCloudRole, ok := pod.ObjectMeta.Labels["shipcloud-role"]
	if !ok || shipCloudRole != editSession.GetRole() {
		return nil
	}

	id, ok := pod.ObjectMeta.Labels[editSession.GetType()]
	if !ok {
		w.Logger.Errorw("editworker informer expected to find edit label in pod", zap.String("id", id), zap.String("pod.name", pod.Name))
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
	if !ok || shipCloudRole != editSession.GetRole() {
		return nil
	}

	id, ok := newPod.ObjectMeta.Labels[editSession.GetType()]
	if !ok {
		w.Logger.Errorw("editworker informer expected to find udpate label in pod", zap.String("pod.name", oldPod.Name))
		return nil
	}

	if oldPod.Status.Phase == newPod.Status.Phase {
		return nil
	}

	shipState := ship.NewStateManager(w.Config)
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
		defer deleteState()

		editSession, err := w.Store.GetEdit(context.TODO(), id)
		if err != nil {
			return errors.Wrap(err, "get edit session")
		}

		stateJSON, err := shipState.GetState(stateID)
		if err != nil {
			return errors.Wrap(err, "get secret")
		}

		var stateMetadata types.ShipStateMetadata
		err = json.Unmarshal(stateJSON, &stateMetadata)
		if err != nil {
			return errors.Wrap(err, "unmarshal state json")
		}

		if err := w.Store.UpdateWatchFromState(context.TODO(), editSession.WatchID, stateJSON); err != nil {
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
			w.Logger.Errorw("editworker informer, no s3 filepath found in pod labels", zap.String("pod.name", newPod.Name))
			return nil
		}
		decodedS3Filepath, err := base64.RawStdEncoding.DecodeString(s3Filepath)
		if err != nil {
			return errors.Wrap(err, "decode filepath")
		}

		editSequenceStr, ok := newPod.ObjectMeta.Labels["edit-sequence"]
		if !ok {
			w.Logger.Errorw("editworker informer, no edit sequence found in pod labels", zap.String("pod.name", newPod.Name))
			return nil
		}
		editSequence, err := strconv.Atoi(string(editSequenceStr))
		if err != nil {
			w.Logger.Errorw("editworker informer, failed to convert edit sequence from string", zap.String("pod.name", newPod.Name), zap.String("editSequence", editSequenceStr))
			return nil
		}

		if err := w.postEditActions(editSession.WatchID, editSequence, string(decodedS3Filepath)); err != nil {
			return errors.Wrap(err, "postEditActions")
		}
	}

	return nil
}
