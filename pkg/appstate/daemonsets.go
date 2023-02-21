package appstate

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/replicatedhq/kots/pkg/appstate/types"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const DaemonSetResourceKind = "daemonset"
const DaemonSetPodVersionLabel = "controller-revision-hash"

type daemonSetEventHandler struct {
	informers       []types.StatusInformer
	resourceStateCh chan<- types.ResourceState
	clientset       kubernetes.Interface
	targetNamespace string
}

type validContainer struct {
	exists      bool
	volumeNames map[string]bool
}

func init() {
	registerResourceKindNames(DaemonSetResourceKind, "daemonsets", "ds")
}

func runDaemonSetController(ctx context.Context, clientset kubernetes.Interface,
	targetNamespace string, informers []types.StatusInformer, resourceStateCh chan<- types.ResourceState,
) {
	listwatch := &cache.ListWatch{
		ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
			return clientset.AppsV1().DaemonSets(targetNamespace).List(context.TODO(), options)
		},
		WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
			return clientset.AppsV1().DaemonSets(targetNamespace).Watch(context.TODO(), options)
		},
	}

	informer := cache.NewSharedInformer(listwatch, &appsv1.DaemonSet{}, time.Minute)

	eventHandler := &daemonSetEventHandler{
		informers:       filterStatusInformersByResourceKind(informers, DaemonSetResourceKind),
		resourceStateCh: resourceStateCh,
		clientset:       clientset,
		targetNamespace: targetNamespace,
	}

	runInformer(ctx, informer, eventHandler)
}

func (h *daemonSetEventHandler) ObjectCreated(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}

	h.resourceStateCh <- makeDaemonSetResourceState(r, h.calculateDaemonSetState(r))
}

func (h *daemonSetEventHandler) ObjectDeleted(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}

	h.resourceStateCh <- makeDaemonSetResourceState(r, types.StateMissing)
}

func (h *daemonSetEventHandler) ObjectUpdated(obj interface{}) {
	r := h.cast(obj)
	if _, ok := h.getInformer(r); !ok {
		return
	}

	h.resourceStateCh <- makeDaemonSetResourceState(r, h.calculateDaemonSetState(r))
}

func (h *daemonSetEventHandler) getInformer(r *appsv1.DaemonSet) (types.StatusInformer, bool) {
	if r != nil {
		for _, informer := range h.informers {
			if r.Namespace == informer.Namespace && r.Name == informer.Name {
				return informer, true
			}
		}
	}

	return types.StatusInformer{}, false
}

func (h *daemonSetEventHandler) cast(obj interface{}) *appsv1.DaemonSet {
	r, _ := obj.(*appsv1.DaemonSet)
	return r
}

