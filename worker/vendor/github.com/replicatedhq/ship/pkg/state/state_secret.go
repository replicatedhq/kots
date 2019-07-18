package state

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type secretSerializer struct {
	secretNS   string
	secretName string
	secretKey  string
	logger     log.Logger
}

func newSecretSerializer(logger log.Logger, ns string, name string, key string) stateSerializer {
	return &secretSerializer{logger: logger, secretNS: ns, secretName: name, secretKey: key}
}

func (s *secretSerializer) Load() (State, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return State{}, errors.Wrap(err, "get in cluster config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return State{}, errors.Wrap(err, "get kubernetes client")
	}

	if s.secretNS == "" {
		return State{}, errors.New("secret-namespace is not set")
	}
	if s.secretName == "" {
		return State{}, errors.New("secret-name is not set")
	}
	if s.secretKey == "" {
		return State{}, errors.New("secret-key is not set")
	}

	secret, err := clientset.CoreV1().Secrets(s.secretNS).Get(s.secretName, metav1.GetOptions{})
	if err != nil {
		return State{}, errors.Wrap(err, "get secret")
	}

	serialized, ok := secret.Data[s.secretKey]
	if !ok {
		err := fmt.Errorf("key not found in secret %q", s.secretName)
		return State{}, errors.Wrap(err, "get state from secret")
	}

	// An empty secret should be treated as empty state
	if len(strings.TrimSpace(string(serialized))) == 0 {
		return State{}, nil
	}

	var state State
	if err := json.Unmarshal(serialized, &state); err != nil {
		return State{}, errors.Wrap(err, "unmarshal state")
	}

	level.Debug(s.logger).Log(
		"event", "state.unmarshal",
		"type", "versioned",
		"source", "secret",
		"value", fmt.Sprintf("%+v", state),
	)

	level.Debug(s.logger).Log("event", "state.resolve", "type", "versioned")
	return state, nil
}

func (s *secretSerializer) Save(state State) error {
	serialized, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return errors.Wrap(err, "serialize state")
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.Wrap(err, "get in cluster config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return errors.Wrap(err, "get kubernetes client")
	}

	secret, err := clientset.CoreV1().Secrets(s.secretNS).Get(s.secretName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "get secret")
	}

	secret.Data[s.secretKey] = serialized
	debug := level.Debug(log.With(s.logger, "method", "serializeHelmValues"))

	debug.Log("event", "serializeAndWriteStateSecret", "name", secret.Name)

	_, err = clientset.CoreV1().Secrets(s.secretNS).Update(secret)
	if err != nil {
		return errors.Wrap(err, "update secret")
	}

	return nil
}

func (s *secretSerializer) Remove() error {
	return errors.New("secret based state storage does not support state removal")
}
