package client

import (
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/operator/applier"
	"github.com/replicatedhq/kots/pkg/operator/types"
)

func createManifests(manifests []string, targetNS string, kubernetesApplier applier.KubectlInterface, waitFlag bool) {
	resources := decodeManifests(manifests)
	createResources(resources, targetNS, kubernetesApplier, waitFlag)
}

func createResources(resources types.Resources, targetNS string, kubernetesApplier applier.KubectlInterface, waitFlag bool) {
	resources = sortResourcesForCreation(resources)
	for _, r := range resources {
		createResource(r, targetNS, waitFlag, kubernetesApplier)
	}
}

func createResource(resource types.Resource, targetNS string, waitFlag bool, kubernetesApplier applier.KubectlInterface) {
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
