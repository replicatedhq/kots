package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

// runHooksInformer will create goroutines to start various informers for kots objects
func (c *Client) runHooksInformer() error {
	restconfig, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get in cluster config")
	}
	clientset, err := kubernetes.NewForConfig(restconfig)
	if err != nil {
		return errors.Wrap(err, "failed to get new kubernetes client")
	}

	restClient := clientset.BatchV1().RESTClient()

	c.hookStopChans = []chan struct{}{}

	// Watch jobs
	go func() {
		jobWatchList := cache.NewListWatchFromClient(restClient, "jobs", c.TargetNamespace, fields.Everything())
		resyncPeriod := 30 * time.Second

		_, controller := cache.NewInformer(jobWatchList, &batchv1.Job{}, resyncPeriod,
			cache.ResourceEventHandlerFuncs{
				UpdateFunc: func(oldObj interface{}, newObj interface{}) {
					job, ok := newObj.(*batchv1.Job)
					if !ok {
						fmt.Println("error getting new job")
						return
					}

					// if the job doesn't contain our annotation, ignore it
					hookValue, ok := job.Annotations["kots.io/hook-delete-policy"]
					if !ok {
						// fmt.Println("no annotation found on job, not going to handle any cleanup")
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
						fmt.Printf("not cleaning up job %q, active=%d, succeeeded=%d, failed=%d\n", job.Name, job.Status.Active, job.Status.Succeeded, job.Status.Failed)
						return
					}

					fmt.Printf("attempting to %s delete job %s\n", reason, job.Name)

					grace := int64(0)
					policy := metav1.DeletePropagationBackground
					opts := metav1.DeleteOptions{
						GracePeriodSeconds: &grace,
						PropagationPolicy:  &policy,
					}
					if err := clientset.BatchV1().Jobs(job.Namespace).Delete(context.TODO(), job.Name, opts); err != nil {
						fmt.Printf("error deleting job: %s\n", err.Error())
						return
					}
					fmt.Printf("deleted %s job %s\n", reason, job.Name)
				},
			},
		)
		stopChan := make(chan struct{})
		c.hookStopChans = append(c.hookStopChans, stopChan)
		controller.Run(stopChan)
	}()

	return nil
}

func (c *Client) shutdownHooksInformer() {
	for _, stopChan := range c.hookStopChans {
		stopChan <- struct{}{}
	}
}
