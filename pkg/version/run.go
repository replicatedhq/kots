package version

import (
	"time"
)

var RunAt time.Time

func init() {
	RunAt = time.Now().UTC()
	initBuild()
}
