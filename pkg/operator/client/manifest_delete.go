package client

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/operator/applier"
	"github.com/replicatedhq/kots/pkg/operator/types"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	DefaultDeletionPlan = types.Plan{
		BeforeAll: []string{
			"APIService",
			"Ingress",
			"Service",
			"Pod",
			"CronJob",
			"Job",
			"StatefulSet",
			"HorizontalPodAutoscaler",
			"Deployment",
			"ReplicaSet",
			"ReplicationController",
			"DaemonSet",
		},
		AfterAll: []string{
			"RoleBindingList",
			"RoleBinding",
			"RoleList",
			"Role",
			"ClusterRoleBindingList",
			"ClusterRoleBinding",
			"ClusterRoleList",
			"ClusterRole",
			"CustomResourceDefinition",
			"PersistentVolumeClaim",
			"PersistentVolume",
			"ConfigMap",
			"SecretList",
			"Secret",
			"ServiceAccount",
			"PodDisruptionBudget",
			"PodSecurityPolicy",
			"LimitRange",
			"ResourceQuota",
		},
	}
)

func deleteManifests(manifests []string, targetNS string, kubernetesApplier applier.KubectlInterface, plan types.Plan, waitFlag bool) {
	resources := decodeManifests(manifests)
	deleteResources(resources, targetNS, kubernetesApplier, plan, waitFlag)
}

func deleteResources(resources types.Resources, targetNS string, kubernetesApplier applier.KubectlInterface, plan types.Plan, waitFlag bool) {
	resources = resources.SortWithPlan(plan)
	for _, r := range resources {
		deleteResource(r, targetNS, waitFlag, kubernetesApplier)
	}
}

func deleteResource(resource types.Resource, targetNS string, waitFlag bool, kubernetesApplier applier.KubectlInterface) {
	group := ""
	kind := ""
	name := ""
	namespace := targetNS
	wait := waitFlag

	if resource.GVK != nil {
		group = resource.GVK.Group
		kind = resource.GVK.Kind
		unstructured := resource.Unstructured
		wait = shouldWaitForResourceDeletion(resource.GVK.Kind, waitFlag)

		if unstructured != nil {
			name = unstructured.GetName()
			if ns := unstructured.GetNamespace(); ns != "" {
				namespace = ns
			} else {
				namespace = targetNS
			}
		}
		logger.Infof("deleting manifest: %s/%s/%s/%s", resource.GVK.Group, resource.GVK.Version, resource.GVK.Kind, name)
	} else {
		logger.Infof("deleting unidentified manifest: %s", resource.Manifest)
	}

	stdout, stderr, err := kubernetesApplier.Remove(namespace, []byte(resource.Manifest), wait)
	if err != nil {
		logger.Infof("stdout (delete) = %s", stdout)
		logger.Infof("stderr (delete) = %s", stderr)
		logger.Infof("error: %s", err.Error())
	} else {
		logger.Infof("manifest deleted: %s/%s/%s", group, kind, name)
	}
}

func decodeManifests(manifests []string) types.Resources {
	resources := types.Resources{}

	for _, manifest := range manifests {
		resource := types.Resource{
			Manifest: manifest,
		}

		unstruct := &unstructured.Unstructured{}
		_, gvk, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(manifest), nil, unstruct)
		if err != nil {
			logger.Infof("error decoding manifest: %v", err.Error())
		} else {
			resource.Unstructured = unstruct
			resource.GVK = gvk
		}

		resources = append(resources, resource)
	}

	return resources
}

func shouldWaitForResourceDeletion(kind string, waitFlag bool) bool {
	if kind == "PersistentVolumeClaim" {
		// blocking on PVC delete will create a deadlock if
		// it's used by a pod that has not been deleted yet.
		return false
	}
	return waitFlag
}

func clearNamespaces(appSlug string, namespacesToClear []string, isRestore bool, restoreLabelSelector labels.Selector, plan types.Plan, k8sDynamicClient dynamic.Interface, gvrs map[schema.GroupVersionResource]struct{}) error {
	// 2 minute wait, 60 loops with 2 second sleep
	waitTimeout := 60
	waitSleepInSec := 2 * time.Second
	// Extra time in case the app-slug annotation was not being used.
	// This is the time it takes for the app controller to delete the app
	// after the namespace is deleted.
	waitExtraInSec := 20 * time.Second

	// skip resources that don't have API endpoints or don't have applied objects
	var skip = sets.NewString(
		"/v1/bindings",
		"/v1/events",
		"extensions/v1beta1/replicationcontrollers",
		"apps/v1/controllerrevisions",
		"authentication.k8s.io/v1/tokenreviews",
		"authorization.k8s.io/v1/localsubjectaccessreviews",
		"authorization.k8s.io/v1/subjectaccessreviews",
		"authorization.k8s.io/v1/selfsubjectaccessreviews",
		"authorization.k8s.io/v1/selfsubjectrulesreviews",
	)

	gvrsToDelete := []schema.GroupVersionResource{}
	for gvr := range gvrs {
		s := fmt.Sprintf("%s/%s/%s", gvr.Group, gvr.Version, gvr.Resource)
		if !skip.Has(s) {
			gvrsToDelete = append(gvrsToDelete, gvr)
		}
	}

	err := clearNamespacesWithWait(appSlug, namespacesToClear, isRestore, restoreLabelSelector, plan, k8sDynamicClient, gvrsToDelete, waitTimeout, waitSleepInSec, waitExtraInSec)
	if err != nil {
		return errors.Wrap(err, "error deleting namespaces")
	}

	return nil
}

