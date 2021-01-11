// VulcanizeDB
// Copyright © 2021 Vulcanize

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.

// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package watcher_test

import (
	"errors"
	"time"

	"github.com/makerdao/vulcanizedb/libraries/shared/watcher"
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
		throttler := watcher.NewThrottler(&mockTimer)
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
		throttler := watcher.NewThrottler(&mockTimer)

		throttler.Throttle(30, func() error { return nil })

		Expect(mockTimer.SleepTime()).To(Equal(time.Duration(20)))
	})

	It("requires calling Start before the callback, and WaitFor after", func() {
		mockTimer := MockTimer{elapsedTime: 10}
		throttler := watcher.NewThrottler(&mockTimer)

		throttler.Throttle(30, func() error {
			Expect(mockTimer.Started()).To(BeTrue())
			Expect(mockTimer.SleepTime()).To(Equal(time.Duration(0)))
			return nil
		})

		Expect(mockTimer.SleepTime()).To(Equal(time.Duration(20)))
	})
})
