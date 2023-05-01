package client

import (
	"sort"

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

// sortResourcesForCreation groups and sorts resources by creation weight first,
// and resources that have the same creation weight are then sorted by kind based on the kind creation order.
// for each weight group, unknown kinds are created last.
func sortResourcesForCreation(resources types.Resources) types.Resources {
	sortedResources := types.Resources{}
	resourcesByWeight := resources.GroupByCreationWeight()

	weights := []string{}
	for weight := range resourcesByWeight {
		weights = append(weights, weight)
	}
	sort.Strings(weights)

	for _, weight := range weights {
		creationOrder := KindCreationOrder
		resourcesByKind := resourcesByWeight[weight].GroupByKind()

		for kind := range resourcesByKind {
			unknown := true
			for _, deletionKind := range creationOrder {
				if kind == deletionKind {
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
	}

	return sortedResources
}

// sortResourcesForDeletion groups and sorts resources by deletion weight first,
// and resources that have the same deletion weight are then sorted by kind based on the kind deletion order.
// for each weight group, unknown kinds are deleted first.
func sortResourcesForDeletion(resources types.Resources) types.Resources {
	sortedResources := types.Resources{}
	resourcesByWeight := resources.GroupByDeletionWeight()

	weights := []string{}
	for weight := range resourcesByWeight {
		weights = append(weights, weight)
	}
	sort.Strings(weights)

	for _, weight := range weights {
		deletionOrder := KindDeletionOrder
		resourcesByKind := resourcesByWeight[weight].GroupByKind()

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
	}

	return sortedResources
}