func clearNamespacesWithWait(appSlug string, namespacesToClear []string, isRestore bool, restoreLabelSelector labels.Selector, plan types.Plan, k8sDynamicClient dynamic.Interface, gvrsToDelete []schema.GroupVersionResource, waitTimeOut int, waitSleep time.Duration, waitExtra time.Duration) error {
	for _, namespace := range namespacesToClear {
		logger.Infof("Ensuring all %s objects have been removed from namespace %s\n", appSlug, namespace)

		for i := waitTimeOut; i >= 0; i-- { // 2 minute wait, 60 loops with 2 second sleep
			resources := getResourcesInNamespace(k8sDynamicClient, gvrsToDelete, appSlug, namespace, isRestore, restoreLabelSelector)
			if len(resources) == 0 {
				logger.Infof("Namespace %s successfully cleared of app %s\n", namespace, appSlug)
				break
			}

			resources = resources.SortWithPlan(plan)

			for _, r := range resources {
				if r.Unstructured.GetDeletionTimestamp() != nil {
					logger.Infof("Pending deletion %s/%s/%s\n", namespace, r.GVR, r.Unstructured.GetName())
					continue
				}

				logger.Infof("Deleting %s/%s/%s\n", namespace, r.GVR, r.Unstructured.GetName())

				if err := k8sDynamicClient.Resource(r.GVR).Namespace(namespace).Delete(context.TODO(), r.Unstructured.GetName(), metav1.DeleteOptions{}); err != nil {
					logger.Errorf("Resource %s (%s) in namespace %s could not be deleted: %v\n", r.Unstructured.GetName(), r.GVR, namespace, err)
				}
			}

			if i == 0 {
				return fmt.Errorf("Failed to clear app %s from namespace %s\n", appSlug, namespace)
			}

			logger.Infof("Namespace %s still has objects from app %s: sleeping...\n", namespace, appSlug)
			time.Sleep(waitSleep)
		}
	}

	if len(namespacesToClear) > 0 {
		// Extra time in case the app-slug annotation was not being used.
		time.Sleep(waitExtra)
	}

	return nil
}

func getResourcesInNamespace(dyn dynamic.Interface, gvrs []schema.GroupVersionResource, appSlug string, namespace string, isRestore bool, restoreLabelSelector labels.Selector) types.Resources {
	resources := types.Resources{}

	for _, gvr := range gvrs {
		// there may be other resources that can't be listed besides what's in the skip set so ignore error
		unstructuredList, err := dyn.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
		if unstructuredList == nil {
			if err != nil {
				logger.Errorf("failed to list namespace resources: %s", err.Error())
			}
			continue
		}

		for _, u := range unstructuredList.Items {
			if isRestore {
				itemLabelMap := u.GetLabels()
				if excludeLabel, exists := itemLabelMap["velero.io/exclude-from-backup"]; exists && excludeLabel == "true" {
					continue
				}

				itemLabelSet := labels.Set(itemLabelMap)
				if restoreLabelSelector != nil && !restoreLabelSelector.Matches(itemLabelSet) {
					continue
				}
			}

			annotations := u.GetAnnotations()
			if annotations["kots.io/app-slug"] == appSlug {
				// copy the object so we don't modify the cache
				unstruct := u.DeepCopy()
				gvk := unstruct.GetObjectKind().GroupVersionKind()
				resource := types.Resource{
					Unstructured: unstruct,
					GVK:          &gvk,
					GVR:          gvr,
				}
				resources = append(resources, resource)
			}
		}
	}

	return resources
}

func deletePVCs(namespace string, appLabelSelector *metav1.LabelSelector, appslug string, clientset kubernetes.Interface) error {
	if appLabelSelector == nil {
		appLabelSelector = &metav1.LabelSelector{
			MatchLabels: map[string]string{},
		}
	}
	appLabelSelector.MatchLabels["kots.io/app-slug"] = appslug

	podsList, err := clientset.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: getLabelSelector(appLabelSelector),
	})
	if err != nil {
		return errors.Wrap(err, "failed to get list of app pods")
	}

	pvcs := make([]string, 0)
	for _, pod := range podsList.Items {
		for _, v := range pod.Spec.Volumes {
			if v.PersistentVolumeClaim != nil {
				pvcs = append(pvcs, v.PersistentVolumeClaim.ClaimName)
			}
		}
	}

	if len(pvcs) == 0 {
		logger.Infof("no pvcs to delete in %s for pods that match %s", namespace, getLabelSelector(appLabelSelector))
		return nil
	}
	logger.Infof("deleting %d pvcs in %s for pods that match %s", len(pvcs), namespace, getLabelSelector(appLabelSelector))

	for _, pvc := range pvcs {
		grace := int64(0)
		policy := metav1.DeletePropagationBackground
		opts := metav1.DeleteOptions{
			GracePeriodSeconds: &grace,
			PropagationPolicy:  &policy,
		}
		logger.Infof("deleting pvc: %s", pvc)
		err := clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(context.TODO(), pvc, opts)
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to delete pvc %s", pvc)
		}
	}

	return nil
}

func getLabelSelector(appLabelSelector *metav1.LabelSelector) string {
	allKeys := make([]string, 0)
	for key := range appLabelSelector.MatchLabels {
		allKeys = append(allKeys, key)
	}

	sort.Strings(allKeys)

	allLabels := make([]string, 0)
	for _, key := range allKeys {
		allLabels = append(allLabels, fmt.Sprintf("%s=%s", key, appLabelSelector.MatchLabels[key]))
	}

	return strings.Join(allLabels, ",")
}
