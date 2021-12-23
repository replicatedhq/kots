package upstream

import (
	"io/ioutil"
	"path"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
)

func SaveInstallation(installation *kotsv1beta1.Installation, upstreamDir string) error {
	filename := path.Join(upstreamDir, "userdata", "installation.yaml")
	err := ioutil.WriteFile(filename, mustMarshalInstallation(installation), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write installation")
	}
	return nil
}
