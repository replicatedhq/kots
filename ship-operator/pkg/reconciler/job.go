package reconciler

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func jobIsRunning(job *batchv1.Job) bool {
	if job == nil {
		return false
	}
	return !jobIsComplete(job)
}

func jobIsComplete(job *batchv1.Job) bool {
	if job == nil {
		return false
	}
	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobComplete && c.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

func (r *Reconciler) deleteWatchJob() error {
	err := r.Client.Delete(context.TODO(), r.watchJob)
	if err != nil {
		level.Error(log.With(r.logger)).Log("method", "reconciler.deleteWatchJob", "error", err)
		return err
	}
	return nil
}

func (r *Reconciler) deleteUpdateJob() error {
	level.Debug(log.With(r.logger)).Log("method", "Reconciler.deleteUpdateJob", "name", r.updateJob.Name, "ns", r.updateJob.Namespace)
	err := r.Client.Delete(context.TODO(), r.updateJob)
	if err != nil {
		level.Error(log.With(r.logger)).Log("method", "reconciler.deleteUpdateJob", "error", err)
		return err
	}
	return nil
}

func (r *Reconciler) createWatchJob() error {
	job := r.generator.WatchJob(r.stateSecretSHA())
	if err := r.Client.Create(context.TODO(), job); err != nil {
		level.Error(log.With(r.logger)).Log("method", "createWatchJob", "error", err)
		return err
	}
	return nil
}

func (r *Reconciler) createUpdateJob() error {
	job := r.generator.UpdateJob(r.stateSecretSHA())
	if err := r.Client.Create(context.TODO(), job); err != nil {
		level.Error(log.With(r.logger)).Log("method", "createUpdateJob", "error", err)
		return err
	}
	return nil
}

func (r *Reconciler) shouldUpdateJob(found, desired *batchv1.Job) bool {
	debug := level.Debug(log.With(r.logger, "method", "Reconciler.shouldUpdateJob"))

	if found.Name != desired.Name {
		debug.Log("update.required", "true", "event", "name.changed", "old", found.Name, "new", desired.Name)
		return true
	}
	// ok if found has other annotations beyond desired
	if !isSubset(desired.GetAnnotations(), found.GetAnnotations()) {
		debug.Log("update.required", "true", "event", "annotations.changed", "old", fmt.Sprintf("%+v", found.GetAnnotations()), "new", fmt.Sprintf("%+v", desired.GetAnnotations()))
		return true
	}
	if found.Spec.Template.Spec.RestartPolicy != desired.Spec.Template.Spec.RestartPolicy {
		debug.Log("update.required", "true", "restart.policy.changed", "old", found.Spec.Template.Spec.RestartPolicy, "new", desired.Spec.Template.Spec.RestartPolicy)
		return true
	}
	if r.shouldUpdateContainerList(found.Spec.Template.Spec.InitContainers, desired.Spec.Template.Spec.InitContainers) {
		debug.Log("update.required", "true", "event", "init.containers.changed")
		return true
	}
	if r.shouldUpdateContainerList(found.Spec.Template.Spec.Containers, desired.Spec.Template.Spec.Containers) {
		debug.Log("update.required", "true", "event", "containers.changed")
		return true
	}
	if !reflect.DeepEqual(found.Spec.Template.Spec.Volumes, desired.Spec.Template.Spec.Volumes) {
		debug.Log("update.required", "true", "event", "volumes.changed")
		return true
	}

	debug.Log("update.required", "false")
	return false
}

func (r *Reconciler) pruneCompletedPods() error {
	ctx := context.TODO()

	opts := &client.ListOptions{
		Namespace: r.instance.Namespace,
	}
	pods := &corev1.PodList{}
	if err := r.List(ctx, opts, pods); err != nil {
		level.Error(log.With(r.logger)).Log("method", "pruneCompletedPods", "error", err)
		return err
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodSucceeded {
			continue
		}

		deletePod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: pod.Namespace,
				Name:      pod.Name,
			},
		}

		level.Debug(log.With(r.logger)).Log("method", "Reconciler.deletePod", "name", pod.Name, "ns", pod.Namespace)
		if err := r.Client.Delete(ctx, deletePod); err != nil {
			level.Error(log.With(r.logger)).Log("method", "reconciler.deletePod", "error", err)
			// seems like we should attempt to delete all pods even in the face of errors...
			continue
		}

	}

	return nil
}

// returns true if every key value pair in a is also in b
func isSubset(a, b map[string]string) bool {
	for key, aValue := range a {
		bValue, ok := b[key]
		if !ok {
			return false
		}
		if aValue != bValue {
			return false
		}
	}
	return true
}
