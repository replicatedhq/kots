package state

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/replicatedhq/ship/pkg/constants"
	"github.com/spf13/afero"
)

type fileSerializer struct {
	fs     afero.Afero
	logger log.Logger
}

func newFileSerializer(fs afero.Afero, logger log.Logger) stateSerializer {
	return &fileSerializer{fs: fs, logger: logger}
}

func (s *fileSerializer) Load() (State, error) {
	if _, err := s.fs.Stat(constants.StatePath); os.IsNotExist(err) {
		level.Debug(s.logger).Log("msg", "no saved state exists", "path", constants.StatePath)
		return State{}, nil
	}

	serialized, err := s.fs.ReadFile(constants.StatePath)
	if err != nil {
		return State{}, errors.Wrap(err, "read state file")
	}

	var state State
	if err := json.Unmarshal(serialized, &state); err != nil {
		return State{}, errors.Wrap(err, "unmarshal state")
	}

	level.Debug(s.logger).Log("event", "state.resolve", "type", "versioned")
	return state, nil
}

func (s *fileSerializer) Save(state State) error {
	serialized, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return errors.Wrap(err, "serialize state")
	}

	err = s.fs.MkdirAll(filepath.Dir(constants.StatePath), 0700)
	if err != nil {
		return errors.Wrap(err, "mkdir state")
	}

	err = s.fs.WriteFile(constants.StatePath, serialized, 0644)
	if err != nil {
		return errors.Wrap(err, "write state file")
	}

	return nil
}
