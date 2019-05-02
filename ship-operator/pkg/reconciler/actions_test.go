package reconciler

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"testing"

	shipv1beta1 "github.com/replicatedhq/ship-operator/pkg/apis/ship/v1beta1"
	"github.com/replicatedhq/ship-operator/pkg/generator"
	"github.com/replicatedhq/ship-operator/pkg/logger"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetActions(t *testing.T) {
	aState := []byte("a")
	aSHA := fmt.Sprintf("%x", sha256.New().Sum(aState))

	var tests = []struct {
		name       string
		reconciler *Reconciler
		answer     actions
	}{
		{
			name: "instance does not exist, update job exists, secret has meta",
			reconciler: &Reconciler{
				instance:  nil,
				updateJob: newUpdateJob(aSHA),
				watchJob:  nil,
				secret:    newSecretWithMeta(aState),
			},
			answer: actions{
				removeSecretMeta: true,
			},
		},
		{
			name: "instance does not exist, no jobs exist, secret does not have meta",
			reconciler: &Reconciler{
				instance:  nil,
				secret:    newSecretWithoutMeta(aState),
				watchJob:  nil,
				updateJob: nil,
			},
			answer: actions{},
		},
		{
			name: "instance exists; no jobs exist; secret lacks meta",
			reconciler: &Reconciler{
				instance:  newShipWatch(),
				updateJob: nil,
				watchJob:  nil,
				secret:    newSecretWithoutMeta(aState),
			},
			answer: actions{
				addSecretMeta:  true,
				createWatchJob: true,
			},
		},
		{
			name: "instance exists, no jobs exist, secret has meta",
			reconciler: &Reconciler{
				instance:  newShipWatch(),
				updateJob: nil,
				watchJob:  nil,
				secret:    newSecretWithMeta(aState),
			},
			answer: actions{
				createWatchJob: true,
			},
		},
		{
			name: "instance exists, watch job is running, secret has meta",
			reconciler: &Reconciler{
				instance:  newShipWatch(),
				watchJob:  newWatchJob(aSHA),
				updateJob: nil,
				secret:    newSecretWithMeta(aState),
			},
			answer: actions{},
		},
		{
			name: "instance exists, update job is running, secret has meta",
			reconciler: &Reconciler{
				instance:  newShipWatch(),
				watchJob:  nil,
				updateJob: newUpdateJob(aSHA),
				secret:    newSecretWithMeta(aState),
			},
		},
		{
			name: "instance exists, update job is running, watch job is completed, secret has meta",
			reconciler: &Reconciler{
				instance:  newShipWatch(),
				watchJob:  withStatusCompleted(newWatchJob(aSHA)),
				updateJob: newUpdateJob(aSHA),
				secret:    newSecretWithMeta(aState),
			},
			answer: actions{
				deleteWatchJob: true,
			},
		},
		{
			name: "instance exists, watch job is running, update job is completed, secret has meta",
			reconciler: &Reconciler{
				instance:  newShipWatch(),
				watchJob:  newWatchJob(aSHA),
				updateJob: withStatusCompleted(newUpdateJob(aSHA)),
				secret:    newSecretWithMeta(aState),
			},
			answer: actions{
				deleteUpdateJob: true,
			},
		},
		{
			name: "instance exists, watch job is completed, secret has meta",
			reconciler: &Reconciler{
				instance:  newShipWatch(),
				watchJob:  withStatusCompleted(newWatchJob(aSHA)),
				updateJob: nil,
				secret:    newSecretWithMeta(aState),
			},
			answer: actions{
				createUpdateJob: true,
			},
		},
		{
			name: "instance exists, update job is completed, secret has meta",
			reconciler: &Reconciler{
				instance:  newShipWatch(),
				watchJob:  nil,
				updateJob: withStatusCompleted(newUpdateJob(aSHA)),
				secret:    newSecretWithMeta(aState),
			},
			answer: actions{
				createWatchJob: true,
			},
		},
		{
			name: "instance exists, watch job is running, secret missing meta",
			reconciler: &Reconciler{
				instance:  newShipWatch(),
				watchJob:  newWatchJob(aSHA),
				updateJob: nil,
				secret:    newSecretWithoutMeta(aState),
			},
			answer: actions{
				addSecretMeta: true,
			},
		},
		{
			name: "instance exists, both jobs running, secret has meta",
			reconciler: &Reconciler{
				instance:  newShipWatch(),
				watchJob:  newWatchJob(aSHA),
				updateJob: newUpdateJob(aSHA),
				secret:    newSecretWithMeta(aState),
			},
			answer: actions{
				deleteWatchJob: true,
			},
		},
		{
			name: "instance exists, both jobs completed, secret has meta",
			reconciler: &Reconciler{
				instance:  newShipWatch(),
				watchJob:  withStatusCompleted(newWatchJob(aSHA)),
				updateJob: withStatusCompleted(newUpdateJob(aSHA)),
				secret:    newSecretWithMeta(aState),
			},
			answer: actions{
				deleteWatchJob: true,
			},
		},
		{
			name: "instance exists, watch job running, no secret",
			reconciler: &Reconciler{
				instance:  newShipWatch(),
				watchJob:  newWatchJob(""),
				updateJob: nil,
				secret:    nil,
			},
			answer: actions{},
		},
		{
			name: "instance exists, no jobs, no secret",
			reconciler: &Reconciler{
				instance:  newShipWatch(),
				watchJob:  nil,
				updateJob: nil,
				secret:    nil,
			},
			answer: actions{
				createWatchJob: true,
			},
		},
		{
			name: "instance exists, watch job is running, secret has changed",
			reconciler: &Reconciler{
				instance:  newShipWatch(),
				watchJob:  newWatchJob(aSHA),
				updateJob: nil,
				secret:    newSecretWithMeta([]byte("b")),
			},
			answer: actions{
				deleteWatchJob: true,
				createWatchJob: true,
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.reconciler.logger = logger.FromEnv()
			test.reconciler.instanceName = "myapp"
			test.reconciler.generator = generator.NewGenerator(test.reconciler.instance)
			output := test.reconciler.getActions()
			if !reflect.DeepEqual(output, test.answer) {
				t.Errorf("got %+v, want %+v", output, test.answer)
			}
		})
	}
}