// The pods in a daemonset can be identified by the match label set in the daemonset and the
// daemonset template spec is compared against running pods' specs to determine the updating state.
func (h *daemonSetEventHandler) calculateDaemonSetState(r *appsv1.DaemonSet) types.State {
	if r == nil {
		return types.StateUnavailable
	}

	// Generate maps of valid data from the template to check against the pod.
	// Note: the resource requests between the template and spec can differ.
	validContainers := make(map[string]validContainer, 0)
	for i := range r.Spec.Template.Spec.Containers {
		r.Spec.Template.Spec.Containers[i].Resources.Requests = nil
		container := r.Spec.Template.Spec.Containers[i]

		temp := validContainer{exists: true}
		for _, volume := range container.VolumeMounts {
			if temp.volumeNames == nil {
				temp.volumeNames = make(map[string]bool)
			}

			temp.volumeNames[volume.Name] = true
		}

		validContainers[container.Name] = temp
	}

	validImagePullSecrets := make(map[string]bool, 0)
	for _, secret := range r.Spec.Template.Spec.ImagePullSecrets {
		validImagePullSecrets[secret.Name] = true
	}

	validVolumes := make(map[string]bool, 0)
	for _, volume := range r.Spec.Template.Spec.Volumes {
		validVolumes[volume.Name] = true
	}

	// The affinities and termination grace period will be different from the template spec and running nodes.
	r.Spec.Template.Spec.Affinity = nil
	r.Spec.Template.Spec.TerminationGracePeriodSeconds = nil

	templateBytes, err := json.Marshal(r.Spec.Template.Spec)
	if err != nil {
		log.Printf("failed to marshal the template spec: %s", err)
		return types.StateUnavailable
	}

	hasher := sha256.New()
	hasher.Write(templateBytes)
	validHash := hasher.Sum(nil)

	// Select all pods that match the daemonset.
	selector := ""
	for key, value := range r.Spec.Selector.MatchLabels {
		if len(selector) > 0 {
			selector += ","
		}
		selector += fmt.Sprintf("%s=%s", key, value)
	}

	pods, err := h.clientset.CoreV1().Pods(h.targetNamespace).List(context.TODO(), metav1.ListOptions{LabelSelector: selector})
	if err != nil {
		log.Printf("failed to get daemonset pod list: %s", err)
		return types.StateUnavailable
	}

	// Remove things from the spec that won't match then compare the hashes. If they differ, then the daemonset is being updated.
	for _, pod := range pods.Items {
		for i, container := range pod.Spec.Containers {
			if validContainers[container.Name].exists {
				container.Env = nil
				container.Resources.Requests = nil

				mounts := []corev1.VolumeMount{}
				for _, volume := range container.VolumeMounts {
					if validContainers[container.Name].volumeNames[volume.Name] {
						mounts = append(mounts, volume)
					}
				}
				container.VolumeMounts = mounts
			}

			pod.Spec.Containers[i] = container
		}

		secrets := []corev1.LocalObjectReference{}
		for _, secret := range pod.Spec.ImagePullSecrets {
			if validImagePullSecrets[secret.Name] {
				secrets = append(secrets, secret)
			}
		}
		pod.Spec.ImagePullSecrets = secrets

		volumes := []corev1.Volume{}
		for _, volume := range pod.Spec.Volumes {
			if validVolumes[volume.Name] {
				volumes = append(volumes, volume)
			}
		}
		pod.Spec.Volumes = volumes

		pod.Spec.Affinity = nil
		pod.Spec.DeprecatedServiceAccount = ""
		pod.Spec.EnableServiceLinks = nil
		pod.Spec.NodeName = ""
		pod.Spec.PreemptionPolicy = nil
		pod.Spec.Priority = nil
		pod.Spec.ServiceAccountName = ""
		pod.Spec.Tolerations = nil
		pod.Spec.TerminationGracePeriodSeconds = nil

		podBytes, err := json.Marshal(pod.Spec)
		if err != nil {
			log.Printf("failed to marshal the pod spec: %s", err)
			return types.StateUnavailable
		}

		hasher.Reset()
		hasher.Write(podBytes)
		podHash := hasher.Sum(nil)

		if bytes.Compare(podHash, validHash) != 0 {
			return types.StateUpdating
		}
	}

	if r.Status.NumberUnavailable > 0 {
		return types.StateDegraded
	}

	if r.Status.NumberMisscheduled > 0 {
		return types.StateDegraded
	}

	if r.Status.CurrentNumberScheduled != r.Status.DesiredNumberScheduled {
		return types.StateDegraded
	}

	if r.Status.NumberReady >= r.Status.DesiredNumberScheduled {
		return types.StateReady
	}

	if r.Status.NumberReady > 0 {
		return types.StateDegraded
	}

	return types.StateUnavailable
}

func makeDaemonSetResourceState(r *appsv1.DaemonSet, state types.State) types.ResourceState {
	return types.ResourceState{
		Kind:      DaemonSetResourceKind,
		Name:      r.Name,
		Namespace: r.Namespace,
		State:     state,
	}
}
