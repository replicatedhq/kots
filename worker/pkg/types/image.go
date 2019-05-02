package types

import (
	"time"
)

type ImageCheck struct {
	ID                string
	Name              string
	CheckedAt         time.Time
	IsPrivate         bool
	VersionsBehind    int64
	DetectedVersion   string
	LatestVersion     string
	CompatibleVersion string
	CheckError        string
	Path              string
}
