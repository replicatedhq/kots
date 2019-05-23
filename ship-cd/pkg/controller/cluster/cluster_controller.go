/*
Copyright 2019 Replicated, Inc..

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cluster

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	clustersv1alpha1 "github.com/replicatedhq/ship-cluster/ship-cd/pkg/apis/clusters/v1alpha1"
	clustersclientv1alpha1 "github.com/replicatedhq/ship-cluster/ship-cd/pkg/client/shipclusterclientset/typed/clusters/v1alpha1"

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Cluster Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCluster{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("cluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Cluster
	err = c.Watch(&source.Kind{Type: &clustersv1alpha1.Cluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	eventPoller := make(chan event.GenericEvent)
	err = c.Watch(&source.Channel{Source: eventPoller}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	go func() error {
		for {
			time.Sleep(time.Second * 20) // TODO config?

			client, err := clustersclientv1alpha1.NewForConfig(mgr.GetConfig())
			if err != nil {
				fmt.Println(err)
			}

			shipClusters, err := client.Clusters("default").List(metav1.ListOptions{})
			if err != nil {
				fmt.Println(err)
			}

			for _, shipCluster := range shipClusters.Items {
				eventPoller <- event.GenericEvent{
					Meta: &metav1.ObjectMeta{
						Name:      shipCluster.Name,
						Namespace: shipCluster.Namespace,
					},
				}
			}

		}
	}()

	return nil
}

var _ reconcile.Reconciler = &ReconcileCluster{}

// ReconcileCluster reconciles a Cluster object
type ReconcileCluster struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Cluster object and makes changes based on the state read
// and what is in the Cluster.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  The scaffolding writes
// a Deployment as an example
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=clusters.replicated.com,resources=clusters,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=clusters.replicated.com,resources=clusters/status,verbs=get;update;patch
func (r *ReconcileCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Cluster instance
	instance := &clustersv1alpha1.Cluster{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Poll for updates
	desiredState, err := getDesiredStateFromShipServer(instance.Spec.ShipAPIServer, instance.Spec.Token)
	if err != nil {
		fmt.Println(err)
		return reconcile.Result{}, err
	}

	// TODO continue if one fails
	for _, watch := range desiredState.Present {
		// As of v1alpha1 release, we only support text (yaml)
		decoded, err := base64.StdEncoding.DecodeString(string(watch))
		if err != nil {
			fmt.Println("error decodeding desired resources")
			fmt.Println(err)
			return reconcile.Result{}, err
		}

		fmt.Printf("deploying %q\n", decoded)
		if err := r.ensureResourcesPresent(decoded); err != nil {
			fmt.Println("error creating resources")
			fmt.Println(err)
			return reconcile.Result{}, err
		}

		fmt.Printf("finished deploying\n")
	}

	for _, watch := range desiredState.Missing {
		// As of v1alpha1 release, we only support text (yaml)
		decoded, err := base64.StdEncoding.DecodeString(string(watch))
		if err != nil {
			fmt.Println("error decodeding missing resources")
			fmt.Println(err)
			return reconcile.Result{}, err
		}

		fmt.Printf("removing %q\n", decoded)
		if err := r.ensureResourcesMissing(decoded); err != nil {
			fmt.Println(err)
			return reconcile.Result{}, err
		}

		fmt.Printf("finished removing\n")
	}

	return reconcile.Result{}, nil
}
