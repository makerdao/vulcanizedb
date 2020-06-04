// Copyright 2019 Vulcanize
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package fetcher

import (
	"fmt"

	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage/types"
	"github.com/makerdao/vulcanizedb/libraries/shared/streamer"
	"github.com/makerdao/vulcanizedb/libraries/shared/test_data/old_geth_patches"
	"github.com/makerdao/vulcanizedb/pkg/fs"
	"github.com/sirupsen/logrus"
)

type GethRpcStorageFetcher struct {
	statediffPayloadChan chan filters.Payload
	streamer             streamer.Streamer
	gethVersion          GethPatchVersion
	statusWriter         fs.StatusWriter
}

func NewGethRpcStorageFetcher(streamer streamer.Streamer, statediffPayloadChan chan filters.Payload, gethVersion GethPatchVersion, statusWriter fs.StatusWriter) GethRpcStorageFetcher {
	return GethRpcStorageFetcher{
		statediffPayloadChan: statediffPayloadChan,
		streamer:             streamer,
		gethVersion:          gethVersion,
		statusWriter:         statusWriter,
	}
}

func (fetcher GethRpcStorageFetcher) FetchStorageDiffs(out chan<- types.RawDiff, errs chan<- error) {
	ethStatediffPayloadChan := fetcher.statediffPayloadChan
	clientSubscription, clientSubErr := fetcher.streamer.Stream(ethStatediffPayloadChan)
	if clientSubErr != nil {
		errs <- clientSubErr
		panic(fmt.Sprintf("Error creating a geth client subscription: %v", clientSubErr))
	}
	logrus.Info("Successfully created a geth client subscription: ", clientSubscription)

	writeErr := fetcher.statusWriter.Write()
	if writeErr != nil {
		errs <- writeErr
	}

	for {
		select {
		case err := <-clientSubscription.Err():
			logrus.Errorf("error with client subscription: %s", err.Error())
			errs <- err
		case diffPayload := <-ethStatediffPayloadChan:
			logrus.Trace("received a statediff payload")
			stateDiff, decodeErr := fetcher.decodeStateDiffRlpFromPayload(diffPayload)
			if decodeErr != nil {
				logrus.Warn("Error decoding state diff into RLP: ", decodeErr)
				errs <- decodeErr
			}
			logrus.Tracef("received a statediff from block: %v", stateDiff.BlockNumber)

			logrus.Trace(fmt.Sprintf("iterating through %d accounts on stateDiff for block %d", len(stateDiff.UpdatedAccounts), stateDiff.BlockNumber))
			for _, account := range stateDiff.UpdatedAccounts {
				logrus.Trace(fmt.Sprintf("iterating through %d Storage values on account", len(account.Storage)))
				for _, accountStorage := range account.Storage {
					diff, formatErr := fetcher.formatDiff(account, stateDiff, accountStorage)
					logrus.Tracef("adding storage diff to out channel. keccak of address: %v, block height: %v, storage key: %v, storage value: %v",
						diff.HashedAddress.Hex(), diff.BlockHeight, diff.StorageKey.Hex(), diff.StorageValue.Hex())
					if formatErr != nil {
						errs <- formatErr
					}

					out <- diff
				}
			}
		}

	}
}

func (fetcher GethRpcStorageFetcher) formatDiff(account filters.AccountDiff, stateDiff *filters.StateDiff, storage filters.StorageDiff) (types.RawDiff, error) {
	if fetcher.gethVersion == OldGethPatch {
		return types.FromOldGethStateDiff(account, stateDiff, storage)
	} else {
		return types.FromNewGethStateDiff(account, stateDiff, storage)
	}
}

func (fetcher GethRpcStorageFetcher) decodeStateDiffRlpFromPayload(payload filters.Payload) (*filters.StateDiff, error) {
	if fetcher.gethVersion == NewGethPatchWithFilter {
		var stateDiff filters.StateDiff
		decodeErr := rlp.DecodeBytes(payload.StateDiffRlp, &stateDiff)
		if decodeErr != nil {
			return &filters.StateDiff{}, decodeErr
		}
		return &stateDiff, nil
	} else {
		return fetcher.decodeStateDiffRlpFromPayloadForOldPatches(payload)
	}
}

func (fetcher GethRpcStorageFetcher) decodeStateDiffRlpFromPayloadForOldPatches(payload filters.Payload) (*filters.StateDiff, error) {
	oldPatchStateDiff := new(old_geth_patches.StateDiffOldPatch)
	decodeErr := rlp.DecodeBytes(payload.StateDiffRlp, oldPatchStateDiff)
	if decodeErr != nil {
		return &filters.StateDiff{}, decodeErr
	}
	stateDiff := convertToNewStateDiff(*oldPatchStateDiff)
	return &stateDiff, nil
}

func getAccountsFromDiff(stateDiff filters.StateDiff) []filters.AccountDiff {
	return stateDiff.UpdatedAccounts
}

func convertToNewStateDiff(oldStateDiff old_geth_patches.StateDiffOldPatch) filters.StateDiff {
	accounts := append(oldStateDiff.UpdatedAccounts, oldStateDiff.CreatedAccounts...)
	accounts = append(accounts, oldStateDiff.DeletedAccounts...)
	convertedAccounts := convertToNewAccountDiff(accounts)
	return filters.StateDiff{
		BlockNumber:     oldStateDiff.BlockNumber,
		BlockHash:       oldStateDiff.BlockHash,
		UpdatedAccounts: convertedAccounts,
	}
}

func convertToNewAccountDiff(oldAccountDiffs []old_geth_patches.AccountDiffOldPatch) []filters.AccountDiff {
	var accounts []filters.AccountDiff
	for _, account := range oldAccountDiffs {
		newStorage := convertToNewStorageDiff(account.Storage)
		newAccount := filters.AccountDiff{
			Key:     account.Key,
			Value:   account.Value,
			Storage: newStorage,
		}
		accounts = append(accounts, newAccount)

	}
	return accounts
}

func convertToNewStorageDiff(oldStorageDiffs []old_geth_patches.StorageDiffOldPatch) []filters.StorageDiff {
	var storageDiffs []filters.StorageDiff
	for _, storage := range oldStorageDiffs {
		newStorage := filters.StorageDiff{
			Key:   storage.Key,
			Value: storage.Value,
		}
		storageDiffs = append(storageDiffs, newStorage)
	}
	return storageDiffs
}
