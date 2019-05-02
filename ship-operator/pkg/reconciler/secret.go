package reconciler

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

// stateSecretSHA returns the SHA256 checksum of the secret key specified in the instance
// or an empty string if not found
func (r *Reconciler) stateSecretSHA() string {
	if r.instance == nil || r.secret == nil {
		return ""
	}
	return fmt.Sprintf("%x", sha256.New().Sum(r.stateSecretData()))
}

func (r *Reconciler) stateSecretData() []byte {
	if r.instance == nil || r.secret == nil {
		return nil
	}
	return r.secret.Data[r.instance.Spec.State.ValueFrom.SecretKeyRef.Key]
}

func (r *Reconciler) addSecretMeta() error {
	labels := r.secret.GetLabels()
	labels["shipwatch"] = ""
	r.secret.SetLabels(labels)

	// add the full shipwatch name to annotations; there may be multiple
	// shipwatches using this secret.
	annotations := r.secret.GetAnnotations()
	names := strings.Split(annotations["shipwatch"], ",")
	found := false
	for _, name := range names {
		if name == r.instanceName {
			found = true
			break
		}
	}
	if !found {
		names = append(names, r.instanceName)
		annotations["shipwatch"] = strings.Join(names, ",")
		r.secret.SetAnnotations(annotations)
	}

	err := r.Client.Update(context.TODO(), r.secret)
	if err != nil {
		level.Error(log.With(r.logger)).Log("method", "addSecretMeta", "step", "update.secret", "error", err)
		return err
	}
	return nil
}

func (r *Reconciler) removeSecretMeta() error {
	annotations := r.secret.GetAnnotations()
	names := strings.Split(annotations["shipwatch"], ",")
	next := make([]string, 0)
	for _, name := range names {
		if name != r.instanceName {
			next = append(next, name)
		}
	}
	if len(next) == 0 {
		// remove the "shipwatch" label if all shipwatch annotations have been removed
		delete(annotations, "shipwatch")
		labels := r.secret.GetLabels()
		delete(labels, "shipwatch")
		r.secret.SetLabels(labels)
	} else {
		annotations["shipwatch"] = strings.Join(next, ",")
	}
	r.secret.SetAnnotations(annotations)

	// also remove the "shipwatch" label if no more shipwatch instances refer to this secret
	if len(next) == 0 {
		labels := r.secret.GetLabels()
		delete(labels, "shipwatch")
		r.secret.SetLabels(labels)
	}

	err := r.Client.Update(context.TODO(), r.secret)
	if err != nil {
		level.Error(log.With(r.logger)).Log("method", "reconciler.removeSecretMeta")
		return err
	}
	return nil
}
