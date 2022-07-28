package k8sutil

import (
	corev1 "k8s.io/api/core/v1"
)

// copy the env vars from the desired to existing. this could undo a change that the user had.
// we don't know which env vars we set and which are user edited. this method avoids deleting
// env vars that the user added, but doesn't handle edited vars
func MergeEnvVars(desired []corev1.EnvVar, existing []corev1.EnvVar) []corev1.EnvVar {
	mergedEnvs := []corev1.EnvVar{}
	mergedEnvs = append(mergedEnvs, desired...)
	for _, e := range existing {
		exists := false
		for _, env := range desired {
			if env.Name == e.Name {
				exists = true
			}
		}
		if !exists {
			mergedEnvs = append(mergedEnvs, e)
		}
	}
	return mergedEnvs
}

func MergeInitContainers(desired []corev1.Container, existing []corev1.Container) []corev1.Container {
	additional := []corev1.Container{}
	for _, desiredContainer := range desired {
		found := false
		for i, existingContainer := range existing {
			if existingContainer.Name != desiredContainer.Name {
				continue
			}
			existing[i] = *desiredContainer.DeepCopy()
			found = true
			break
		}
		if !found {
			additional = append(additional, *desiredContainer.DeepCopy())
		}
	}
	return append(existing, additional...)
}
