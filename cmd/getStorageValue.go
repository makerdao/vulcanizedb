// Copyright © 2020 NAME HERE <EMAIL ADDRESS>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/makerdao/vulcanizedb/libraries/shared/factories/storage"
	storage2 "github.com/makerdao/vulcanizedb/libraries/shared/storage"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage/types"
	"github.com/makerdao/vulcanizedb/libraries/shared/transformer"
	"github.com/makerdao/vulcanizedb/pkg/core"
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres"
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres/repositories"
	"github.com/makerdao/vulcanizedb/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var getStorageValueBlockNumber int64

// getStorageValueCmd represents the getStorageValue command
var getStorageValueCmd = &cobra.Command{
	Use:   "getStorageValue",
	Short: "Gets all storage values for configured contracts at the given block.",
	Long: `Fetches and persists storage values of the configured contracts at a given block. It is important to note that the storage value gotten with this
	command may not be different from the previous block in the database.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		SubCommand = cmd.CalledAs()
		LogWithCommand = *logrus.WithField("SubCommand", SubCommand)
		fmt.Println("getStorageValue called")
		return getStorageAt(getStorageValueBlockNumber)
	},
}

func init() {
	getStorageValueCmd.Flags().Int64VarP(&getStorageValueBlockNumber, "get-storage-value-block-number", "b", -1, "block number to fetch storage at for all configured transformers")
	rootCmd.AddCommand(getStorageValueCmd)
}

func getStorageAt(blockNumber int64) error {
	blockChain := getBlockChain()
	db := utils.LoadPostgres(databaseConfig, blockChain.Node())
	storageInitializers := exportTransformers()
	commandRunner := NewStorageValueCommandRunner(blockChain, &db, storageInitializers)
	return commandRunner.Run(blockNumber)
}

func NewStorageValueCommandRunner(bc core.BlockChain, db *postgres.DB, initializers []transformer.StorageTransformerInitializer) StorageValueCommandRunner{
	return StorageValueCommandRunner{
		bc:      bc,
		db: db,
		headerRepo:      repositories.NewHeaderRepository(db),
		storageDiffRepo: storage2.NewDiffRepository(db),
		initializers:    initializers,
	}
}

type StorageValueCommandRunner struct{
	bc core.BlockChain
	db *postgres.DB
	headerRepo repositories.HeaderRepository
	storageDiffRepo storage2.DiffRepository
	initializers []transformer.StorageTransformerInitializer
}

func (r *StorageValueCommandRunner) Run(blockNumber int64) error {
	addressToKeys, getKeysErr := r.getStorageKeys()
	if getKeysErr != nil {
		return getKeysErr
	}

	header, getHeaderErr := r.headerRepo.GetHeader(blockNumber)
	if getHeaderErr != nil {
		return getHeaderErr
	}

	for address, keys := range addressToKeys {
		persistStorageErr := r.persistStorageValues(address, keys, blockNumber, header.Hash)
		if persistStorageErr != nil {
			return persistStorageErr
		}
	}

	return nil
}

func (r *StorageValueCommandRunner) persistStorageValues(address common.Address, keys []common.Hash, blockNumber int64, headerHash string) error {
	blockNumberBigInt := big.NewInt(blockNumber)
	for _, key := range keys {
		value, getStorageErr := r.bc.GetStorageAt(address, key, blockNumberBigInt)
		if getStorageErr != nil {
			return getStorageErr
		}
		diff := types.RawDiff{
			HashedAddress: crypto.Keccak256Hash(address[:]),
			BlockHash:     common.HexToHash(headerHash),
			BlockHeight:   int(blockNumber),
			StorageKey:    key,
			StorageValue:  common.BytesToHash(value),
		}

		_, createDiffErr := r.storageDiffRepo.CreateStorageDiff(diff)
		if createDiffErr != nil {
			fmt.Println(createDiffErr)
			return createDiffErr
		}
	}
	return nil
}

func (r *StorageValueCommandRunner) getStorageKeys() (map[common.Address][]common.Hash, error) {
	addressToKeys := make(map[common.Address][]common.Hash)
	for _, i := range r.initializers {
		transformer := i(r.db)
		keysLookup, ok := transformer.GetStorageKeysLookup().(storage.KeysLookup)
		if !ok {
			errorString := fmt.Sprintf("%v type incompatible. Should be a storage.KeysLookup", keysLookup)
			return addressToKeys, errors.New(errorString)
		}
		keys, getKeysErr := keysLookup.GetKeys()
		if getKeysErr != nil {
			return addressToKeys, getKeysErr
		}
		addressToKeys[transformer.GetContractAddress()] = keys
	}

	return addressToKeys, nil
}

