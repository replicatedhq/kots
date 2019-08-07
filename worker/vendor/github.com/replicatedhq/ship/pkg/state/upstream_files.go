package state

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/spf13/afero"

	"github.com/replicatedhq/ship/pkg/constants"
)

// write the upstream files to disk from state if they exist
func writeUpstreamFiles(fs afero.Afero, state State) error {
	if state.V1 == nil || state.V1.UpstreamContents == nil {
		return nil
	}

	contents := state.V1.UpstreamContents

	err := fs.RemoveAll(constants.UpstreamContentsPath)
	if err != nil {
		return errors.Wrapf(err, "clear upstream contents path %s", constants.UpstreamContentsPath)
	}

	err = fs.MkdirAll(constants.UpstreamContentsPath, 0755)
	if err != nil {
		return errors.Wrapf(err, "create upstream contents path %s", constants.UpstreamContentsPath)
	}

	// different behavior for replicated.app upstreams vs others
	// replicated.app can be serialized directly, while other upstreams should be the raw files
	if contents.AppRelease != nil {
		appReleaseBytes, err := json.MarshalIndent(contents.AppRelease, "", "  ")
		if err != nil {
			return errors.Wrap(err, "marshal app release")
		}

		err = fs.WriteFile(constants.UpstreamAppReleasePath, appReleaseBytes, 0644)
		if err != nil {
			return errors.Wrapf(err, "write app release to %s", constants.UpstreamAppReleasePath)
		}
	} else {
		// write each upstream file individually
		for _, upstreamFile := range contents.UpstreamFiles {
			upstreamDir := filepath.Join(constants.UpstreamContentsPath, filepath.Dir(upstreamFile.FilePath))

			err := fs.MkdirAll(upstreamDir, 0755)
			if err != nil {
				return errors.Wrapf(err, "create directory within upstream contents path %s", upstreamDir)
			}

			upstreamFilePath := filepath.Join(constants.UpstreamContentsPath, upstreamFile.FilePath)
			rawContents, err := base64.StdEncoding.DecodeString(upstreamFile.FileContents)
			if err != nil {
				return errors.Wrapf(err, "decode upstream file %s contents", upstreamFile.FilePath)
			}

			err = fs.WriteFile(upstreamFilePath, rawContents, 0644)
			if err != nil {
				return errors.Wrapf(err, "write upstream file %s within upstream contents path %s", upstreamFile.FilePath, upstreamDir)
			}
		}
	}

	return nil
}

// read the upstream files from disk and replace them in state if they exist
func readUpstreamFiles(fs afero.Afero, state State) (State, error) {
	exists, err := fs.Exists(constants.UpstreamAppReleasePath)
	if err != nil {
		return state, errors.Wrapf(err, "check upstream app release file at %s", constants.UpstreamAppReleasePath)
	}
	if exists {
		// read in the file and use it as the upstream app release
		appReleaseBytes, err := fs.ReadFile(constants.UpstreamAppReleasePath)
		if err != nil {
			return state, errors.Wrapf(err, "read upstream app release file at %s", constants.UpstreamAppReleasePath)
		}
		if state.V1 == nil {
			state.V1 = &V1{}
		}
		if state.V1.UpstreamContents == nil {
			state.V1.UpstreamContents = &UpstreamContents{}
		}

		newAppRelease := ShipRelease{}
		err = json.Unmarshal(appReleaseBytes, &newAppRelease)
		if err != nil {
			return state, errors.Wrapf(err, "unmarshal upstream app release file from %s", constants.UpstreamAppReleasePath)
		}

		state.V1.UpstreamContents.AppRelease = &newAppRelease
		return state, nil
	}

	exists, err = fs.Exists(constants.UpstreamContentsPath)
	if err != nil {
		return state, errors.Wrapf(err, "check upstream dir %s", constants.UpstreamContentsPath)
	}
	if exists {
		newUpstreamContents := &UpstreamContents{}
		// read each upstream file individually, if any exist
		err = fs.Walk(constants.UpstreamContentsPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return errors.Wrapf(err, "walk %s", path)
			}
			if info.IsDir() {
				// ignore dirs, only save files
				return nil
			}

			// read file and save it as part of newUpstreamContents
			fileBytes, err := fs.ReadFile(path)
			if err != nil {
				return errors.Wrapf(err, "read file %s", path)
			}

			relpath, err := filepath.Rel(constants.UpstreamContentsPath, path)
			if err != nil {
				return errors.Wrapf(err, "find relative path to file %s", path)
			}

			encodedFile := base64.StdEncoding.EncodeToString(fileBytes)
			newUpstreamContents.UpstreamFiles = append(newUpstreamContents.UpstreamFiles, UpstreamFile{
				FilePath:     relpath,
				FileContents: encodedFile,
			})
			return nil
		})
		if err != nil {
			return state, errors.Wrapf(err, "walk files in %s", constants.UpstreamContentsPath)
		}

		if len(newUpstreamContents.UpstreamFiles) != 0 {
			// only replace upstreamContents in state if there are files read
			if state.V1 == nil {
				state.V1 = &V1{}
			}
			state.V1.UpstreamContents = newUpstreamContents
		}
	}

	return state, nil
}
