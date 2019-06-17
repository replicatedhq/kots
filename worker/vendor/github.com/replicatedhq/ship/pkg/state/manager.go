package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/api"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/replicatedhq/ship/pkg/patch"
	"github.com/replicatedhq/ship/pkg/util"
	"github.com/replicatedhq/ship/pkg/version"

	"github.com/spf13/afero"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type Manager interface {
	SerializeHelmValues(values string, defaults string) error
	SerializeReleaseName(name string) error
	SerializeNamespace(namespace string) error
	SerializeConfig(
		assets []api.Asset,
		meta api.ReleaseMetadata,
		templateContext map[string]interface{},
	) error
	TryLoad() (State, error)
	StateUpdate(updater Update) (State, error)
	RemoveStateFile() error
	SaveKustomize(kustomize *Kustomize) error
	SerializeUpstream(URL string) error
	SerializeContentSHA(contentSHA string) error
	SerializeShipMetadata(api.ShipAppMetadata, string) error
	SerializeAppMetadata(api.ReleaseMetadata) error
	SerializeUpstreamContents(contents *UpstreamContents) error
	Save(v State) error
	ResetLifecycle() error
	UpdateVersion()

	AddCert(name string, newCert util.CertType) error
	AddCA(name string, newCA util.CAType) error
}

var _ Manager = &MManager{}

// MManager is the saved output of a plan run to load on future runs
type MManager struct {
	Logger         log.Logger
	FS             afero.Afero
	V              *viper.Viper
	patcher        patch.Patcher
	stateUpdateMut sync.Mutex
	StateRWMut     sync.RWMutex
}

func (m *MManager) Save(v State) error {
	debug := level.Debug(log.With(m.Logger, "method", "SerializeShipMetadata"))

	debug.Log("event", "safeStateUpdate")
	_, err := m.StateUpdate(func(state State) (State, error) {
		state = v
		return state, nil
	})
	return err
}

func NewManager(
	logger log.Logger,
	fs afero.Afero,
	v *viper.Viper,
) Manager {
	return &MManager{
		Logger: logger,
		FS:     fs,
		V:      v,
	}
}

type Update func(State) (State, error)

// applies the provided updater to the current state. Returns the new state and err
func (m *MManager) StateUpdate(updater Update) (State, error) {
	m.stateUpdateMut.Lock()
	defer m.stateUpdateMut.Unlock()

	currentState, err := m.TryLoad()
	if err != nil {
		return State{}, errors.Wrap(err, "tryLoad in safe updater")
	}

	if currentState.V1 == nil {
		currentState.V1 = &V1{}
	}

	updatedState, err := updater(currentState.Versioned())
	if err != nil {
		return State{}, errors.Wrap(err, "run state update function in safe updater")
	}

	return updatedState, errors.Wrap(m.serializeAndWriteState(updatedState), "write state in safe updater")
}

// SerializeShipMetadata is used by `ship init` to serialize metadata from ship applications to state file
func (m *MManager) SerializeShipMetadata(metadata api.ShipAppMetadata, applicationType string) error {
	debug := level.Debug(log.With(m.Logger, "method", "SerializeShipMetadata"))

	debug.Log("event", "safeStateUpdate")
	_, err := m.StateUpdate(func(state State) (State, error) {
		state.V1.Metadata = &Metadata{
			ApplicationType: applicationType,
			ReleaseNotes:    metadata.ReleaseNotes,
			Version:         metadata.Version,
			Icon:            metadata.Icon,
			Name:            metadata.Name,
		}
		return state, nil
	})
	return err
}

// SerializeAppMetadata is used by `ship app` to serialize replicated app metadata to state file
func (m *MManager) SerializeAppMetadata(metadata api.ReleaseMetadata) error {
	debug := level.Debug(log.With(m.Logger, "method", "SerializeAppMetadata"))

	debug.Log("event", "safeStateUpdate")
	_, err := m.StateUpdate(func(state State) (State, error) {
		if state.V1.Metadata == nil {
			state.V1.Metadata = &Metadata{}
		}
		state.V1.Metadata.ApplicationType = "replicated.app"
		state.V1.Metadata.ReleaseNotes = metadata.ReleaseNotes
		state.V1.Metadata.Version = metadata.Semver
		state.V1.Metadata.CustomerID = metadata.CustomerID
		state.V1.Metadata.InstallationID = metadata.InstallationID
		state.V1.Metadata.LicenseID = metadata.LicenseID
		state.V1.Metadata.AppSlug = metadata.AppSlug
		state.V1.Metadata.License = License{
			ID:        metadata.License.ID,
			Assignee:  metadata.License.Assignee,
			CreatedAt: metadata.License.CreatedAt,
			ExpiresAt: metadata.License.ExpiresAt,
			Type:      metadata.License.Type,
		}
		return state, nil
	})
	return err
}

// SerializeUpstream is used by `ship init` to serialize a state file with ChartURL to disk
func (m *MManager) SerializeUpstream(upstream string) error {
	debug := level.Debug(log.With(m.Logger, "method", "SerializeUpstream"))

	debug.Log("event", "safeStateUpdate")
	_, err := m.StateUpdate(func(state State) (State, error) {
		state.V1.Upstream = upstream
		return state, nil
	})
	return err
}

