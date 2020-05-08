package util

import (
	"sync"
	"time"
)

func NewThrottle(after time.Duration) func(f func()) {
	t := &throttler{after: after}

	return func(f func()) {
		t.add(f)
	}
}

type throttler struct {
	mu    sync.Mutex
	after time.Duration
	f     func()
	timer *time.Timer
}

func (t *throttler) add(f func()) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.f = f

	if t.timer == nil {
		t.timer = time.NewTimer(t.after)
		go func() {
			<-t.timer.C
			t.mu.Lock()
			defer t.mu.Unlock()
			t.f()
			t.timer = nil
		}()
	}
}
