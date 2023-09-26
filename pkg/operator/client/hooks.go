package client

import (
	"context"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

// runHooksInformer will create goroutines to start various informers for kots objects
func (c *Client) runHooksInformer(namespace string) error {
	logger.Infof("running hooks informer for namespace %s", namespace)

	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get clientset")
	}
	restClient := clientset.BatchV1().RESTClient()

	// Watch jobs
	go func() {
		jobWatchList := cache.NewListWatchFromClient(restClient, "jobs", namespace, fields.Everything())
		resyncPeriod := 30 * time.Second

		_, controller := cache.NewInformer(jobWatchList, &batchv1.Job{}, resyncPeriod,
			cache.ResourceEventHandlerFuncs{
				UpdateFunc: func(oldObj interface{}, newObj interface{}) {

					job, ok := newObj.(*batchv1.Job)
					if !ok {
						logger.Errorf("expected batchv1.Job, but got %T", newObj)
						return
					}

					// if the job doesn't contain our annotation, ignore it
					hookValue, ok := job.Annotations["kots.io/hook-delete-policy"]
					if !ok {
						return
					}

					cleanUpJob := false
					reason := ""
					if job.Status.Active == 0 && job.Status.Succeeded > 0 && strings.Contains(hookValue, "hook-succeeded") {
						cleanUpJob = true
						reason = "successful"
					}

					if job.Status.Active == 0 && job.Status.Failed > 0 && strings.Contains(hookValue, "hook-failed") {
						cleanUpJob = true
						reason = "failed"
					}

					if !cleanUpJob {
						return
					}

					grace := int64(0)
					policy := metav1.DeletePropagationBackground
					opts := metav1.DeleteOptions{
						GracePeriodSeconds: &grace,
						PropagationPolicy:  &policy,
					}
					if err := clientset.BatchV1().Jobs(job.Namespace).Delete(context.TODO(), job.Name, opts); err != nil {
						logger.Error(errors.Wrap(err, "failed to delete job"))
						return
					}
					logger.Debugf("deleted %s job %s\n", reason, job.Name)
				},
			},
		)
		stopChan := make(chan struct{})
		c.HookStopChans = append(c.HookStopChans, stopChan)
		controller.Run(stopChan)
	}()

	return nil
}

func (c *Client) shutdownHooksInformer() {
	for _, stopChan := range c.HookStopChans {
		stopChan <- struct{}{}
	}
}
