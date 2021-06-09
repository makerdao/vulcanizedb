package utils_test

import (
	"errors"
	"time"

	"github.com/makerdao/vulcanizedb/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// start
// elapsedTime
// waitFor
type MockTimer struct {
	started     bool
	elapsedTime time.Duration
	sleepTime   time.Duration
}

func (timer *MockTimer) Start() {
	timer.started = true
}

func (timer MockTimer) Started() bool {
	return timer.started
}

func (timer MockTimer) ElapsedTime() time.Duration {
	return timer.elapsedTime
}

func (timer *MockTimer) WaitFor(sleepTime time.Duration) {
	timer.sleepTime = sleepTime
}

func (timer MockTimer) SleepTime() time.Duration {
	return timer.sleepTime
}

var _ = Describe("Throttler", func() {
	It("Passes through to the function passed in - and returns its error", func() {
		expectedError := errors.New("Test Error")
		mockTimer := MockTimer{}
		throttler := utils.NewThrottler(&mockTimer)
		called := false
		actualError := throttler.Throttle(0, func() error {
			called = true
			return expectedError
		})

		Expect(called).To(BeTrue())
		Expect(actualError).To(Equal(expectedError))
	})

	It("Sleeps for the minimumTime - elapsedTime using the Timer", func() {
		mockTimer := MockTimer{elapsedTime: 10}
		throttler := utils.NewThrottler(&mockTimer)

		throttler.Throttle(30, func() error { return nil })

		Expect(mockTimer.SleepTime()).To(Equal(time.Duration(20)))
	})

	It("requires calling Start before the callback, and WaitFor after", func() {
		mockTimer := MockTimer{elapsedTime: 10}
		throttler := utils.NewThrottler(&mockTimer)

		throttler.Throttle(30, func() error {
			Expect(mockTimer.Started()).To(BeTrue())
			Expect(mockTimer.SleepTime()).To(Equal(time.Duration(0)))
			return nil
		})

		Expect(mockTimer.SleepTime()).To(Equal(time.Duration(20)))
	})
})
