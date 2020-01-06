package client

import (
	"fmt"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/rest"
	"github.com/pkg/errors"
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

	restClient := clientset.RESTClient()

	c.hookStopChans = []chan struct{}{}

	// Watch jobs
	go func() {
		jobWatchList := cache.NewListWatchFromClient(restClient, "jobs", "", fields.Everything())
		resyncPeriod := 30 * time.Second

		_, controller := cache.NewInformer(jobWatchList, &batchv1.Job{}, resyncPeriod,
			cache.ResourceEventHandlerFuncs{
				UpdateFunc: func(oldObj interface{}, newObj interface{}) {
					job, ok := newObj.(*batchv1.Job)
					if !ok {
						fmt.Errorf("error getting new job")
						return
					}

					// if the job doesn't contain our annotation, ignore it
					hookValue, ok := job.Annotations["kots.io/hook-delete-policy"]
					if !ok {
						return
					}

					cleanUpJob := false
					if job.Status.Active == 0 && job.Status.Succeeded == 0 && strings.Contains(hookValue, "hook-succeeded") {
						cleanUpJob = true
					}

					if job.Status.Active == 0 && job.Status.Succeeded > 0 && strings.Contains(hookValue, "hook-failed") {
						cleanUpJob = true
					}

					if !cleanUpJob {
						return
					}

					fmt.Printf("atempting to delete job %s\n", job.Name)
					if err := clientset.BatchV1().Jobs(job.Namespace).Delete(job.Name, &metav1.DeleteOptions{}); err != nil {
						fmt.Errorf("error deleting job: %s", err.Error())
						return
					}
					fmt.Printf("deleted job %s\n", job.Name)
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
