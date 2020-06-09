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
	"io/ioutil"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
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

var (
	processingDiffsLogString = "processing %d storage diffs for account %s using %s"
	addingDiffsLogString = "adding storage diff to out channel. keccak of address: %v, block height: %v, storage key: %v, storage value: %v"
)
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

			switch fetcher.gethVersion {
			case OldGethPatch:
				fetcher.handleDiffsFromOldGethPatch(diffPayload, out, errs)
			case NewGethPatchWithService:
				fetcher.handleDiffsFromNewGethPatchWithService(diffPayload, out, errs)
			case NewGethPatchWithFilter:
				fetcher.handleDiffsFromNewGethPatchWithFilter(diffPayload, out, errs)
			}
		}
	}
}

func (fetcher GethRpcStorageFetcher) handleDiffsFromOldGethPatch(payload filters.Payload, out chan<- types.RawDiff, errs chan<- error) {
	stateDiff, decodeErr := fetcher.decodeStateDiffRlpFromPayloadForOldPatches(payload)
	if decodeErr != nil {
		errs <- fmt.Errorf("error decoding storage diff from %s payload: %w", fetcher.gethVersion, decodeErr)
		return
	}

	for _, account := range stateDiff.UpdatedAccounts {
		logrus.Info(fmt.Sprintf(processingDiffsLogString, len(account.Storage), common.Bytes2Hex(account.Key), fetcher.gethVersion))
		for _, accountStorage := range account.Storage {
			rawDiff, formatErr := types.FromOldGethStateDiff(account, stateDiff, accountStorage)
			if formatErr != nil {
				errs <- formatErr
			}

			logrus.Tracef(addingDiffsLogString, rawDiff.HashedAddress.Hex(), rawDiff.BlockHeight, rawDiff.StorageKey.Hex(), rawDiff.StorageValue.Hex())
			out <- rawDiff
		}
	}
}

func (fetcher GethRpcStorageFetcher) handleDiffsFromNewGethPatchWithService(payload filters.Payload, out chan<- types.RawDiff, errs chan<- error) {
	stateDiff, decodeErr := fetcher.decodeStateDiffRlpFromPayloadForOldPatches(payload)
	if decodeErr != nil {
		errs <- fmt.Errorf("error decoding storage diff from %s payload: %w", fetcher.gethVersion, decodeErr)
		return
	}

	for _, account := range stateDiff.UpdatedAccounts {
		logrus.Info(fmt.Sprintf(processingDiffsLogString, len(account.Storage), common.Bytes2Hex(account.Key), fetcher.gethVersion))
		for _, accountStorage := range account.Storage {
			rawDiff, formatErr := types.FromNewGethStateDiff(account, stateDiff, accountStorage)
			if formatErr != nil {
				errs <- formatErr
			}

			logrus.Tracef(addingDiffsLogString, rawDiff.HashedAddress.Hex(), rawDiff.BlockHeight, rawDiff.StorageKey.Hex(), rawDiff.StorageValue.Hex())
			out <- rawDiff
		}
	}
}

func (fetcher GethRpcStorageFetcher) handleDiffsFromNewGethPatchWithFilter(payload filters.Payload, out chan<- types.RawDiff, errs chan<- error) {
	var stateDiff filters.StateDiff
	decodeErr := rlp.DecodeBytes(payload.StateDiffRlp, &stateDiff)
	if decodeErr != nil {
		errs <- fmt.Errorf("error decoding storage diff from %s payload: %w", fetcher.gethVersion, decodeErr)
		return
	}

	for _, account := range stateDiff.UpdatedAccounts {
		logrus.Info(fmt.Sprintf(processingDiffsLogString, len(account.Storage), common.Bytes2Hex(account.Key), fetcher.gethVersion))
		for _, accountStorage := range account.Storage {
			rawDiff, formatErr := types.FromNewGethStateDiff(account, &stateDiff, accountStorage)
			if formatErr != nil {
				errs <- formatErr
			}

			logrus.Tracef(addingDiffsLogString, rawDiff.HashedAddress.Hex(), rawDiff.BlockHeight, rawDiff.StorageKey.Hex(), rawDiff.StorageValue.Hex())
			out <- rawDiff
		}
	}
}

func (fetcher GethRpcStorageFetcher) decodeStateDiffRlpFromPayloadForOldPatches(payload filters.Payload) (*filters.StateDiff, error) {
	oldPatchStateDiff := new(StateDiffOldPatch)
	decodeErr := rlp.DecodeBytes(payload.StateDiffRlp, oldPatchStateDiff)
	if decodeErr != nil {
		return nil, decodeErr
	}
	stateDiff := convertToNewStateDiff(*oldPatchStateDiff)
	return &stateDiff, nil
}

func convertToNewStateDiff(oldStateDiff StateDiffOldPatch) filters.StateDiff {
	accounts := append(oldStateDiff.UpdatedAccounts, oldStateDiff.CreatedAccounts...)
	accounts = append(accounts, oldStateDiff.DeletedAccounts...)
	convertedAccounts := convertToNewAccountDiff(accounts)
	return filters.StateDiff{
		BlockNumber:     oldStateDiff.BlockNumber,
		BlockHash:       oldStateDiff.BlockHash,
		UpdatedAccounts: convertedAccounts,
	}
}

func convertToNewAccountDiff(oldAccountDiffs []AccountDiffOldPatch) []filters.AccountDiff {
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

func convertToNewStorageDiff(oldStorageDiffs []StorageDiffOldPatch) []filters.StorageDiff {
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

type StateDiffOldPatch struct {
	BlockNumber     *big.Int              `json:"blockNumber"     gencodec:"required"`
	BlockHash       common.Hash           `json:"blockHash"       gencodec:"required"`
	CreatedAccounts []AccountDiffOldPatch `json:"createdAccounts"`
	DeletedAccounts []AccountDiffOldPatch `json:"deletedAccounts"`
	UpdatedAccounts []AccountDiffOldPatch `json:"updatedAccounts" gencodec:"required"`

	encoded []byte
	err     error
}

type AccountDiffOldPatch struct {
	Leaf    bool                  `json:"leaf"`
	Key     []byte                `json:"key"         gencodec:"required"`
	Value   []byte                `json:"value"       gencodec:"required"`
	Proof   [][]byte              `json:"proof"`
	Path    []byte                `json:"path"`
	Storage []StorageDiffOldPatch `json:"storage"     gencodec:"required"`
}

type StorageDiffOldPatch struct {
	Leaf  bool     `json:"leaf"`
	Key   []byte   `json:"key"         gencodec:"required"`
	Value []byte   `json:"value"       gencodec:"required"`
	Proof [][]byte `json:"proof"`
	Path  []byte   `json:"path"`
}