// SerializeContentSHA writes the contentSHA to the state file
func (m *MManager) SerializeContentSHA(contentSHA string) error {
	debug := level.Debug(log.With(m.Logger, "method", "SerializeContentSHA"))

	debug.Log("event", "safeStateUpdate")
	_, err := m.StateUpdate(func(state State) (State, error) {
		state.V1.ContentSHA = contentSHA
		return state, nil
	})
	return err
}

// SerializeHelmValues takes user input helm values and serializes a state file to disk
func (m *MManager) SerializeHelmValues(values string, defaults string) error {
	debug := level.Debug(log.With(m.Logger, "method", "serializeHelmValues"))

	debug.Log("event", "safeStateUpdate")
	_, err := m.StateUpdate(func(state State) (State, error) {
		state.V1.HelmValues = values
		state.V1.HelmValuesDefaults = defaults
		return state, nil
	})
	return err
}

// SerializeReleaseName serializes to disk the name to use for helm template
func (m *MManager) SerializeReleaseName(name string) error {
	debug := level.Debug(log.With(m.Logger, "method", "serializeReleaseName"))

	debug.Log("event", "safeStateUpdate")
	_, err := m.StateUpdate(func(state State) (State, error) {
		state.V1.ReleaseName = name
		return state, nil
	})
	return err
}

// SerializeNamespace serializes to disk the namespace to use for helm template
func (m *MManager) SerializeNamespace(namespace string) error {
	debug := level.Debug(log.With(m.Logger, "method", "serializeNamespace"))

	debug.Log("event", "safeStateUpdate")
	_, err := m.StateUpdate(func(state State) (State, error) {
		state.V1.Namespace = namespace
		return state, nil
	})
	return err
}

// SerializeConfig takes the application data and input params and serializes a state file to disk
func (m *MManager) SerializeConfig(assets []api.Asset, meta api.ReleaseMetadata, templateContext map[string]interface{}) error {
	debug := level.Debug(log.With(m.Logger, "method", "serializeConfig"))

	debug.Log("event", "safeStateUpdate")
	_, err := m.StateUpdate(func(state State) (State, error) {
		state.V1.Config = templateContext
		return state, nil
	})
	return err
}

// SerializeConfig takes the application data and input params and serializes a state file to disk
func (m *MManager) SerializeUpstreamContents(contents *UpstreamContents) error {
	debug := level.Debug(log.With(m.Logger, "method", "serializeUpstreamContents"))

	debug.Log("event", "safeStateUpdate")
	_, err := m.StateUpdate(func(state State) (State, error) {

		state.V1.UpstreamContents = contents
		return state, nil
	})
	return err
}

// TryLoad will attempt to load a state file from disk, if present
func (m *MManager) TryLoad() (State, error) {
	m.StateRWMut.RLock()
	defer m.StateRWMut.RUnlock()
	stateFrom := m.V.GetString("state-from")
	if stateFrom == "" {
		stateFrom = "file"
	}

	// TODO consider an interface

	switch stateFrom {
	case "file":
		return m.tryLoadFromFile()
	case "secret":
		return m.tryLoadFromSecret()
	default:
		err := fmt.Errorf("unsupported state-from value: %q", stateFrom)
		return State{}, errors.Wrap(err, "try load state")
	}
}

// ResetLifecycle is used by `ship update --headed` to reset the saved stepsCompleted
// in the state.json
func (m *MManager) ResetLifecycle() error {
	debug := level.Debug(log.With(m.Logger, "method", "ResetLifecycle"))

	debug.Log("event", "safeStateUpdate")
	_, err := m.StateUpdate(func(state State) (State, error) {

		state.V1.Lifecycle = nil
		return state, nil
	})
	return err
}

// tryLoadFromSecret will attempt to load the state from a secret
// currently only supports in-cluster execution
func (m *MManager) tryLoadFromSecret() (State, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return State{}, errors.Wrap(err, "get in cluster config")
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return State{}, errors.Wrap(err, "get kubernetes client")
	}

	ns := m.V.GetString("secret-namespace")
	if ns == "" {
		return State{}, errors.New("secret-namespace is not set")
	}
	secretName := m.V.GetString("secret-name")
	if secretName == "" {
		return State{}, errors.New("secret-name is not set")
	}
	secretKey := m.V.GetString("secret-key")
	if secretKey == "" {
		return State{}, errors.New("secret-key is not set")
	}

	secret, err := clientset.CoreV1().Secrets(ns).Get(secretName, metav1.GetOptions{})
	if err != nil {
		return State{}, errors.Wrap(err, "get secret")
	}

	serialized, ok := secret.Data[secretKey]
	if !ok {
		err := fmt.Errorf("key %q not found in secret %q", secretKey, secretName)
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

	level.Debug(m.Logger).Log(
		"event", "state.unmarshal",
		"type", "versioned",
		"source", "secret",
		"value", fmt.Sprintf("%+v", state),
	)

	level.Debug(m.Logger).Log("event", "state.resolve", "type", "versioned")
	return state, nil
}

