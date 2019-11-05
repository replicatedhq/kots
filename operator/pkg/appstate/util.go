package appstate

import (
	"sort"

	"github.com/replicatedhq/kotsadm/operator/pkg/appstate/types"
)

func normalizeStatusInformers(informers []types.StatusInformer, targetNamespace string) (next []types.StatusInformer) {
	for _, informer := range informers {
		informer.Kind = getResourceKindCommonName(informer.Kind)
		if informer.Namespace == "" {
			informer.Namespace = targetNamespace
		}
		next = append(next, informer)
	}
	return
}

func filterStatusInformersByResourceKind(informers []types.StatusInformer, kind string) (next []types.StatusInformer) {
	for _, informer := range informers {
		if informer.Kind == kind {
			next = append(next, informer)
		}
	}
	return
}

func buildResourceStatesFromStatusInformers(informers []types.StatusInformer) types.ResourceStates {
	next := types.ResourceStates{}
	for _, informer := range informers {
		next = append(next, types.ResourceState{
			Kind:      informer.Kind,
			Name:      informer.Name,
			Namespace: informer.Namespace,
			State:     types.StateMissing,
		})
	}
	sort.Sort(next)
	return next
}

func resourceStatesApplyNew(resourceStates types.ResourceStates, informers []types.StatusInformer, resourceState types.ResourceState) (next types.ResourceStates, didChange bool) {
	for _, r := range resourceStates {
		if resourceState.Kind == r.Kind &&
			resourceState.Namespace == r.Namespace &&
			resourceState.Name == r.Name &&
			resourceState.State != r.State {
			didChange = true
			next = append(next, resourceState)
		} else {
			next = append(next, r)
		}
	}
	sort.Sort(next)
	return
}
