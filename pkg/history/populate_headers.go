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

package history

import (
	"fmt"

	"github.com/makerdao/vulcanizedb/pkg/core"
	"github.com/makerdao/vulcanizedb/pkg/datastore"
	"github.com/sirupsen/logrus"
)

func PopulateMissingHeaders(blockChain core.BlockChain, headerRepository datastore.HeaderRepository, startingBlockNumber int64) error {
	lastBlock, lastBlockErr := blockChain.LastBlock()
	if lastBlockErr != nil {
		return fmt.Errorf("error getting client's last block: %w", lastBlockErr)
	}

	blockNumbers, missingBlocksErr := headerRepository.MissingBlockNumbers(startingBlockNumber, lastBlock.Int64())
	if missingBlocksErr != nil {
		return fmt.Errorf("error getting missing block numbers: %w", missingBlocksErr)
	}
	if len(blockNumbers) == 0 {
		return nil
	}

	logrus.Debug(getBlockRangeString(blockNumbers))
	updateErr := RetrieveAndUpdateHeaders(blockChain, headerRepository, blockNumbers)
	if updateErr != nil {
		return fmt.Errorf("error getting/updating headers: %w", updateErr)
	}
	return nil
}

func RetrieveAndUpdateHeaders(blockChain core.BlockChain, headerRepository datastore.HeaderRepository, blockNumbers []int64) error {
	headers, err := blockChain.GetHeadersByNumbers(blockNumbers)
	for _, header := range headers {
		_, err = headerRepository.CreateOrUpdateHeader(header)
		if err != nil {
			return err
		}
	}
	return nil
}

func getBlockRangeString(blockRange []int64) string {
	return fmt.Sprintf("Backfilling |%v| blocks", len(blockRange))
}
