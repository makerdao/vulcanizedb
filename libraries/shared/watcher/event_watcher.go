// VulcanizeDB
// Copyright © 2019 Vulcanize

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

import (
	"fmt"
	"io"
	"time"

	"github.com/makerdao/vulcanizedb/libraries/shared/constants"
	"github.com/makerdao/vulcanizedb/libraries/shared/factories/event"
	"github.com/makerdao/vulcanizedb/libraries/shared/logs"
	"github.com/makerdao/vulcanizedb/pkg/core"
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres"
	"github.com/makerdao/vulcanizedb/pkg/fs"
	"github.com/sirupsen/logrus"
)

type EventWatcher struct {
	blockChain                   core.BlockChain
	db                           *postgres.DB
	LogExtractor                 logs.ILogExtractor
	ExpectedExtractorError       error
	LogDelegator                 logs.ILogDelegator
	ExpectedDelegatorError       error
	MaxConsecutiveUnexpectedErrs int
	RetryInterval                time.Duration
	StatusWriter                 fs.StatusWriter
}

func NewEventWatcher(db *postgres.DB, bc core.BlockChain, extractor logs.ILogExtractor, delegator logs.ILogDelegator, maxConsecutiveUnexpectedErrs int, retryInterval time.Duration, statusWriter fs.StatusWriter) EventWatcher {
	return EventWatcher{
		blockChain:                   bc,
		db:                           db,
		LogExtractor:                 extractor,
		ExpectedExtractorError:       logs.ErrNoUncheckedHeaders,
		LogDelegator:                 delegator,
		ExpectedDelegatorError:       logs.ErrNoLogs,
		MaxConsecutiveUnexpectedErrs: maxConsecutiveUnexpectedErrs,
		RetryInterval:                retryInterval,
		StatusWriter:                 statusWriter,
	}
}

// Adds transformers to the watcher so that their logs will be extracted and delegated.
func (watcher *EventWatcher) AddTransformers(initializers []event.TransformerInitializer) error {
	for _, initializer := range initializers {
		t := initializer(watcher.db)

		watcher.LogDelegator.AddTransformer(t)
		err := watcher.LogExtractor.AddTransformerConfig(t.GetConfig())
		if err != nil {
			return err
		}
	}
	return nil
}

// Extracts and delegates watched log events.
func (watcher *EventWatcher) Execute(recheckHeaders constants.TransformerExecution) error {
	writeErr := watcher.StatusWriter.Write()
	if writeErr != nil {
		return fmt.Errorf("error confirming health check: %w", writeErr)
	}

	//only writers should close channels
	delegateErrsChan := make(chan error)
	extractErrsChan := make(chan error)
	executeQuitChan := make(chan bool)

	go watcher.extractLogs(recheckHeaders, extractErrsChan, executeQuitChan)
	go watcher.delegateLogs(delegateErrsChan, executeQuitChan)

	for {
		select {
		case delegateErr := <-delegateErrsChan:
			logrus.Warnf("error delegating logs in event watcher: %s", delegateErr.Error())
			close(executeQuitChan)
			return delegateErr
		case extractErr := <-extractErrsChan:
			logrus.Warnf("error extracting logs in event watcher: %s", extractErr.Error())
			close(executeQuitChan)
			return extractErr
		}
	}
}

func (watcher *EventWatcher) extractLogs(recheckHeaders constants.TransformerExecution, errs chan error, quitChan chan bool) {
	call := func() error { return watcher.LogExtractor.ExtractLogs(recheckHeaders) }
	// io.ErrUnexpectedEOF errors are sometimes returned from fetching logs at the head of the chain when fetching from an uncle or fork block
	expectedErrors := []error{watcher.ExpectedExtractorError, io.ErrUnexpectedEOF}
	watcher.withRetry(call, expectedErrors, "extracting", errs, quitChan)
}

func (watcher *EventWatcher) delegateLogs(errs chan error, quitChan chan bool) {
	call := func() error { return watcher.LogDelegator.DelegateLogs(ResultsLimit) }
	watcher.withRetry(call, []error{watcher.ExpectedDelegatorError}, "delegating", errs, quitChan)
}

func (watcher *EventWatcher) withRetry(call func() error, expectedErrors []error, operation string, errs chan error, quitChan chan bool) {
	defer close(errs)
	consecutiveUnexpectedErrCount := 0
	for {
		select {
		case <-quitChan:
			return
		default:
			err := call()
			if err == nil {
				consecutiveUnexpectedErrCount = 0
			} else {
				if isUnexpectedError(err, expectedErrors) {
					consecutiveUnexpectedErrCount++
					if consecutiveUnexpectedErrCount > watcher.MaxConsecutiveUnexpectedErrs {
						errs <- err
						return
					}
				}
				time.Sleep(watcher.RetryInterval)
			}
		}
	}
}

func isUnexpectedError(currentError error, expectedErrors []error) bool {
	for _, expectedError := range expectedErrors {
		if currentError == expectedError {
			return false
		}
	}

	return true
}
