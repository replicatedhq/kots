package midstream

import (
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

func findNewPatches(new []kustomizetypes.PatchStrategicMerge, existing []kustomizetypes.PatchStrategicMerge) []kustomizetypes.PatchStrategicMerge {
	newPatches := make([]kustomizetypes.PatchStrategicMerge, 0)
	names := make(map[string]bool)

	for _, e := range existing {
		names[string(e)] = true
	}

	for _, n := range new {
		if _, exists := names[string(n)]; !exists {
			names[string(n)] = true
			newPatches = append(newPatches, n)
		}
	}

	return newPatches
}

func findNewStrings(new []string, existing []string) []string {
	newStrings := make([]string, 0)
	names := make(map[string]bool)

	for _, e := range existing {
		names[e] = true
	}

	for _, n := range new {
		if _, exists := names[n]; !exists {
			names[n] = true
			newStrings = append(newStrings, n)
		}
	}

	return newStrings
}
