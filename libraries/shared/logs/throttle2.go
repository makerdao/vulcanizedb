package logs

import (
	"time"

	"github.com/makerdao/vulcanizedb/pkg/core"
)

type Callback func(header core.Header) error
type ThrottlerFunc func(time.Duration, Callback, core.Header) error

type Timer interface {
	WaitFor(sleep time.Duration)
	ElapsedTime() time.Duration
	Start()
}

type StandardTimer struct {
	start time.Time
}

func (timer StandardTimer) WaitFor(sleepTime time.Duration) {
	time.Sleep(sleepTime)
}

func (timer StandardTimer) ElapsedTime() time.Duration {
	t := time.Now()
	return t.Sub(timer.start)
}

func (timer *StandardTimer) Start() {
	timer.start = time.Now()
}

type Throttler struct {
	timer Timer
}

func NewThrottler(timer Timer) Throttler {
	return Throttler{
		timer: timer,
	}
}

func (throttler Throttler) Throttle(minTime time.Duration, f Callback, header core.Header) error {
	throttler.timer.Start()
	err := f(header)
	throttler.timer.WaitFor(minTime - throttler.timer.ElapsedTime())
	return err
}
