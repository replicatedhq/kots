package types

import (
	"time"
)

type UpgradeOptions struct {
	Namespace           string
	ForceUpgradeKurl    bool
	Timeout             time.Duration
	EnsureRBAC          bool
	SimultaneousUploads int

	KotsadmOptions KotsadmOptions
}
