package apparchive

import (
	"os"
	"path"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

func SaveInstallation(installation *kotsv1beta1.Installation, upstreamDir string) error {
	filename := path.Join(upstreamDir, "userdata", "installation.yaml")
	err := os.WriteFile(filename, kotsutil.MustMarshalInstallation(installation), 0644)
	if err != nil {
		return errors.Wrap(err, "failed to write installation")
	}
	return nil
}
