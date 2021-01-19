package history

import (
	"fmt"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage/types"
	"github.com/makerdao/vulcanizedb/pkg/core"
)

type StorageDiffValidator struct {
	blockChain            core.BlockChain
	storageDiffRepository storage.DiffRepository
	windowSize            int
}

func NewStorageDiffValidator(blockChain core.BlockChain, diffRespository storage.DiffRepository, windowSize int) StorageDiffValidator {
	return StorageDiffValidator{
		blockChain:            blockChain,
		storageDiffRepository: diffRespository,
		windowSize:            windowSize,
	}
}

type ByBlockNumber []core.Header

func (a ByBlockNumber) Len() int           { return len(a) }
func (a ByBlockNumber) Less(i, j int) bool { return a[i].BlockNumber < a[j].BlockNumber }
func (a ByBlockNumber) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func (v StorageDiffValidator) ValidateDiffs() error {
	window, makeWindowErr := MakeValidationWindow(v.blockChain, v.windowSize)
	log.Info("Validation window: ", window)
	if makeWindowErr != nil {
		return fmt.Errorf("error creating validation window: %s", makeWindowErr.Error())
	}
	blockNumbers := MakeRange(window.LowerBound, window.UpperBound)
	headers, getHeadersErr := v.blockChain.GetHeadersByNumbers(blockNumbers)
	if getHeadersErr != nil {
		return getHeadersErr
	}
	sort.Sort(ByBlockNumber(headers))

	startingBlockNumber := headers[0].BlockNumber
	endingBlockNumber := headers[len(headers)-1].BlockNumber
	diffs, getDiffsErr := v.storageDiffRepository.GetDiffsForBlockHeightRange(startingBlockNumber, endingBlockNumber)
	if getDiffsErr != nil {
		return getDiffsErr
	}

	markPendingErr := v.markNoncanonicalDiffsPending(diffs, headers)
	if markPendingErr != nil {
		return markPendingErr
	}

	return nil
}

func (v StorageDiffValidator) markNoncanonicalDiffsPending(diffs []types.PersistedDiff, headers []core.Header) error {
	for _, diff := range diffs {
		header := getHeaderByBlockNumber(int64(diff.BlockHeight), headers)
		if !isDiffCanonical(diff, header) && isDiffStatusTransformed(diff) {
			logMessage := fmt.Sprintf("Diff %d from block %d is being marked PENDING because it is noncanonical", diff.ID, diff.BlockHeight)
			log.Info(logMessage)
			err := v.storageDiffRepository.MarkPending(diff.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getHeaderByBlockNumber(blockNumber int64, headers []core.Header) core.Header {
	for _, header := range headers {
		if header.BlockNumber == blockNumber {
			return header
		}
	}

	return core.Header{}
}

func isDiffStatusTransformed(diff types.PersistedDiff) bool {
	return diff.Status == storage.Transformed
}

func isDiffCanonical(diff types.PersistedDiff, header core.Header) bool {
	return diff.BlockHash == common.HexToHash(header.Hash)
}
