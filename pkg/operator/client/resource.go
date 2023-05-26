package client

import (
	"sort"
	"strconv"

	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/operator/types"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/kubernetes/scheme"
)

var (
	// These lists are inspired by Helm: https://github.com/helm/helm/blob/v3.11.3/pkg/releaseutil/kind_sorter.go
	// Unknown kinds are created last.
	KindCreationOrder = []string{
		"Namespace",
		"NetworkPolicy",
		"ResourceQuota",
		"LimitRange",
		"PodSecurityPolicy",
		"PodDisruptionBudget",
		"ServiceAccount",
		"Secret",
		"SecretList",
		"ConfigMap",
		"StorageClass",
		"PersistentVolume",
		"PersistentVolumeClaim",
		"CustomResourceDefinition",
		"ClusterRole",
		"ClusterRoleList",
		"ClusterRoleBinding",
		"ClusterRoleBindingList",
		"Role",
		"RoleList",
		"RoleBinding",
		"RoleBindingList",
		"Service",
		"DaemonSet",
		"Pod",
		"ReplicationController",
		"ReplicaSet",
		"Deployment",
		"HorizontalPodAutoscaler",
		"StatefulSet",
		"Job",
		"CronJob",
		"IngressClass",
		"Ingress",
		"APIService",
	}
	// Unknown kinds are deleted first.
	KindDeletionOrder = []string{
		"APIService",
		"Ingress",
		"IngressClass",
		"Service",
		"CronJob",
		"Job",
		"StatefulSet",
		"HorizontalPodAutoscaler",
		"Deployment",
		"ReplicaSet",
		"ReplicationController",
		"Pod",
		"DaemonSet",
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
		"StorageClass",
		"ConfigMap",
		"SecretList",
		"Secret",
		"ServiceAccount",
		"PodDisruptionBudget",
		"PodSecurityPolicy",
		"LimitRange",
		"ResourceQuota",
		"NetworkPolicy",
		"Namespace",
	}
)

func decodeManifests(manifests []string) types.Resources {
	resources := types.Resources{}

	for _, manifest := range manifests {
		resource := types.Resource{
			Manifest: manifest,
		}

		unstructured := &unstructured.Unstructured{}
		_, gvk, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(manifest), nil, unstructured)
		if err != nil {
			logger.Infof("error decoding manifest: %v", err.Error())
			resource.DecodeErrMsg = err.Error()
		} else {
			resource.Unstructured = unstructured
			resource.GVK = gvk
		}

		resources = append(resources, resource)
	}

	return resources
}

// groupAndSortResourcesForCreation sorts resources by phase and then by kind based on the kind creation order.
// unknown kinds are created last.
func groupAndSortResourcesForCreation(resources types.Resources) types.Phases {
	resourcesByPhase := resources.GroupByPhaseAnnotation(types.CreationPhaseAnnotation)

	sortedPhases := getSortedPhases(resourcesByPhase)

	phases := types.Phases{}
	for _, name := range sortedPhases {
		sortedResources := types.Resources{}

		creationOrder := KindCreationOrder
		resourcesByKind := resourcesByPhase[name].GroupByKind()

		for kind := range resourcesByKind {
			unknown := true
			for _, creationKind := range creationOrder {
				if kind == creationKind {
					unknown = false
					break
				}
			}
			if unknown {
				// unknown kinds are create last
				creationOrder = append(creationOrder, kind)
			}
		}

		for _, kind := range creationOrder {
			sortedResources = append(sortedResources, resourcesByKind[kind]...)
		}

		phase := types.Phase{
			Name:      name,
			Resources: sortedResources,
		}
		phases = append(phases, phase)
	}

	return phases
}

// groupAndSortResourcesForDeletion sorts resources by phase and then by kind based on the kind deletion order.
// unknown kinds are deleted first.
func groupAndSortResourcesForDeletion(resources types.Resources) types.Phases {
	resourcesByPhase := resources.GroupByPhaseAnnotation(types.DeletionPhaseAnnotation)

	sortedPhases := getSortedPhases(resourcesByPhase)

	phases := types.Phases{}
	for _, name := range sortedPhases {
		sortedResources := types.Resources{}

		deletionOrder := KindDeletionOrder
		resourcesByKind := resourcesByPhase[name].GroupByKind()

		for kind := range resourcesByKind {
			unknown := true
			for _, deletionKind := range deletionOrder {
				if kind == deletionKind {
					unknown = false
					break
				}
			}
			if unknown {
				// unknown kinds are deleted first
				deletionOrder = append([]string{kind}, deletionOrder...)
			}
		}

		for _, kind := range deletionOrder {
			sortedResources = append(sortedResources, resourcesByKind[kind]...)
		}

		phase := types.Phase{
			Name:      name,
			Resources: sortedResources,
		}
		phases = append(phases, phase)
	}

	return phases
}

func getSortedPhases(resourcesByPhase map[string]types.Resources) []string {
	sortedPhases := []string{}
	for phase := range resourcesByPhase {
		sortedPhases = append(sortedPhases, phase)
	}

	sort.Slice(sortedPhases, func(i, j int) bool {
		iInt, err := strconv.ParseInt(sortedPhases[i], 10, 64)
		if err != nil {
			iInt = 0
		}
		jInt, err := strconv.ParseInt(sortedPhases[j], 10, 64)
		if err != nil {
			jInt = 0
		}
		return iInt < jInt
	})

	return sortedPhases
}
