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

package logs

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/makerdao/vulcanizedb/libraries/shared/constants"
	"github.com/makerdao/vulcanizedb/libraries/shared/factories/event"
	"github.com/makerdao/vulcanizedb/libraries/shared/fetcher"
	"github.com/makerdao/vulcanizedb/libraries/shared/transactions"
	"github.com/makerdao/vulcanizedb/pkg/core"
	"github.com/makerdao/vulcanizedb/pkg/datastore"
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres"
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres/repositories"
	"github.com/sirupsen/logrus"
)

var (
	ErrNoUncheckedHeaders = errors.New("no unchecked headers available for log fetching")
	ErrNoWatchedAddresses = errors.New("no watched addresses configured in the log extractor")
)

type ILogExtractor interface {
	AddTransformerConfig(config event.TransformerConfig) error
	ExtractLogs(recheckHeaders constants.TransformerExecution) error
}

type LogExtractor struct {
	Addresses                []common.Address
	CheckedHeadersRepository datastore.CheckedHeadersRepository
	CheckedLogsRepository    datastore.CheckedLogsRepository
	Fetcher                  fetcher.ILogFetcher
	LogRepository            datastore.EventLogRepository
	StartingBlock            *int64
	EndingBlock              *int64
	Syncer                   transactions.ITransactionsSyncer
	Topics                   []common.Hash
	RecheckHeaderCap         int64
}

func NewLogExtractor(db *postgres.DB, bc core.BlockChain) *LogExtractor {
	return &LogExtractor{
		CheckedHeadersRepository: repositories.NewCheckedHeadersRepository(db),
		CheckedLogsRepository:    repositories.NewCheckedLogsRepository(db),
		Fetcher:                  fetcher.NewLogFetcher(bc),
		LogRepository:            repositories.NewEventLogRepository(db),
		Syncer:                   transactions.NewTransactionsSyncer(db, bc),
		RecheckHeaderCap:         constants.RecheckHeaderCap,
	}
}

// Add additional logs to extract
func (extractor *LogExtractor) AddTransformerConfig(config event.TransformerConfig) error {
	checkedHeadersErr := extractor.updateCheckedHeaders(config)
	if checkedHeadersErr != nil {
		return checkedHeadersErr
	}

	if shouldResetStartingBlockToEarlierTransformerBlock(config.StartingBlockNumber, extractor.StartingBlock) {
		extractor.StartingBlock = &config.StartingBlockNumber
	}

	if shouldResetEndingBlockToLaterTransformerBlock(config.EndingBlockNumber, extractor.EndingBlock) {
		extractor.EndingBlock = &config.EndingBlockNumber
	}

	addresses := event.HexStringsToAddresses(config.ContractAddresses)
	extractor.Addresses = append(extractor.Addresses, addresses...)
	extractor.Topics = append(extractor.Topics, common.HexToHash(config.Topic))
	return nil
}

func shouldResetStartingBlockToEarlierTransformerBlock(currentTransformerBlock int64, extractorBlock *int64) bool {
	isExtractorBlockNil := extractorBlock == nil
	if isExtractorBlockNil {
		return true
	}

	isTransformerBlockLessThan := currentTransformerBlock < *extractorBlock
	return isTransformerBlockLessThan
}

func shouldResetEndingBlockToLaterTransformerBlock(currentTransformerBlock int64, extractorBlock *int64) bool {
	isExtractorBlockNil := extractorBlock == nil
	isTransformerBlockNegativeOne := currentTransformerBlock == int64(-1)

	if isExtractorBlockNil || isTransformerBlockNegativeOne {
		return true
	}

	isCurrentBlockNegativeOne := *extractorBlock != int64(-1)
	isTransformerBlockGreater := currentTransformerBlock > *extractorBlock

	return isCurrentBlockNegativeOne && isTransformerBlockGreater
}

// Fetch and persist watched logs
func (extractor LogExtractor) ExtractLogs(recheckHeaders constants.TransformerExecution) error {
	if len(extractor.Addresses) < 1 {
		logrus.Errorf("error extracting logs: %s", ErrNoWatchedAddresses.Error())
		return ErrNoWatchedAddresses
	}

	uncheckedHeaders, uncheckedHeadersErr := extractor.CheckedHeadersRepository.UncheckedHeaders(*extractor.StartingBlock, *extractor.EndingBlock, extractor.getCheckCount(recheckHeaders))
	if uncheckedHeadersErr != nil {
		logrus.Errorf("error fetching missing headers: %s", uncheckedHeadersErr)
		return uncheckedHeadersErr
	}

	if len(uncheckedHeaders) < 1 {
		return ErrNoUncheckedHeaders
	}

	for _, header := range uncheckedHeaders {
		logs, fetchLogsErr := extractor.Fetcher.FetchLogs(extractor.Addresses, extractor.Topics, header)
		if fetchLogsErr != nil {
			logError("error fetching logs for header: %s", fetchLogsErr, header)
			return fetchLogsErr
		}

		if len(logs) > 0 {
			transactionsSyncErr := extractor.Syncer.SyncTransactions(header.Id, logs)
			if transactionsSyncErr != nil {
				logError("error syncing transactions: %s", transactionsSyncErr, header)
				return transactionsSyncErr
			}

			createLogsErr := extractor.LogRepository.CreateEventLogs(header.Id, logs)
			if createLogsErr != nil {
				logError("error persisting logs: %s", createLogsErr, header)
				return createLogsErr
			}
		}

		markHeaderCheckedErr := extractor.CheckedHeadersRepository.MarkHeaderChecked(header.Id)
		if markHeaderCheckedErr != nil {
			logError("error marking header checked: %s", markHeaderCheckedErr, header)
			return markHeaderCheckedErr
		}
	}
	return nil
}

func logError(description string, err error, header core.Header) {
	logrus.WithFields(logrus.Fields{
		"headerId":    header.Id,
		"headerHash":  header.Hash,
		"blockNumber": header.BlockNumber,
	}).Errorf(description, err.Error())
}

func (extractor *LogExtractor) getCheckCount(recheckHeaders constants.TransformerExecution) int64 {
	if recheckHeaders == constants.HeaderUnchecked {
		return 1
	} else {
		return extractor.RecheckHeaderCap
	}
}

func (extractor *LogExtractor) updateCheckedHeaders(config event.TransformerConfig) error {
	alreadyWatchingLog, watchingLogErr := extractor.CheckedLogsRepository.AlreadyWatchingLog(config.ContractAddresses, config.Topic)
	if watchingLogErr != nil {
		return watchingLogErr
	}
	if !alreadyWatchingLog {
		uncheckHeadersErr := extractor.CheckedHeadersRepository.MarkHeadersUncheckedSince(config.StartingBlockNumber)
		if uncheckHeadersErr != nil {
			return uncheckHeadersErr
		}
		markLogWatchedErr := extractor.CheckedLogsRepository.MarkLogWatched(config.ContractAddresses, config.Topic)
		if markLogWatchedErr != nil {
			return markLogWatchedErr
		}
	}
	return nil
}
