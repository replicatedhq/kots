package types

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type AllResourceRequirements struct {
	Kotsadm  *ResourceRequirements
	Minio    *ResourceRequirements
	Postgres *ResourceRequirements
	Dex      *ResourceRequirements
}

type ResourceRequirements struct {
	PodCpuRequest         resource.Quantity
	PodCpuRequestIsSet    bool
	PodMemoryRequest      resource.Quantity
	PodMemoryRequestIsSet bool
	PodCpuLimit           resource.Quantity
	PodCpuLimitIsSet      bool
	PodMemoryLimit        resource.Quantity
	PodMemoryLimitIsSet   bool
}

func (r *ResourceRequirements) ToCoreV1ResourceRequirements() corev1.ResourceRequirements {
	return r.UpdateCoreV1ResourceRequirements(corev1.ResourceRequirements{})
}

func (r *ResourceRequirements) UpdateCoreV1ResourceRequirements(resources corev1.ResourceRequirements) corev1.ResourceRequirements {
	if r == nil {
		return resources
	}
	if r.PodCpuLimitIsSet {
		if resources.Limits == nil {
			resources.Limits = corev1.ResourceList{}
		}
		resources.Limits["cpu"] = r.PodCpuLimit
	}
	if r.PodMemoryLimitIsSet {
		if resources.Limits == nil {
			resources.Limits = corev1.ResourceList{}
		}
		resources.Limits["memory"] = r.PodMemoryLimit
	}
	if r.PodMemoryRequestIsSet {
		if resources.Requests == nil {
			resources.Requests = corev1.ResourceList{}
		}
		resources.Requests["memory"] = r.PodMemoryRequest
	}
	if r.PodCpuRequestIsSet {
		if resources.Requests == nil {
			resources.Requests = corev1.ResourceList{}
		}
		resources.Requests["cpu"] = r.PodCpuRequest
	}
	return resources
}
