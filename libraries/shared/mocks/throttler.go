package mocks

import (
	"time"

	"github.com/makerdao/vulcanizedb/utils"
)

type MockThrottler struct {
	SleepTime  time.Duration
	calledWith utils.Callback
}

func (throttler *MockThrottler) Throttle(sleep time.Duration, f utils.Callback) error {
	throttler.SleepTime = sleep
	throttler.calledWith = f
	return f()
}
