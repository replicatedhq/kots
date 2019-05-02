package generator

import (
	"os"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	shipv1beta1 "github.com/replicatedhq/ship-operator/pkg/apis/ship/v1beta1"
	"github.com/replicatedhq/ship-operator/pkg/logger"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Generator struct {
	instance *shipv1beta1.ShipWatch
	logger   log.Logger
}

func NewGenerator(instance *shipv1beta1.ShipWatch) *Generator {
	generator := &Generator{
		instance: instance,
		logger:   logger.FromEnv(),
	}

	return generator
}

// generate nodeSelector for pods from flag like --node-selector replicated/node-pool=untrusted
func (g *Generator) nodeSelector() map[string]string {
	// TODO use viper
	nodeSelector := os.Getenv("NODE_SELECTOR")
	if nodeSelector == "" {
		return nil
	}
	parts := strings.Split(nodeSelector, "=")
	if len(parts) != 2 {
		level.Error(log.With(g.logger, "event", "parse.node-selector", "value", nodeSelector))
		return nil
	}
	level.Debug(log.With(g.logger, "event", "generate.nodeSelector", "key", parts[0], "value", parts[1]))
	return map[string]string{parts[0]: parts[1]}
}

func (g *Generator) getTerminationGracePeriodSeconds() *int64 {
	terminationGracePeriodSeconds := int64(0)
	return &terminationGracePeriodSeconds
}

func (g *Generator) WatchJob(stateSHA string) *batchv1.Job {
	level.Debug(log.With(g.logger)).Log("method", "generateWatchJob")

	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      g.instance.Name + "-watch",
			Namespace: g.instance.Namespace,
			Annotations: map[string]string{
				"ship.replicated.com/state-sha": stateSHA,
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector:       g.nodeSelector(),
					RestartPolicy:      corev1.RestartPolicyOnFailure, // This is important, we'll monitor this pod for success
					ServiceAccountName: g.instance.Spec.State.ValueFrom.SecretKeyRef.ServiceAccountName,
					Containers: []corev1.Container{
						g.generateWatchContainer(),
					},
					TerminationGracePeriodSeconds: g.getTerminationGracePeriodSeconds(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"ship-type": "watch"},
				},
			},
		},
	}

	trueValue := true
	instanceOwner := metav1.OwnerReference{
		APIVersion: "ship.replicated.com/v1beta1",
		Kind:       "ShipWatch",
		Name:       g.instance.Name,
		UID:        g.instance.GetUID(),
		Controller: &trueValue,
	}
	job.SetOwnerReferences([]metav1.OwnerReference{instanceOwner})

	return job
}

func (g *Generator) UpdateJob(stateSHA string) *batchv1.Job {
	debug := level.Debug(log.With(g.logger, "method", "generateUpdateJob"))

	job := &batchv1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "batch/v1",
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      g.instance.Name + "-update",
			Namespace: g.instance.Namespace,
			Annotations: map[string]string{
				"ship.replicated.com/state-sha": stateSHA,
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeSelector:       g.nodeSelector(),
					RestartPolicy:      corev1.RestartPolicyOnFailure, // This is important, we'll monitor this pod for success
					ServiceAccountName: g.instance.Spec.State.ValueFrom.SecretKeyRef.ServiceAccountName,
					InitContainers: []corev1.Container{
						g.generateUpdateContainer(),
					},
					Volumes: []corev1.Volume{
						{
							Name: "shared",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
					},
					TerminationGracePeriodSeconds: g.getTerminationGracePeriodSeconds(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"ship-type": "update"},
				},
			},
		},
	}

	debug.Log("event", "generating action containers")
	for _, action := range g.instance.Spec.Actions {
		if action.PullRequest != nil {
			container, volumes := g.generatePullRequestContainer(action.PullRequest)
			job.Spec.Template.Spec.Containers = append(job.Spec.Template.Spec.Containers, container)
			job.Spec.Template.Spec.Volumes = append(job.Spec.Template.Spec.Volumes, volumes...)
		} else if action.Webhook != nil {
			container := g.generateWebhookContainer(action.Webhook)
			job.Spec.Template.Spec.Containers = append(job.Spec.Template.Spec.Containers, container)
		}
	}

	trueValue := true
	instanceOwner := metav1.OwnerReference{
		APIVersion: "ship.replicated.com/v1beta1",
		Kind:       "ShipWatch",
		Name:       g.instance.Name,
		UID:        g.instance.GetUID(),
		Controller: &trueValue,
	}
	job.SetOwnerReferences([]metav1.OwnerReference{instanceOwner})

	return job
}
