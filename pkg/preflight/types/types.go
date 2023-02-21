package types

import (
	"time"

	"github.com/replicatedhq/troubleshoot/pkg/preflight"
)

type PreflightResult struct {
	Result                     string     `json:"result"`
	CreatedAt                  *time.Time `json:"createdAt"`
	AppSlug                    string     `json:"appSlug"`
	ClusterSlug                string     `json:"clusterSlug"`
	Skipped                    bool       `json:"skipped"`
	HasFailingStrictPreflights bool       `json:"hasFailingStrictPreflights"`
}

type PreflightError struct {
	IsRBAC bool   `json:"isRbac"`
	Error  string `json:"error"`
}

type PreflightResults struct {
	Results []*preflight.UploadPreflightResult `json:"results,omitempty"`
	Errors  []*PreflightError                  `json:"errors,omitempty"`
}
