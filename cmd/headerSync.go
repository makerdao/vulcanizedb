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

package cmd

import (
	"fmt"
	"time"

	"github.com/makerdao/vulcanizedb/pkg/core"
	"github.com/makerdao/vulcanizedb/pkg/datastore"
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres/repositories"
	"github.com/makerdao/vulcanizedb/pkg/eth"
	"github.com/makerdao/vulcanizedb/pkg/fs"
	"github.com/makerdao/vulcanizedb/pkg/history"
	"github.com/makerdao/vulcanizedb/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var startingBlockFlagName = "starting-block-number"

// headerSyncCmd represents the headerSync command
var headerSyncCmd = &cobra.Command{
	Use:   "headerSync",
	Short: "Syncs VulcanizeDB with local ethereum node's block headers",
	Long: `Run this command to sync VulcanizeDB with an ethereum node. It populates
Postgres with block headers. You may point to a config file, specify settings via 
CLI flags, or it will attempt to run with default values.`,

	Run: func(cmd *cobra.Command, args []string) {
		SubCommand = cmd.CalledAs()
		LogWithCommand = *logrus.WithField("SubCommand", SubCommand)

		err := headerSync()
		if err != nil {
			LogWithCommand.Fatalf("error executing header sync: %s", err.Error())
		}
	},
}

func init() {
	rootCmd.AddCommand(headerSyncCmd)
	headerSyncCmd.Flags().Int64VarP(&startingBlockNumber, startingBlockFlagName, "s", 0, "Block number to start syncing from")
}

func backFillAllHeaders(blockchain core.BlockChain, headerRepository datastore.HeaderRepository, missingBlocksPopulated chan int, startingBlockNumber int64) {
	populated, err := history.PopulateMissingHeaders(blockchain, headerRepository, startingBlockNumber, validationWindowSize)
	if err != nil {
		LogWithCommand.Errorf("backfillAllHeaders: Error populating headers: %s", err.Error())
	}
	missingBlocksPopulated <- populated
}

func headerSync() error {
	ticker := time.NewTicker(pollingInterval)
	defer ticker.Stop()
	blockChain := getBlockChain()
	validationErr := validateHeaderSyncArgs(blockChain)
	if validationErr != nil {
		return fmt.Errorf("error validating args: %w", validationErr)
	}
	db := utils.LoadPostgres(databaseConfig, blockChain.Node())

	headerRepository := repositories.NewHeaderRepository(&db)
	validator := history.NewHeaderValidator(blockChain, headerRepository, validationWindowSize)
	missingBlocksPopulated := make(chan int)

	statusWriter := fs.NewStatusWriter("/tmp/header_sync_health_check", []byte("headerSync starting\n"))
	writeErr := statusWriter.Write()
	if writeErr != nil {
		return fmt.Errorf("error writing health check file: %w", writeErr)
	}

	go backFillAllHeaders(blockChain, headerRepository, missingBlocksPopulated, startingBlockNumber)

	for {
		select {
		case <-ticker.C:
			window, err := validator.ValidateHeaders()
			if err != nil {
				LogWithCommand.Errorf("headerSync: ValidateHeaders failed: %s", err.Error())
			}
			LogWithCommand.Debug(window.GetString())
		case n := <-missingBlocksPopulated:
			if n == 0 {
				time.Sleep(3 * time.Second)
			}
			go backFillAllHeaders(blockChain, headerRepository, missingBlocksPopulated, startingBlockNumber)
		}
	}
}

func validateHeaderSyncArgs(blockChain *eth.BlockChain) error {
	chainHead, err := blockChain.ChainHead()
	if err != nil {
		return fmt.Errorf("error getting last block from chain: %w", err)
	}
	lastBlockNumber := chainHead.Int64()
	if startingBlockNumber > lastBlockNumber {
		return fmt.Errorf("--%s (%d) greater than client's most recent synced block (%d)",
			startingBlockFlagName, startingBlockNumber, lastBlockNumber)
	}
	return nil
}
