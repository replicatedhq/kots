package midstream

import (
	"sigs.k8s.io/kustomize/v3/pkg/image"
	kustomizetypes "sigs.k8s.io/kustomize/v3/pkg/types"
)

func removeExistingImages(new []image.Image, existing []image.Image) []image.Image {
	filteredImages := make([]image.Image, 0)
	names := make(map[string]bool)

	for _, n := range new {
		names[n.Name] = true
	}

	for _, e := range existing {
		if _, exists := names[e.Name]; !exists {
			names[e.Name] = true
			filteredImages = append(filteredImages, e)
		}
	}

	return filteredImages
}

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
