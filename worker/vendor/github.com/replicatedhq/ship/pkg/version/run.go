package version

import (
	"fmt"
	"time"
)

var RunAt time.Time
var RunAtEpoch string

func init() {
	RunAt = time.Now()
	RunAtEpoch = fmt.Sprintf("%d", RunAt.Unix())
}
