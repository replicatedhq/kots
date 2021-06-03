package types

import "time"

type PreflightResult struct {
	Result      string     `json:"result"`
	CreatedAt   *time.Time `json:"createdAt"`
	AppSlug     string     `json:"appSlug"`
	ClusterSlug string     `json:"clusterSlug"`
	Skipped     bool       `json:"skipped"`
}
