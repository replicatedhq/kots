package types

import (
	"time"
)

type UpgradeOptions struct {
	Namespace                 string
	ForceUpgradeKurl          bool
	Timeout                   time.Duration
	EnsureRBAC                bool
	SimultaneousUploads       int
	StorageBaseURI            string
	StorageBaseURIPlainHTTP   bool
	IncludeMinio              bool
	IncludeDockerDistribution bool

	KotsadmOptions KotsadmOptions
}
