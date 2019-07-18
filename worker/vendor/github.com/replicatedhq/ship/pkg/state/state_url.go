package state

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-kit/kit/log"
	"github.com/pkg/errors"
)

type urlSerializer struct {
	getURL string
	putURL string
	logger log.Logger
}

func newURLSerializer(logger log.Logger, getURL string, putURL string) stateSerializer {
	return &urlSerializer{logger: logger, getURL: getURL, putURL: putURL}
}

func (s *urlSerializer) Load() (State, error) {
	resp, err := http.Get(s.getURL)
	if err != nil {
		return State{}, errors.Wrap(err, "make get request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return State{}, errors.Errorf("unexpected get status: %v", resp.StatusCode)
	}

	serialized, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return State{}, errors.Wrap(err, "read state body")
	}

	// An empty file should be treated as empty state
	if len(strings.TrimSpace(string(serialized))) == 0 {
		return State{}, nil
	}

	var state State
	if err := json.Unmarshal(serialized, &state); err != nil {
		return State{}, errors.Wrap(err, "unmarshal state")
	}

	return state, nil
}

func (s *urlSerializer) Save(state State) error {
	serialized, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return errors.Wrap(err, "serialize state")
	}

	req, err := http.NewRequest("PUT", s.putURL, bytes.NewBuffer(serialized))
	if err != nil {
		return errors.Wrap(err, "create request")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "put state object")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusCreated {
		return errors.Errorf("unexpected put status: %v", resp.StatusCode)
	}

	return nil
}

func (s *urlSerializer) Remove() error {
	return errors.New("URL based state storage does not support state removal")
}
