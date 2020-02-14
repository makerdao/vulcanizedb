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

package mocks

import (
	"github.com/makerdao/vulcanizedb/libraries/shared/constants"
	"github.com/makerdao/vulcanizedb/libraries/shared/factories/event"
	"github.com/makerdao/vulcanizedb/libraries/shared/logs"
)

type MockLogExtractor struct {
	AddedConfigs              []event.TransformerConfig
	AddTransformerConfigError error
	ExtractLogsCount          int
	ExtractLogsErrors         []error
}

func (extractor *MockLogExtractor) AddTransformerConfig(config event.TransformerConfig) error {
	extractor.AddedConfigs = append(extractor.AddedConfigs, config)
	return extractor.AddTransformerConfigError
}

func (extractor *MockLogExtractor) ExtractLogs(recheckHeaders constants.TransformerExecution) error {
	extractor.ExtractLogsCount++
	if len(extractor.ExtractLogsErrors) > 1 {
		var errorThisRun error
		errorThisRun, extractor.ExtractLogsErrors = extractor.ExtractLogsErrors[0], extractor.ExtractLogsErrors[1:]
		return errorThisRun
	} else if len(extractor.ExtractLogsErrors) == 1 {
		thisErr := extractor.ExtractLogsErrors[0]
		extractor.ExtractLogsErrors = []error{}
		return thisErr
	}
	// return no unchecked headers error so that extractor hits retry interval when delegator under test
	return logs.ErrNoUncheckedHeaders
}