func newShipWatch() *shipv1beta1.ShipWatch {
	sw := &shipv1beta1.ShipWatch{
		TypeMeta: metav1.TypeMeta{APIVersion: shipv1beta1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myapp",
			Namespace: metav1.NamespaceDefault,
		},
		Spec: shipv1beta1.ShipWatchSpec{
			State: shipv1beta1.StateSpec{
				ValueFrom: shipv1beta1.ShipWatchValueFromSpec{
					SecretKeyRef: shipv1beta1.SecretKeyRef{
						Name: "myappsecret",
						Key:  "state.json",
					},
				},
			},
		},
	}

	return sw
}

func newSecretWithoutMeta(state []byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myappsecret",
			Namespace: metav1.NamespaceDefault,
		},
		Data: map[string][]byte{
			"state.json": state,
		},
	}
}

func newSecretWithMeta(state []byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{APIVersion: "v1", Kind: "Secret"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "myappsecret",
			Namespace: metav1.NamespaceDefault,
			Labels: map[string]string{
				"shipwatch": "",
			},
			Annotations: map[string]string{
				"shipwatch": "myapp",
			},
		},
		Data: map[string][]byte{
			"state.json": state,
		},
	}
}

func newUpdateJob(secretSHA string) *batchv1.Job {
	sw := newShipWatch()
	gen := generator.NewGenerator(sw)
	return gen.UpdateJob(secretSHA)
}

func newWatchJob(secretSHA string) *batchv1.Job {
	sw := newShipWatch()
	gen := generator.NewGenerator(sw)
	return gen.WatchJob(secretSHA)
}

func withStatusCompleted(job *batchv1.Job) *batchv1.Job {
	job.Status.Conditions = append(job.Status.Conditions, batchv1.JobCondition{
		Type:   batchv1.JobComplete,
		Status: corev1.ConditionTrue,
	})

	return job
}
