package types

import (
	"time"
)

type WatchVersion struct {
	WatchID           string
	CreatedAt         time.Time
	VersionLabel      string
	Status            string
	SourceBranch      string
	Sequence          int
	PullRequestNumber int
}
