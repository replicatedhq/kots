package state

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/afero"

	"github.com/replicatedhq/ship/pkg/constants"
)

// write the helm values file and original values file to disk from state if they exist
func writeHelmFiles(fs afero.Afero, state State) error {
	if state.CurrentHelmValues() == "" && state.CurrentHelmValuesDefaults() == "" {
		exists, err := fs.Exists(constants.HelmValuesPath)
		if err != nil {
			return errors.Wrapf(err, "check if helm values dir %s exists", constants.HelmValuesPath)
		} else if exists {
			err = fs.RemoveAll(constants.HelmValuesPath)
			if err != nil {
				return errors.Wrapf(err, "remove helm values dir %s", constants.HelmValuesPath)
			}
		}
		return nil
	}

	err := fs.MkdirAll(constants.HelmValuesPath, os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "create dir %s for helm values files", constants.HelmValuesPath)
	}

	err = fs.WriteFile(filepath.Join(constants.HelmValuesPath, "values.yaml"), []byte(state.CurrentHelmValues()), os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "write helm values file at %s", filepath.Join(constants.HelmValuesPath, "values.yaml"))
	}

	err = fs.WriteFile(filepath.Join(constants.HelmValuesPath, "defaults.yaml"), []byte(state.CurrentHelmValuesDefaults()), os.ModePerm)
	if err != nil {
		return errors.Wrapf(err, "write helm defaults file at %s", filepath.Join(constants.HelmValuesPath, "defaults.yaml"))
	}

	return nil
}

// read the helm values file and original values file from disk and replace them in state if they exist
func readHelmFiles(fs afero.Afero, state State) (State, error) {
	exists, err := fs.Exists(filepath.Join(constants.HelmValuesPath, "values.yaml"))
	if err != nil {
		return state, errors.Wrapf(err, "check helm values file at %s", filepath.Join(constants.HelmValuesPath, "values.yaml"))
	}
	if exists {
		// read in the file and use it as state.HelmValues
		valuesContents, err := fs.ReadFile(filepath.Join(constants.HelmValuesPath, "values.yaml"))
		if err != nil {
			return state, errors.Wrapf(err, "read helm values file at %s", filepath.Join(constants.HelmValuesPath, "values.yaml"))
		}
		if state.V1 == nil {
			state.V1 = &V1{}
		}
		state.V1.HelmValues = string(valuesContents)
	}

	exists, err = fs.Exists(filepath.Join(constants.HelmValuesPath, "defaults.yaml"))
	if err != nil {
		return state, errors.Wrapf(err, "check helm defaults file at %s", filepath.Join(constants.HelmValuesPath, "defaults.yaml"))
	}
	if exists {
		// read in the file and use it as state.HelmValuesDefaults
		defaultsContents, err := fs.ReadFile(filepath.Join(constants.HelmValuesPath, "defaults.yaml"))
		if err != nil {
			return state, errors.Wrapf(err, "read helm defaults file at %s", filepath.Join(constants.HelmValuesPath, "defaults.yaml"))
		}
		if state.V1 == nil {
			state.V1 = &V1{}
		}
		state.V1.HelmValuesDefaults = string(defaultsContents)
	}

	return state, nil
}
