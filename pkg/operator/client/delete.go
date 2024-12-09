package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/k8sutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/operator/applier"
	"github.com/replicatedhq/kots/pkg/operator/types"
	"github.com/replicatedhq/kots/pkg/util"
	kuberneteserrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/discovery"
)

type DiffAndDeleteOptions struct {
	PreviousManifests    string
	CurrentManifests     string
	AdditionalNamespaces []string
	IsRestore            bool
	RestoreLabelSelector *metav1.LabelSelector
	Wait                 bool
}

func (c *Client) diffAndDeleteManifests(opts DiffAndDeleteOptions) error {
	decodedPrevious, err := base64.StdEncoding.DecodeString(opts.PreviousManifests)
	if err != nil {
		return errors.Wrap(err, "failed to base64 decode previous manifests")
	}

	decodedCurrent, err := base64.StdEncoding.DecodeString(opts.CurrentManifests)
	if err != nil {
		return errors.Wrap(err, "failed to base64 decode manifests")
	}

	// we need to find the gvk+names that are present in the previous, but not in the current and then remove them
	// namespaces that were removed from YAML and added to additionalNamespaces should not be removed
	decodedPreviousDocs := util.ConvertToSingleDocs(decodedPrevious)

	type previousObject struct {
		spec   string
		delete bool
	}
	decodedPreviousMap := map[string]previousObject{}
	for _, decodedPreviousDoc := range decodedPreviousDocs {
		k, o := GetGVKWithNameAndNs(decodedPreviousDoc, c.TargetNamespace)

		delete := true
		if o.APIVersion == "v1" && o.Kind == "Namespace" {
			for _, n := range opts.AdditionalNamespaces {
				if o.Metadata.Name == n {
					delete = false
					break
				}
			}
		}

		// if this is a restore process, only delete resources that are part of the backup and will be restored
		// e.g. resources that do not have the exclude label, and match the restore/backup label selector
		if opts.IsRestore {
			if excludeLabel, exists := o.Metadata.Labels["velero.io/exclude-from-backup"]; exists && excludeLabel == "true" {
				delete = false
			}
			if opts.RestoreLabelSelector != nil {
				s, err := metav1.LabelSelectorAsSelector(opts.RestoreLabelSelector)
				if err != nil {
					return errors.Wrap(err, "failed to convert label selector to a selector")
				}
				if !s.Matches(labels.Set(o.Metadata.Labels)) {
					delete = false
				}
			}
		}

		// if this is a keep resource, don't delete it
		// e.g. for migration to Helm v1beta2
		if keep, ok := o.Metadata.Annotations["kots.io/keep"]; ok && keep == "true" {
			logger.Infof("Skipping deletion of resource %s/%s", o.Kind, o.Metadata.Name)
			delete = false
		}

		decodedPreviousMap[k] = previousObject{
			spec:   string(decodedPreviousDoc),
			delete: delete,
		}
	}

	// now get the current names
	decodedCurrentDocs := util.ConvertToSingleDocs(decodedCurrent)
	decodedCurrentMap := map[string]string{}
	for _, decodedCurrentDoc := range decodedCurrentDocs {
		k, _ := GetGVKWithNameAndNs(decodedCurrentDoc, c.TargetNamespace)
		decodedCurrentMap[k] = string(decodedCurrentDoc)
	}

	// this is pretty raw, and required kubectl...  we should
	// consider some other options here?
	kubernetesApplier, err := c.getApplier()
	if err != nil {
		return errors.Wrap(err, "failed to get applier")
	}

	// now remove anything that's in previous but not in current
	manifestsToDelete := [][]byte{}
	for k, previous := range decodedPreviousMap {
		if _, ok := decodedCurrentMap[k]; ok {
			continue
		}
		if !previous.delete {
			continue
		}
		manifestsToDelete = append(manifestsToDelete, []byte(previous.spec))
	}

	// TODO: return error here?
	c.deleteManifests(manifestsToDelete, kubernetesApplier, opts.Wait)

	return nil
}

func (c *Client) deleteManifests(manifests [][]byte, kubernetesApplier applier.KubectlInterface, waitFlag bool) {
	resources := decodeManifests(manifests)
	c.deleteResources(resources, kubernetesApplier, waitFlag)
}

func (c *Client) deleteResources(resources types.Resources, kubernetesApplier applier.KubectlInterface, waitFlag bool) {
	phases := groupAndSortResourcesForDeletion(resources)
	for _, phase := range phases {
		logger.Infof("deleting resources in phase %s", phase.Name)
		for _, r := range phase.Resources {
			c.deleteResource(r, waitFlag, kubernetesApplier)
		}
	}
}

