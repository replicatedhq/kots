package k8sutil

import (
	corev1 "k8s.io/api/core/v1"
)

func MergeEnvVars(desired []corev1.EnvVar, existing []corev1.EnvVar, override bool) []corev1.EnvVar {
	mergedEnvVars := []corev1.EnvVar{}
	mergedEnvVars = append(mergedEnvVars, existing...)
	for _, desiredEnvVar := range desired {
		idx := -1
		for existingEnvVarIndex, existingEnvVar := range existing {
			if existingEnvVar.Name != desiredEnvVar.Name {
				continue
			}
			idx = existingEnvVarIndex
			break
		}
		if idx == -1 {
			// not found, add it
			mergedEnvVars = append(mergedEnvVars, *desiredEnvVar.DeepCopy())
		} else if override {
			// found and should override
			mergedEnvVars[idx] = *desiredEnvVar.DeepCopy()
		}
	}
	return mergedEnvVars
}

func MergeVolumes(desired []corev1.Volume, existing []corev1.Volume, override bool) []corev1.Volume {
	mergedVolumes := []corev1.Volume{}
	mergedVolumes = append(mergedVolumes, existing...)
	for _, desiredVolume := range desired {
		idx := -1
		for existingVolumeIndex, existingVolume := range existing {
			if existingVolume.Name != desiredVolume.Name {
				continue
			}
			idx = existingVolumeIndex
			break
		}
		if idx == -1 {
			// not found, add it
			mergedVolumes = append(mergedVolumes, *desiredVolume.DeepCopy())
		} else if override {
			// found and should override
			mergedVolumes[idx] = *desiredVolume.DeepCopy()
		}
	}
	return mergedVolumes
}

func MergeVolumeMounts(desired []corev1.VolumeMount, existing []corev1.VolumeMount, override bool) []corev1.VolumeMount {
	mergedVolumeMounts := []corev1.VolumeMount{}
	mergedVolumeMounts = append(mergedVolumeMounts, existing...)
	for _, desiredVolumeMount := range desired {
		idx := -1
		for existingVolumeMountIndex, existingVolumeMount := range existing {
			if existingVolumeMount.Name != desiredVolumeMount.Name {
				continue
			}
			idx = existingVolumeMountIndex
			break
		}
		if idx == -1 {
			// not found, add it
			mergedVolumeMounts = append(mergedVolumeMounts, *desiredVolumeMount.DeepCopy())
		} else if override {
			// found and should override
			mergedVolumeMounts[idx] = *desiredVolumeMount.DeepCopy()
		}
	}
	return mergedVolumeMounts
}
