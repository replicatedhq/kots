package midstream

import (
	kustomizetypes "sigs.k8s.io/kustomize/api/types"
)

func uniquePatches(existing ...[]kustomizetypes.PatchStrategicMerge) []kustomizetypes.PatchStrategicMerge {
	newPatches := make([]kustomizetypes.PatchStrategicMerge, 0)
	names := make(map[string]bool)

	for _, ee := range existing {
		for _, e := range ee {
			if _, exists := names[string(e)]; !exists {
				names[string(e)] = true
				newPatches = append(newPatches, e)
			}
		}
	}

	return newPatches
}

func uniqueStrings(existing ...[]string) []string {
	newStrings := make([]string, 0)
	names := make(map[string]bool)

	for _, ee := range existing {
		for _, e := range ee {
			if _, exists := names[e]; !exists {
				names[e] = true
				newStrings = append(newStrings, e)
			}
		}
	}

	return newStrings
}
