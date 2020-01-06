package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// runHooksInformer will create goroutines to start various informers for kots objects
func (c *Client) runHooksInformer(clientset *kubernetes.Clientset) error {
	restClient := clientset.RESTClient()

	// Watch jobs
	go func() {
		ctx := context.Background()

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

		controller.Run(ctx.Done())
	}()
}
