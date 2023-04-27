package client

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/operator/applier"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	labels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
)

// KindSortOrder is an ordering of Kinds.
type KindSortOrder []string

type KindOrder struct {
	PreOrder  KindSortOrder
	PostOrder KindSortOrder
}

// resource deletion order
var (
	defaultKindDeleteOrder = KindOrder{
		PreOrder: KindSortOrder{
			"CustomResource",
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
		PostOrder: KindSortOrder{
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

type resource struct {
	Manifest     string
	GVR          schema.GroupVersionResource
	GVK          *schema.GroupVersionKind
	Unstructured *unstructured.Unstructured
}

// initResourceKindOrderMap initializes a map of resource type to a list of kinds
func initResourceKindOrderMap(kindOrder KindOrder) map[string][]resource {
	resourceOrderMap := make(map[string][]resource)
	for _, resourceType := range kindOrder.PreOrder {
		resourceOrderMap[resourceType] = []resource{}
	}
	for _, resourceType := range kindOrder.PostOrder {
		resourceOrderMap[resourceType] = []resource{}
	}
	return resourceOrderMap
}

// take KindOrder and a default list of string and return a list of kinds in the order they should be deleted
func getOrderedKinds(kindOrder KindOrder, defaultKinds KindSortOrder) KindSortOrder {
	sort.Strings(defaultKinds)
	orderedKinds := KindSortOrder{}
	orderedKinds = append(orderedKinds, kindOrder.PreOrder...)
	orderedKinds = append(orderedKinds, defaultKinds...)
	orderedKinds = append(orderedKinds, kindOrder.PostOrder...)
	return orderedKinds
}

func deleteManifestResources(manifests []string, targetNS string, kubernetesApplier applier.KubectlInterface, kindDeleteOrder KindOrder, waitFlag bool) {
	resources := decodeManifests(manifests)
	crdGVKMap := buildCrdGVKMap(resources)
	deleteOrderResourceMap, deleteOrderedKinds := buildDeleteKindOrderedResources(kindDeleteOrder, resources, crdGVKMap)

	for _, kind := range deleteOrderedKinds {
		logger.Infof("deleting resources of kind: %s", kind)
		for _, r := range deleteOrderResourceMap[kind] {
			deleteManifestResource(r, targetNS, waitFlag, kubernetesApplier)
		}
	}
}

func deleteManifestResource(resource resource, targetNS string, waitFlag bool, kubernetesApplier applier.KubectlInterface) {
	group := ""
	kind := ""
	name := ""
	namespace := targetNS
	wait := waitFlag
	if resource.GVK != nil {
		group = resource.GVK.Group
		kind = resource.GVK.Kind
		unstructured := resource.Unstructured
		wait = shouldResourceWaitForDeletion(resource.GVK.Kind, waitFlag)
		if unstructured != nil {
			name = unstructured.GetName()
			if ns := unstructured.GetNamespace(); ns != "" {
				namespace = ns
			} else {
				namespace = targetNS
			}
		}
		logger.Infof("deleting manifest(s): %s/%s/%s/%s", resource.GVK.Group, resource.GVK.Version, resource.GVK.Kind, name)
	} else {
		logger.Infof("deleting unidentified manifest: %s", resource.Manifest)
	}

	stdout, stderr, err := kubernetesApplier.Remove(namespace, []byte(resource.Manifest), wait)
	if err != nil {
		logger.Infof("stdout (delete) = %s", stdout)
		logger.Infof("stderr (delete) = %s", stderr)
		logger.Infof("error: %s", err.Error())
	} else {
		logger.Infof("manifest(s) deleted: %s/%s/%s", group, kind, name)
	}
}

func decodeManifests(manifests []string) []resource {
	resources := []resource{}
	for _, manifest := range manifests {
		obj, gvk, err := decodeToUnstructured(manifest)
		if err != nil {
			logger.Infof("error decoding manifest: %v", err.Error())
		}
		resources = append(resources, resource{
			Unstructured: obj,
			GVK:          gvk,
			Manifest:     manifest,
		})
	}
	return resources
}

func decodeToUnstructured(manifest string) (*unstructured.Unstructured, *schema.GroupVersionKind, error) {
	unstruct := &unstructured.Unstructured{}
	_, gvk, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(manifest), nil, unstruct)
	if err != nil {
		return nil, nil, errors.Wrap(err, "error decoding manifest")
	}

	return unstruct, gvk, nil
}

// buildCrdGVKMap builds a map of key group/kind/version for CRDs from a list of resources
func buildCrdGVKMap(resources []resource) map[string]bool {
	var crdGVKMap = make(map[string]bool)
	for _, r := range resources {
		if r.GVK.Kind == "CustomResourceDefinition" {
			// convert unstructured to CRD. if fails, skip and continue. manifest will be deleted in the default order
			crd := &apiextensionsv1beta1.CustomResourceDefinition{}
			err := runtime.DefaultUnstructuredConverter.FromUnstructured(r.Unstructured.Object, &crd)
			if err != nil {
				logger.Infof("error converting unstructured to CRD %v: %v", r.Unstructured.GetName, err.Error())
				continue
			}

			for _, version := range crd.Spec.Versions {
				crdGVK := buildGVKKey(crd.Spec.Group, crd.Spec.Names.Kind, version.Name)
				crdGVKMap[crdGVK] = true
			}
		}
	}
	return crdGVKMap
}

// buildDeleteKindOrderedResources builds a list of resource kinds in the order they should be deleted and a map of resource kind to a list of resources
func buildDeleteKindOrderedResources(deleteKindOrder KindOrder, resources []resource, crdGVKMap map[string]bool) (map[string][]resource, KindSortOrder) {
	defaultOrder := []string{}
	deleteOrderResourceMap := initResourceKindOrderMap(deleteKindOrder)
	for _, r := range resources {
		// if GVK is nil, add it to the "" list, else add it to the GVK list
		if r.GVK == nil {
			if _, ok := deleteOrderResourceMap[""]; !ok {
				defaultOrder = append(defaultOrder, "")
				deleteOrderResourceMap[""] = []resource{}
			}
			deleteOrderResourceMap[""] = append(deleteOrderResourceMap[""], r)
			continue
		}

		crdGVK := buildGVKKey(r.GVK.Group, r.GVK.Kind, r.GVK.Version)
		if _, ok := crdGVKMap[crdGVK]; ok {
			// if customResource is in deleteOrderMap, add it to the list, else add it to the CustomResource list
			if _, ok := deleteOrderResourceMap[r.GVK.Kind]; ok {
				deleteOrderResourceMap[r.GVK.Kind] = append(deleteOrderResourceMap[r.GVK.Kind], r)
			} else {
				deleteOrderResourceMap["CustomResource"] = append(deleteOrderResourceMap["CustomResource"], r)
			}
		} else {
			if _, ok := deleteOrderResourceMap[r.GVK.Kind]; !ok {
				defaultOrder = append(defaultOrder, r.GVK.Kind)
				deleteOrderResourceMap[r.GVK.Kind] = []resource{}
			}
			deleteOrderResourceMap[r.GVK.Kind] = append(deleteOrderResourceMap[r.GVK.Kind], r)
		}
	}

	deleteOrderedKinds := getOrderedKinds(deleteKindOrder, defaultOrder)
	return deleteOrderResourceMap, deleteOrderedKinds
}

func shouldResourceWaitForDeletion(kind string, waitFlag bool) bool {
	if kind == "PersistentVolumeClaim" {
		// blocking on PVC delete will create a deadlock if
		// it's used by a pod that has not been deleted yet.
		return false
	}
	return waitFlag
}

func buildGVKKey(group, kind, version string) string {
	return fmt.Sprintf("%s/%s/%s", group, kind, version)
}

// check if resourcesToDeleteMap has any resources in it
func hasResources(resourcesMap map[string][]resource) bool {
	for _, resources := range resourcesMap {
		if len(resources) > 0 {
			return true
		}
	}
	return false
}

func clearNamespaces(appSlug string, namespacesToClear []string, isRestore bool, restoreLabelSelector labels.Selector, kindDeleteOrder KindOrder, k8sDynamicClient dynamic.Interface, gvrs map[schema.GroupVersionResource]struct{}) error {
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

	deletionGVRs := []schema.GroupVersionResource{}
	for gvr := range gvrs {
		s := fmt.Sprintf("%s/%s/%s", gvr.Group, gvr.Version, gvr.Resource)
		if !skip.Has(s) {
			deletionGVRs = append(deletionGVRs, gvr)
		}
	}

	for _, namespace := range namespacesToClear {
		logger.Infof("Ensuring all %s objects have been removed from namespace %s\n", appSlug, namespace)
		sleepTime := time.Second * 2

		for i := 60; i >= 0; i-- { // 2 minute wait, 60 loops with 2 second sleep
			resourcesToDeleteMap, deleteOrderedKinds := buildDeleteKindOrderedNamespaceResources(k8sDynamicClient, deletionGVRs, appSlug, namespace, isRestore, restoreLabelSelector, kindDeleteOrder)
			if !hasResources(resourcesToDeleteMap) {
				logger.Infof("Namespace %s successfully cleared of app %s\n", namespace, appSlug)
				break
			}

			err := clearNamespacedResources(k8sDynamicClient, namespace, resourcesToDeleteMap, deleteOrderedKinds)
			if err != nil {
				logger.Errorf("Failed to clear resources from app %s, namespace %s: %v\n", appSlug, namespace, err)
			}

			if i == 0 {
				return fmt.Errorf("Failed to clear app %s from namespace %s\n", appSlug, namespace)
			}
			logger.Infof("Namespace %s still has objects from app %s: sleeping...\n", namespace, appSlug)
			time.Sleep(sleepTime)
		}
	}

	if len(namespacesToClear) > 0 {
		// Extra time in case the app-slug annotation was not being used.
		time.Sleep(time.Second * 20)
	}

	return nil
}

func buildDeleteKindOrderedNamespaceResources(dyn dynamic.Interface, gvrs []schema.GroupVersionResource, appSlug string, namespace string, isRestore bool, restoreLabelSelector labels.Selector, deleteKindOrder KindOrder) (map[string][]resource, KindSortOrder) {
	deleteOrdererResourceMap := initResourceKindOrderMap(deleteKindOrder)
	var defaultKindOrder KindSortOrder
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
				labelMap := u.GetLabels()
				if excludeLabel, exists := labelMap["velero.io/exclude-from-backup"]; exists && excludeLabel == "true" {
					continue
				}

				labelSet := labels.Set(labelMap)
				if restoreLabelSelector != nil && !restoreLabelSelector.Matches(labelSet) {
					continue
				}
			}

			annotations := u.GetAnnotations()
			if annotations["kots.io/app-slug"] == appSlug {
				if u.GetDeletionTimestamp() != nil {
					logger.Infof("%s %s is pending deletion\n", gvr, u.GetName())
					continue
				}

				// copy the object so we don't modify the cache
				uCopy := u.DeepCopy()
				gvk := uCopy.GetObjectKind().GroupVersionKind()
				r := resource{
					Unstructured: u.DeepCopy(),
					GVK:          &gvk,
					GVR:          gvr,
				}
				if _, ok := deleteOrdererResourceMap[u.GetKind()]; !ok {
					defaultKindOrder = append(defaultKindOrder, u.GetKind())
					deleteOrdererResourceMap[u.GetKind()] = []resource{}
				}
				deleteOrdererResourceMap[u.GetKind()] = append(deleteOrdererResourceMap[u.GetKind()], r)
			}
		}
	}

	deleteOrderedKinds := getOrderedKinds(deleteKindOrder, defaultKindOrder)
	return deleteOrdererResourceMap, deleteOrderedKinds
}

func clearNamespacedResources(dyn dynamic.Interface, namespace string, resourcesMap map[string][]resource, deleteKindOrders KindSortOrder) error {
	for _, kind := range deleteKindOrders {
		for _, r := range resourcesMap[kind] {
			u := r.Unstructured
			logger.Infof("Deleting %s/%s/%s\n", namespace, r.GVR, u.GetName())
			err := dyn.Resource(r.GVR).Namespace(namespace).Delete(context.TODO(), u.GetName(), metav1.DeleteOptions{})
			if err != nil {
				logger.Errorf("Resource %s (%s) in namespace %s could not be deleted: %v\n", u.GetName(), r.GVR, namespace, err)
				return err
			}
		}
	}
	return nil
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
