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
	"plugin"

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
	commandRunner := GetStorageValueRunner{}
	executeErr := commandRunner.Execute(blockChain, &db, storageInitializers, blockNumber)

	return executeErr
}

type GetStorageValueRunner struct{}

func (r *GetStorageValueRunner) Execute(bc core.BlockChain, db *postgres.DB, initializers []transformer.StorageTransformerInitializer, blockNumber int64) error {
	addressToKeys := make(map[common.Address][]common.Hash)
	for _, i := range initializers {
		transformer := i(db)
		keysLookup, ok := transformer.GetStorageKeysLookup().(storage.KeysLookup)
		if !ok {
			errorString := fmt.Sprintf("%v type incompatible. Should be a storage.KeysLookup", keysLookup)
			return errors.New(errorString)
		}
		keys, getKeysErr := keysLookup.GetKeys()
		if getKeysErr != nil {
			return getKeysErr
		}
		addressToKeys[transformer.GetContractAddress()] = keys
	}

	blockNumberBigInt := big.NewInt(blockNumber)

	diffRepo := storage2.NewDiffRepository(db)

	headerRepo := repositories.NewHeaderRepository(db)
	header, getHeaderErr := headerRepo.GetHeader(blockNumber)
	if getHeaderErr != nil {
		return getHeaderErr
	}

	for address, keys := range addressToKeys {
		for _, key := range keys {
			value, getStorageErr := bc.GetStorageAt(address, key, blockNumberBigInt)
			if getStorageErr != nil {
				return getStorageErr
			}
			diff := types.RawDiff{
				HashedAddress: crypto.Keccak256Hash(address[:]),
				BlockHash:     common.HexToHash(header.Hash),
				BlockHeight:   int(blockNumber),
				StorageKey:    key,
				StorageValue:  common.BytesToHash(value),
			}

			_, createDiffErr := diffRepo.CreateStorageDiff(diff)
			if createDiffErr != nil {
				fmt.Println(createDiffErr)
				return createDiffErr
			}
		}
	}

	return nil
}

func exportTransformers() []transformer.StorageTransformerInitializer {
	prepConfig()

	// Get the plugin path and load the plugin
	_, pluginPath, pathErr := genConfig.GetPluginPaths()

	if pathErr != nil {
		LogWithCommand.Fatalf("failed to get plugin paths: %s", pathErr.Error())
	}

	LogWithCommand.Info("linking plugin ", pluginPath)
	plug, openErr := plugin.Open(pluginPath)
	if openErr != nil {
		LogWithCommand.Fatalf("linking plugin failed: %s", openErr.Error())
	}

	// Load the `Exporter` symbol from the plugin
	LogWithCommand.Info("loading transformers from plugin")
	symExporter, lookupErr := plug.Lookup("Exporter")
	if lookupErr != nil {
		LogWithCommand.Fatalf("loading Exporter symbol failed: %s", lookupErr.Error())
	}

	// Assert that the symbol is of type Exporter
	exporter, ok := symExporter.(Exporter)
	if !ok {
		LogWithCommand.Fatal("plugged-in symbol not of type Exporter")
	}

	// Use the Exporters export method to load the EventTransformerInitializer, StorageTransformerInitializer, and ContractTransformerInitializer sets
	_, ethStorageInitializers, _ := exporter.Export()

	return ethStorageInitializers
}
