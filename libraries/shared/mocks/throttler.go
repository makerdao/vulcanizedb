package mocks

import (
	"time"

	"github.com/makerdao/vulcanizedb/libraries/shared/watcher"
)

type MockThrottler struct {
	SleepTime  time.Duration
	calledWith watcher.Callback
}

func (throttler *MockThrottler) Throttle(sleep time.Duration, f watcher.Callback) error {
	throttler.SleepTime = sleep
	throttler.calledWith = f
	return f()
}
