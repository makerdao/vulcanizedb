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

package watcher

import "time"

type Callback func() error
type ThrottlerFunc func(time.Duration, Callback) error

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

func (throttler Throttler) Throttle(minTime time.Duration, f Callback) error {
	throttler.timer.Start()
	err := f()
	throttler.timer.WaitFor(minTime - throttler.timer.ElapsedTime())
	return err
}
