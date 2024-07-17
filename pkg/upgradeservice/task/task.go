package task

import (
	"fmt"

	"github.com/replicatedhq/kots/pkg/tasks"
)

type Status string

// CAUTION: modifying existing statuses will break backwards compatibility
const (
	StatusStarting         Status = "starting"
	StatusUpgradingCluster Status = "upgrading-cluster"
	StatusUpgradingApp     Status = "upgrading-app"
	StatusUpgradeFailed    Status = "upgrade-failed"
)

// CAUTION: modifying this task id will break backwards compatibility
func GetID(appSlug string) string {
	return fmt.Sprintf("upgrade-service-%s", appSlug)
}

func GetStatus(appSlug string) (string, string, error) {
	return tasks.GetTaskStatus(GetID(appSlug))
}

func ClearStatus(appSlug string) error {
	return tasks.ClearTaskStatus(GetID(appSlug))
}

func SetStatusStarting(appSlug string, msg string) error {
	return tasks.SetTaskStatus(GetID(appSlug), msg, string(StatusStarting))
}

func SetStatusUpgradingCluster(appSlug string, msg string) error {
	return tasks.SetTaskStatus(GetID(appSlug), msg, string(StatusUpgradingCluster))
}

func SetStatusUpgradingApp(appSlug string, msg string) error {
	return tasks.SetTaskStatus(GetID(appSlug), msg, string(StatusUpgradingApp))
}

func SetStatusUpgradeFailed(appSlug string, msg string) error {
	return tasks.SetTaskStatus(GetID(appSlug), msg, string(StatusUpgradeFailed))
}