func (m *MManager) tryLoadFromFile() (State, error) {
	if _, err := m.FS.Stat(constants.StatePath); os.IsNotExist(err) {
		level.Debug(m.Logger).Log("msg", "no saved state exists", "path", constants.StatePath)
		return State{}, nil
	}

	serialized, err := m.FS.ReadFile(constants.StatePath)
	if err != nil {
		return State{}, errors.Wrap(err, "read state file")
	}

	var state State
	if err := json.Unmarshal(serialized, &state); err != nil {
		return State{}, errors.Wrap(err, "unmarshal state")
	}

	level.Debug(m.Logger).Log("event", "state.resolve", "type", "versioned")
	return state, nil
}

func (m *MManager) SaveKustomize(kustomize *Kustomize) error {
	debug := level.Debug(log.With(m.Logger, "method", "SaveKustomize"))

	debug.Log("event", "safeStateUpdate")
	_, err := m.StateUpdate(func(state State) (State, error) {

		state.V1.Kustomize = kustomize
		return state, nil
	})
	return err
}

// RemoveStateFile will attempt to remove the state file from disk
func (m *MManager) RemoveStateFile() error {
	statePath := m.V.GetString("state-file")
	if statePath == "" {
		statePath = constants.StatePath
	}

	err := m.FS.RemoveAll(statePath)
	if err != nil {
		return errors.Wrap(err, "remove state file")
	}

	return nil
}

func (m *MManager) serializeAndWriteState(state State) error {
	m.StateRWMut.Lock()
	defer m.StateRWMut.Unlock()
	debug := level.Debug(log.With(m.Logger, "method", "serializeAndWriteState"))
	state = state.migrateDeprecatedFields()

	stateFrom := m.V.GetString("state-from")
	if stateFrom == "" {
		stateFrom = "file"
	}

	debug.Log("stateFrom", stateFrom)

	switch stateFrom {
	case "file":
		return m.serializeAndWriteStateFile(state)
	case "secret":
		return m.serializeAndWriteStateSecret(state)
	default:
		err := fmt.Errorf("unsupported state-from value: %q", stateFrom)
		return errors.Wrap(err, "serializeAndWriteState")
	}
}

func (m *MManager) serializeAndWriteStateFile(state State) error {

	serialized, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return errors.Wrap(err, "serialize state")
	}

	err = m.FS.MkdirAll(filepath.Dir(constants.StatePath), 0700)
	if err != nil {
		return errors.Wrap(err, "mkdir state")
	}

	err = m.FS.WriteFile(constants.StatePath, serialized, 0644)
	if err != nil {
		return errors.Wrap(err, "write state file")
	}

	return nil
}

func (m *MManager) serializeAndWriteStateSecret(state State) error {
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

	secret, err := clientset.CoreV1().Secrets(m.V.GetString("secret-namespace")).Get(m.V.GetString("secret-name"), metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "get secret")
	}

	secret.Data[m.V.GetString("secret-key")] = serialized
	debug := level.Debug(log.With(m.Logger, "method", "serializeHelmValues"))

	debug.Log("event", "serializeAndWriteStateSecret", "name", secret.Name, "key", m.V.GetString("secret-key"))

	_, err = clientset.CoreV1().Secrets(m.V.GetString("secret-namespace")).Update(secret)
	if err != nil {
		return errors.Wrap(err, "update secret")
	}

	return nil
}

func (m *MManager) AddCert(name string, newCert util.CertType) error {
	debug := level.Debug(log.With(m.Logger, "method", "SaveKustomize"))

	debug.Log("event", "safeStateUpdate")
	_, err := m.StateUpdate(func(state State) (State, error) {

		if state.V1.Certs == nil {
			state.V1.Certs = make(map[string]util.CertType)
		}
		if _, ok := state.V1.Certs[name]; ok {
			return state, fmt.Errorf("cert with name %s already exists in state", name)
		}
		state.V1.Certs[name] = newCert
		return state, nil
	})
	return err
}

func (m *MManager) AddCA(name string, newCA util.CAType) error {
	debug := level.Debug(log.With(m.Logger, "method", "SaveKustomize"))

	debug.Log("event", "safeStateUpdate")
	_, err := m.StateUpdate(func(state State) (State, error) {

		if state.V1.CAs == nil {
			state.V1.CAs = make(map[string]util.CAType)
		}
		if _, ok := state.V1.CAs[name]; ok {
			return state, fmt.Errorf("cert with name %s already exists in state", name)
		}
		state.V1.CAs[name] = newCA
		return state, nil
	})
	return err
}

func (m *MManager) UpdateVersion() {
	debug := level.Debug(log.With(m.Logger, "method", "SaveKustomize"))

	debug.Log("event", "safeStateUpdate")
	_, _ = m.StateUpdate(func(state State) (State, error) {
		if state.V1 == nil {
			state.V1 = &V1{}
		}

		currentVersion := version.GetBuild()
		state.V1.ShipVersion = &currentVersion
		return state, nil
	})
	return
}
