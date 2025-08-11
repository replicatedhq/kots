package types

import (
	"time"
)

type UpgradeOptions struct {
	Namespace             string
	ForceUpgradeKurl      bool
	Timeout               time.Duration
	EnsureRBAC            bool
	StrictSecurityContext *bool
	SimultaneousUploads   int
	IncludeMinio          bool
	NodeSelector          map[string]string

	RegistryConfig RegistryConfig
}
