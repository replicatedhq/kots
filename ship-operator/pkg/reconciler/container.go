package reconciler

import (
	"fmt"
	"reflect"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	corev1 "k8s.io/api/core/v1"
)

func (r *Reconciler) shouldUpdateContainer(found, desired corev1.Container) bool {
	debug := level.Debug(log.With(r.logger, "method", "Reconciler.shouldUpdateContainer", "name", found.Name))

	if found.Name != desired.Name {
		debug.Log("event", "name.changed", "old", found.Name, "new", desired.Name)
		return true
	}
	if found.Image != desired.Image {
		debug.Log("event", "image.changed", "old", found.Image, "new", desired.Image)
		return true
	}
	if found.ImagePullPolicy != desired.ImagePullPolicy {
		debug.Log("event", "imagePullPolicy.changed", "old", found.ImagePullPolicy, "new", desired.ImagePullPolicy)
		return true
	}
	if !reflect.DeepEqual(found.Command, desired.Command) {
		debug.Log("event", "command.changed", "old", fmt.Sprintf("%+v", found.Command), "new", fmt.Sprintf("%+v", desired.Command))
		return true
	}
	if !reflect.DeepEqual(found.Args, desired.Args) {
		debug.Log("event", "args.changed", "old", fmt.Sprintf("%+v", found.Args), "new", fmt.Sprintf("%+v", desired.Args))
		return true
	}
	if !reflect.DeepEqual(found.Env, desired.Env) {
		debug.Log("event", "env.changed", "old", fmt.Sprintf("%+v", found.Env), "new", fmt.Sprintf("%+v", desired.Env))
		return true
	}
	if !reflect.DeepEqual(found.VolumeMounts, desired.VolumeMounts) {
		debug.Log("event", "volumeMounts.changed", "old", fmt.Sprintf("%+v", found.VolumeMounts), "new", fmt.Sprintf("%+v", desired.VolumeMounts))
		return true
	}

	return false
}

func (r *Reconciler) shouldUpdateContainerList(found, desired []corev1.Container) bool {
	debug := level.Debug(log.With(r.logger, "method", "Reconciler.shouldUpdateContainerList"))

	if len(found) != len(desired) {
		debug.Log("event", "container.list.length.changed", "old", len(found), "new", len(desired))
		found = desired
		return true
	}

	for i, _ := range found {
		if r.shouldUpdateContainer(found[i], desired[i]) {
			debug.Log("event", "container.list.item.changed", "index", i)
			return true
		}
	}

	return false
}
