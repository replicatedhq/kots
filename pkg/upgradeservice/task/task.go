package task

import (
	"fmt"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/tasks"
)

type Status string

// CAUTION: modifying existing statuses will break backwards compatibility
const (
	StatusStarting         Status = "starting"
	StatusUpgradingCluster Status = "upgrading-cluster"
)

// CAUTION: modifying this task id will break backwards compatibility
func GetID(appSlug string) string {
	return fmt.Sprintf("upgrade-service-%s", appSlug)
}

func GetStatus(appSlug string) (string, string, error) {
	return tasks.GetTaskStatus(GetID(appSlug))
}

func SetStatusStarting(appSlug string, msg string) error {
	return tasks.SetTaskStatus(GetID(appSlug), msg, string(StatusStarting))
}

func SetStatusUpgradingCluster(appSlug string, msg string) error {
	return tasks.SetTaskStatus(GetID(appSlug), msg, string(StatusUpgradingCluster))
}

func IsStatusUpgradingCluster(appSlug string) (bool, error) {
	status, _, err := GetStatus(appSlug)
	if err != nil {
		return false, errors.Wrap(err, "failed to get status")
	}
	return status == string(StatusUpgradingCluster), nil
}
