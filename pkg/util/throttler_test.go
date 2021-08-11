package util

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestThrottle(t *testing.T) {
	var counter1 uint64

	f1 := func() {
		atomic.AddUint64(&counter1, 1)
	}
	f2 := func() {
		atomic.AddUint64(&counter1, 2)
	}

	throttled := NewThrottle(100 * time.Millisecond)

	for i := 0; i < 3; i++ {
		for j := 0; j < 10; j++ {
			throttled(f1)
			throttled(f2)
		}

		time.Sleep(200 * time.Millisecond)
	}

	c1 := int(atomic.LoadUint64(&counter1))
	if c1 != 6 {
		t.Error("Expected count 6, was", c1)
	}
}

func TestThrottleConcurrent(t *testing.T) {
	var counter1 uint64

	f2 := func() {
		atomic.AddUint64(&counter1, 2)
	}

	throttled := NewThrottle(100 * time.Millisecond)

	for i := 0; i < 3; i++ {
		for j := 0; j < 10; j++ {
			go throttled(f2)
		}

		time.Sleep(200 * time.Millisecond)
	}

	c1 := int(atomic.LoadUint64(&counter1))
	if c1 != 6 {
		t.Error("Expected count 6, was", c1)
	}
}
