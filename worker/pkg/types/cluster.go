package types

import (
	"time"
)

type Cluster struct {
	ID        string
	Type      string
	CreatedAt time.Time

	GitHubOwner          string
	GitHubRepo           string
	GitHubBranch         string
	GitHubInstallationID int
}
