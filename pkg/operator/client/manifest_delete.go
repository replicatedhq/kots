package client

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/operator/applier"
	"github.com/replicatedhq/kots/pkg/operator/types"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
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
	resources = resources.ApplyPlan(plan)
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

		unstructured := &unstructured.Unstructured{}
		_, gvk, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(manifest), nil, unstructured)
		if err != nil {
			logger.Infof("error decoding manifest: %v", err.Error())
		} else {
			resource.Unstructured = unstructured
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

func clearNamespaces(appSlug string, namespacesToClear []string, isRestore bool, restoreLabelSelector *metav1.LabelSelector, plan types.Plan) error {
	dyn, err := k8sutil.GetDynamicClient()
	if err != nil {
		return errors.Wrap(err, "failed to get dynamic client")
	}

	config, err := k8sutil.GetClusterConfig()
	if err != nil {
		return errors.Wrap(err, "failed to get cluster config")
	}

	disc, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return errors.Wrap(err, "failed to create discovery client")
	}

	resourceList, err := disc.ServerPreferredNamespacedResources()
	if err != nil {
		// An application can define an APIService handled by a Deployment in the application itself.
		// In that case there will be a race condition listing resources in that API group and there
		// could be an error here. Most of the API groups would still be in the resource list so the
		// error is not terminal.
		logger.Infof("Failed to list all resources: %v", err)
	}

	gvrs, err := discovery.GroupVersionResources(resourceList)
	if err != nil {
		return errors.Wrap(err, "failed to get group version resources")
	}

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

	listResourcesToDelete := func(namespace string) (types.Resources, error) {
		resources := types.Resources{}

		for _, gvr := range gvrsToDelete {
			// there may be other resources that can't be
			// listed besides what's in the skip set so ignore error
			unstructuredList, err := dyn.Resource(gvr).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
			if unstructuredList == nil {
				if err != nil {
					logger.Errorf("failed to list namespace resources: %s", err.Error())
				}
				continue
			}

			for _, u := range unstructuredList.Items {
				// if this is a restore process, only delete resources that are part of the backup and will be restored
				// e.g. resources that do not have the exclude label, and match the restore/backup label selector
				if isRestore {
					itemLabelsMap := u.GetLabels()
					if excludeLabel, exists := itemLabelsMap["velero.io/exclude-from-backup"]; exists && excludeLabel == "true" {
						continue
					}
					if restoreLabelSelector != nil {
						s, err := metav1.LabelSelectorAsSelector(restoreLabelSelector)
						if err != nil {
							return nil, errors.Wrap(err, "failed to convert label selector to a selector")
						}
						if !s.Matches(labels.Set(itemLabelsMap)) {
							continue
						}
					}
				}

				if u.GetAnnotations()["kots.io/app-slug"] == appSlug {
					unstructured := u.DeepCopy()
					gvk := unstructured.GetObjectKind().GroupVersionKind()
					resource := types.Resource{
						Unstructured: unstructured,
						GVK:          &gvk,
						GVR:          gvr,
					}
					resources = append(resources, resource)
				}
			}
		}

		return resources, nil
	}

	sleepTime := time.Second * 2
	for i := 60; i >= 0; i-- { // 2 minute wait, 60 loops with 2 second sleep
		resourcesToDelete := types.Resources{}

		for _, namespace := range namespacesToClear {
			rs, err := listResourcesToDelete(namespace)
			if err != nil {
				return errors.Wrapf(err, "failed to list resources to delete in namespace %s", namespace)
			}
			resourcesToDelete = append(resourcesToDelete, rs...)
		}

		if len(resourcesToDelete) == 0 {
			logger.Infof("Successfully cleared all resources for app %s\n", appSlug)
			return nil
		}

		resourcesToDelete = resourcesToDelete.ApplyPlan(plan)

		for _, r := range resourcesToDelete {
			if r.Unstructured.GetDeletionTimestamp() != nil {
				logger.Infof("Pending deletion %s/%s/%s\n", r.Unstructured.GetNamespace(), r.GVR, r.Unstructured.GetName())
				continue
			}

			logger.Infof("Deleting %s/%s/%s\n", r.Unstructured.GetNamespace(), r.GVR, r.Unstructured.GetName())

			if err := dyn.Resource(r.GVR).Namespace(r.Unstructured.GetNamespace()).Delete(context.TODO(), r.Unstructured.GetName(), metav1.DeleteOptions{}); err != nil {
				logger.Errorf("Resource %s (%s) in namespace %s could not be deleted: %v\n", r.Unstructured.GetName(), r.GVR, r.Unstructured.GetNamespace(), err)
			}
		}

		time.Sleep(sleepTime)
	}

	return nil
}

func deletePVCs(namespace string, appLabelSelector *metav1.LabelSelector, appslug string) error {
	clientset, err := k8sutil.GetClientset()
	if err != nil {
		return errors.Wrap(err, "failed to get k8s clientset")
	}

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
