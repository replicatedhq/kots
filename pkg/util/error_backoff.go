package util

import (
	"sync"
	"time"
)

type ErrorBackoff struct {
	MinPeriod time.Duration
	MaxPeriod time.Duration

	period            time.Duration
	lastError         error
	lastErrorTimeLock sync.Mutex
	lastErrorTime     time.Time
}

func (r *ErrorBackoff) OnError(err error, fn func()) {
	r.lastErrorTimeLock.Lock()
	defer r.lastErrorTimeLock.Unlock()
	if r.lastError != nil && r.lastError.Error() == err.Error() {
		d := time.Since(r.lastErrorTime)
		if d < r.period {
			return
		}
		r.period = r.period * 2
		if r.period > r.MaxPeriod {
			r.period = r.MaxPeriod
		}
	} else {
		r.period = r.MinPeriod
		r.lastError = err
		r.lastErrorTime = time.Now()
	}
	fn()
}
