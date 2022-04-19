package appstate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/replicatedhq/kots/pkg/appstate/types"
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

// different informers can be watching the same object but using a different api version (e.g. ingress with api version networking.k8s.io/v1 or extensions/v1beta1).
// depending on the version of the kubernetes cluster, one of those api versions / objects could be missing or unsupported.
// this function handles that by getting the maximum state of the same object reported by the informers.
func reduceResourceStates(resourceStates types.ResourceStates, resourceState types.ResourceState) (next types.ResourceStates) {
	m := map[string]types.State{}

	// existing resource states
	for _, r := range resourceStates {
		key := fmt.Sprintf("%s/%s/%s", r.Namespace, r.Kind, r.Name)
		m[key] = types.MaxState(m[key], r.State)
	}

	// new resource state
	key := fmt.Sprintf("%s/%s/%s", resourceState.Namespace, resourceState.Kind, resourceState.Name)
	m[key] = types.MaxState(m[key], resourceState.State)

	// convert back to resource states
	for k, v := range m {
		parts := strings.Split(k, "/")
		next = append(next, types.ResourceState{
			Namespace: parts[0],
			Kind:      parts[1],
			Name:      parts[2],
			State:     v,
		})
	}

	sort.Sort(next)
	return
}
