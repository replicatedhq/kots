package ocistore

import (
	"github.com/pkg/errors"
	installationtypes "github.com/replicatedhq/kots/kotsadm/pkg/online/types"
)

const (
	PendingInstallationsConfigMapName = "kotsadm-pendinginstallation"
)

func (s OCIStore) GetPendingInstallationStatus() (*installationtypes.InstallStatus, error) {
	apps, err := s.ListInstalledApps()
	if err != nil {
		return nil, errors.Wrap(err, "failed to list installed apps")
	}

	if len(apps) == 0 {
		return &installationtypes.InstallStatus{
			InstallStatus:  "not_installed",
			CurrentMessage: "",
		}, nil
	}

	app := apps[0]

	_, message, err := s.GetTaskStatus("online-install")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get task status")
	}

	status := &installationtypes.InstallStatus{
		InstallStatus:  app.InstallState,
		CurrentMessage: message,
	}

	return status, nil
}
