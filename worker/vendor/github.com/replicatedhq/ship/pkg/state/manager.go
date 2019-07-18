package state

import (
	"fmt"
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
	CachedState() (State, error)
	CommitState() error
	ReloadFile() error
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

type stateSerializer interface {
	Load() (State, error)
	Save(State) error
	Remove() error
}

var _ Manager = &MManager{}

var newManagerLock = &sync.Mutex{}
var (
	managerInstance Manager
)

// MManager is the saved output of a plan run to load on future runs
type MManager struct {
	Logger         log.Logger
	FS             afero.Afero
	V              *viper.Viper
	patcher        patch.Patcher
	stateUpdateMut sync.Mutex
	StateRWMut     sync.RWMutex

	cachedState *State
}

func (m *MManager) Save(v State) error {
	debug := level.Debug(log.With(m.Logger, "method", "SerializeShipMetadata"))

	debug.Log("event", "safeStateUpdate")
	_, err := m.StateUpdate(func(state State) (State, error) {
		state = v
		return state, nil
	})
	if err != nil {
		return errors.Wrap(err, "save state")
	}

	return m.CommitState()
}

func GetSingleton() Manager {
	return managerInstance
}

// GetManager will create and return a singleton state manager object.
// This should be used in everything that isn't a test.
func GetManager(logger log.Logger, fs afero.Afero, v *viper.Viper) (Manager, error) {
	newManagerLock.Lock()
	defer newManagerLock.Unlock()

	if managerInstance != nil {
		return managerInstance, nil
	}

	instance := &MManager{
		Logger: logger,
		FS:     fs,
		V:      v,
	}
	if err := instance.tryLoad(); err != nil {
		return nil, errors.Wrap(err, "load state")
	}

	managerInstance = instance

	return instance, nil
}

// This will create a new state manager that isn't a singleton.
// Use this for tests, where state needs to be reset.
func NewDisposableManager(logger log.Logger, fs afero.Afero, v *viper.Viper) (Manager, error) {
	instance := &MManager{
		Logger: logger,
		FS:     fs,
		V:      v,
	}
	if err := instance.tryLoad(); err != nil {
		return nil, errors.Wrap(err, "load state")
	}
	return instance, nil
}

type Update func(State) (State, error)

func (m *MManager) ReloadFile() error {
	newManagerLock.Lock()
	defer newManagerLock.Unlock()

	if err := m.tryLoad(); err != nil {
		return errors.Wrap(err, "reload state")
	}

	return nil
}

// applies the provided updater to the current state. Returns the new state and err
func (m *MManager) StateUpdate(updater Update) (State, error) {
	m.stateUpdateMut.Lock()
	defer m.stateUpdateMut.Unlock()

	if m.cachedState.V1 == nil {
		m.cachedState.V1 = &V1{}
	}

	updatedState, err := updater(m.cachedState.Versioned())
	if err != nil {
		return State{}, errors.Wrap(err, "run state update function in safe updater")
	}

	stateFrom := m.V.GetString("state-from")
	if stateFrom == "" || stateFrom == "file" {
		// When state source is a file, always flush cached contents to disk.
		// This is done to preserve current ship behavior.
		err := m.serializeAndWriteState(updatedState)
		if err != nil {
			return State{}, errors.Wrap(err, "write state in safe updater")
		}
	}

	m.cachedState = &updatedState
	return updatedState, nil
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

func (m *MManager) getStateSerializer() (stateSerializer, error) {
	stateFrom := m.V.GetString("state-from")
	if stateFrom == "" {
		stateFrom = "file"
	}

	switch stateFrom {
	case "file":
		// Even thought there's a "state-file" command line argument,
		// ship has never used it to load state.
		return newFileSerializer(m.FS, m.Logger, constants.StatePath), nil
	case "secret":
		return newSecretSerializer(m.Logger, m.V.GetString("secret-namespace"), m.V.GetString("secret-name"), m.V.GetString("secret-key")), nil
	case "url":
		return newURLSerializer(m.Logger, m.V.GetString("state-get-url"), m.V.GetString("state-put-url")), nil
	default:
		return nil, fmt.Errorf("unsupported state-from value: %q", stateFrom)
	}
}

// CachedState will return currently chached state.
func (m *MManager) CachedState() (State, error) {
	if m.cachedState == nil {
		return State{}, errors.New("state is not initialized")
	}
	return *m.cachedState, nil
}

func (m *MManager) CommitState() error {
	if m.cachedState == nil {
		errors.New("cannot save state that has not been initialized")
	}
	return errors.Wrap(m.serializeAndWriteState(*m.cachedState), "serialize cached state")
}

func (m *MManager) tryLoad() error {
	s, err := m.getStateSerializer()
	if err != nil {
		return errors.Wrap(err, "create state serializer")
	}

	state, err := s.Load()
	if err != nil {
		return errors.Wrap(err, "load state")
	}

	m.cachedState = &state
	return nil
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
	s, err := m.getStateSerializer()
	if err != nil {
		return errors.Wrap(err, "create state serializer")
	}

	err = s.Remove()
	if err != nil {
		return errors.Wrap(err, "remove state file")
	}

	m.cachedState = &State{}

	return nil
}

func (m *MManager) serializeAndWriteState(state State) error {
	m.StateRWMut.Lock()
	defer m.StateRWMut.Unlock()
	state = state.migrateDeprecatedFields()

	s, err := m.getStateSerializer()
	if err != nil {
		return errors.Wrap(err, "create state serializer")
	}

	if err := s.Save(state); err != nil {
		return errors.Wrap(err, "save state")
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