func (c *Client) deleteResource(resource types.Resource, waitFlag bool, kubernetesApplier applier.KubectlInterface) {
	group := resource.GetGroup()
	version := resource.GetVersion()
	kind := resource.GetKind()
	name := resource.GetName()
	wait := shouldWaitForResourceDeletion(kind, waitFlag)

	namespace := resource.GetNamespace()
	if namespace == "" {
		namespace = c.TargetNamespace
	}

	if resource.DecodeErrMsg == "" {
		logger.Infof("deleting resource %s/%s/%s/%s from namespace %s", group, version, kind, name, namespace)
	} else {
		logger.Infof("deleting unidentified resource. unable to parse error: %s", resource.DecodeErrMsg)
	}

	stdout, stderr, err := kubernetesApplier.Remove(namespace, []byte(resource.Manifest), wait)
	if err != nil {
		logger.Infof("stdout (delete) = %s", stdout)
		logger.Infof("stderr (delete) = %s", stderr)
		logger.Infof("error: %s", err.Error())
	} else {
		if resource.DecodeErrMsg == "" {
			logger.Infof("deleted resource %s/%s/%s/%s from namespace %s", group, version, kind, name, namespace)
		} else {
			logger.Info("deleted unidentified resource")
		}
	}
}

func shouldWaitForResourceDeletion(kind string, waitFlag bool) bool {
	if kind == "PersistentVolumeClaim" {
		// blocking on PVC delete will create a deadlock if
		// it's used by a pod that has not been deleted yet.
		return false
	}
	return waitFlag
}

func (c *Client) clearNamespaces(appSlug string, namespacesToClear []string, isRestore bool, restoreLabelSelector *metav1.LabelSelector) error {
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
	for i := 60; i >= 0; i-- { // 2 minute wait
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

		phases := groupAndSortResourcesForDeletion(resourcesToDelete)

		for _, phase := range phases {
			logger.Infof("Deleting resources in phase %s", phase.Name)
			for _, r := range phase.Resources {
				if r.Unstructured.GetDeletionTimestamp() != nil {
					logger.Infof("Pending deletion %s/%s/%s/%s/%s", r.Unstructured.GetNamespace(), r.GVR.Group, r.GVR.Version, r.GVR.Resource, r.Unstructured.GetName())
					continue
				}

				logger.Infof("Deleting %s/%s/%s/%s/%s", r.Unstructured.GetNamespace(), r.GVR.Group, r.GVR.Version, r.GVR.Resource, r.Unstructured.GetName())

				if err := dyn.Resource(r.GVR).Namespace(r.Unstructured.GetNamespace()).Delete(context.TODO(), r.Unstructured.GetName(), metav1.DeleteOptions{}); err != nil {
					logger.Errorf("Resource %s/%s/%s/%s/%s could not be deleted: %v", r.Unstructured.GetNamespace(), r.GVR.Group, r.GVR.Version, r.GVR.Resource, r.Unstructured.GetName(), err)
				}
			}
		}

		time.Sleep(sleepTime)
	}

	return fmt.Errorf("failed to clear all resources for app %s", appSlug)
}

func (c *Client) deletePVCs(appLabelSelector *metav1.LabelSelector, appslug string) error {
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

	appSelector, err := metav1.LabelSelectorAsSelector(appLabelSelector)
	if err != nil {
		return errors.Wrap(err, "failed to convert label selector to a selector")
	}

	podsList, err := clientset.CoreV1().Pods(c.TargetNamespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: appSelector.String(),
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
		logger.Infof("no pvcs to delete in %s for pods that match %s", c.TargetNamespace, appSelector.String())
		return nil
	}
	logger.Infof("deleting %d pvcs in %s for pods that match %s", len(pvcs), c.TargetNamespace, appSelector.String())

	for _, pvc := range pvcs {
		grace := int64(0)
		policy := metav1.DeletePropagationBackground
		opts := metav1.DeleteOptions{
			GracePeriodSeconds: &grace,
			PropagationPolicy:  &policy,
		}
		logger.Infof("deleting pvc: %s", pvc)
		err := clientset.CoreV1().PersistentVolumeClaims(c.TargetNamespace).Delete(context.TODO(), pvc, opts)
		if err != nil && !kuberneteserrors.IsNotFound(err) {
			return errors.Wrapf(err, "failed to delete pvc %s", pvc)
		}
	}

	return nil
}
