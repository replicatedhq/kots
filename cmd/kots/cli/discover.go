package cli

import (
	"github.com/pkg/errors"
	ps "github.com/shirou/gopsutil/host"
)

func discover() (string, string, error) {

	var dist, version string
	dist, _, version, err := ps.PlatformInformation()
	if err != nil {
		return dist, version, errors.Wrap(err, "unable to detect the platform")
	}
	return dist, version, nil
}
