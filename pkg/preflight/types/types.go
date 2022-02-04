package types

import (
	"time"

	troubleshootpreflight "github.com/replicatedhq/troubleshoot/pkg/preflight"
)

type PreflightResult struct {
	Result      string     `json:"result"`
	CreatedAt   *time.Time `json:"createdAt"`
	AppSlug     string     `json:"appSlug"`
	ClusterSlug string     `json:"clusterSlug"`
	Skipped     bool       `json:"skipped"`
}

type PreflightProgress struct {
	CompletedCount int                                              `json:"completedCount"`
	TotalCount     int                                              `json:"totalCount"`
	CurrentName    string                                           `json:"currentName"`
	CurrentStatus  string                                           `json:"currentStatus"`
	UpdatedAt      string                                           `json:"updatedAt"`
	Preflights     map[string]troubleshootpreflight.CollectorStatus `json:"preflights"`
}
