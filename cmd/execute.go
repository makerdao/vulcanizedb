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
	"sync"
	"time"

	"github.com/makerdao/vulcanizedb/libraries/shared/constants"
	"github.com/makerdao/vulcanizedb/libraries/shared/factories/event"
	"github.com/makerdao/vulcanizedb/libraries/shared/factories/storage"
	"github.com/makerdao/vulcanizedb/libraries/shared/logs"
	"github.com/makerdao/vulcanizedb/libraries/shared/transformer"
	"github.com/makerdao/vulcanizedb/libraries/shared/watcher"
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres/repositories"
	"github.com/makerdao/vulcanizedb/pkg/fs"
	"github.com/makerdao/vulcanizedb/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// executeCmd represents the execute command
var executeCmd = &cobra.Command{
	Use:   "execute",
	Short: "Executes a precomposed transformer initializer plugin",
	Long: `Run this command to take the composed plugin and pass it to the appropriate watcher 
to execute over. The plugin file needs to be located in the /plugins directory 
and this command assumes the db migrations remain from when the plugin was composed.
Additionally, the plugin must have been composed by the same version of vulcanizedb 
or else it will not be compatible.

This command needs a config file location specified: 
./vulcanizedb execute --config=./environments/config_name.toml`,
	Run: func(cmd *cobra.Command, args []string) {
		SubCommand = cmd.CalledAs()
		LogWithCommand = *logrus.WithField("SubCommand", SubCommand)
		execute()
	},
}

func execute() {
	executeTransformers()
}

func init() {
	rootCmd.AddCommand(executeCmd)
	executeCmd.Flags().BoolVarP(&recheckHeadersArg, "recheck-headers", "r", false, "whether to re-check headers for watched events")
	executeCmd.Flags().DurationVarP(&retryInterval, "retry-interval", "i", 7*time.Second, "interval duration between retries on execution error")
	executeCmd.Flags().IntVarP(&maxUnexpectedErrors, "max-unexpected-errs", "m", 5, "maximum number of unexpected errors to allow (with retries) before exiting")
	executeCmd.Flags().Int64VarP(&newDiffBlockFromHeadOfChain, "new-diff-blocks-from-head", "d", -1, "number of blocks from head of chain to start reprocessing new diffs, defaults to -1 so all diffs are processsed")
	executeCmd.Flags().Int64VarP(&unrecognizedDiffBlockFromHeadOfChain, "unrecognized-diff-blocks-from-head", "u", -1, "number of blocks from head of chain to start reprocessing unrecognized diffs, defaults to -1 so all diffs are processsed")
}

func executeTransformers() {
	genConfig, configErr := prepConfig()
	if configErr != nil {
		LogWithCommand.Fatalf("SubCommand %v: failed to prepare config: %v", SubCommand, configErr)
	}

	ethEventInitializers, ethStorageInitializers, _, exportTransformersErr := exportTransformers(genConfig)
	if exportTransformersErr != nil {
		LogWithCommand.Fatalf("SubCommand %v: exporting transformers failed: %v", SubCommand, exportTransformersErr)
	}

	// Setup bc and db objects
	blockChain := getBlockChain()
	db := utils.LoadPostgres(databaseConfig, blockChain.Node())
	healthCheckFile := "/tmp/execute_health_check"

	// Execute over transformer sets returned by the exporter
	// Use WaitGroup to wait on both goroutines
	var wg sync.WaitGroup
	if len(ethEventInitializers) > 0 {
		repo, repoErr := repositories.NewCheckedHeadersRepository(&db, genConfig.Schema)
		if repoErr != nil {
			LogWithCommand.Fatalf("failed to create checked headers repository %s for schema %s", repoErr.Error(), genConfig.Schema)
		}
		extractor := logs.NewLogExtractor(&db, blockChain, repo)
		delegator := logs.NewLogDelegator(&db)
		eventHealthCheckMessage := []byte("event watcher starting\n")
		statusWriter := fs.NewStatusWriter(healthCheckFile, eventHealthCheckMessage)
		ew := watcher.NewEventWatcher(&db, blockChain, extractor, delegator, maxUnexpectedErrors, retryInterval, statusWriter)
		addErr := ew.AddTransformers(ethEventInitializers)
		if addErr != nil {
			LogWithCommand.Fatalf("failed to add event transformer initializers to watcher: %s", addErr.Error())
		}
		wg.Add(1)
		go watchEthEvents(&ew, &wg)
	}

	if len(ethStorageInitializers) > 0 {
		newDiffStorageHealthCheckMessage := []byte("storage watcher for new diffs starting\n")
		newDiffStatusWriter := fs.NewStatusWriter(healthCheckFile, newDiffStorageHealthCheckMessage)
		newDiffStorageWatcher := watcher.NewStorageWatcher(&db, newDiffBlockFromHeadOfChain, newDiffStatusWriter)
		newDiffStorageWatcher.AddTransformers(ethStorageInitializers)
		wg.Add(1)
		go watchEthStorage(&newDiffStorageWatcher, &wg)

		unrecognizedDiffStorageHealthCheckMessage := []byte("storage watcher for unrecognized diffs starting\n")
		unrecognizedDiffStatusWriter := fs.NewStatusWriter(healthCheckFile, unrecognizedDiffStorageHealthCheckMessage)
		unrecognizedDiffStorageWatcher := watcher.UnrecognizedStorageWatcher(&db, unrecognizedDiffBlockFromHeadOfChain, unrecognizedDiffStatusWriter)
		unrecognizedDiffStorageWatcher.AddTransformers(ethStorageInitializers)
		wg.Add(1)
		go watchEthStorage(&unrecognizedDiffStorageWatcher, &wg)

		pendingDiffStorageHealthCheckMessage := []byte("storage watcher for pending diffs starting\n")
		pendingDiffStatusWriter := fs.NewStatusWriter(healthCheckFile, pendingDiffStorageHealthCheckMessage)
		pendingDiffStorageWatcher := watcher.PendingStorageWatcher(&db, newDiffBlockFromHeadOfChain, pendingDiffStatusWriter)
		pendingDiffStorageWatcher.AddTransformers(ethStorageInitializers)
		wg.Add(1)
		go watchEthStorage(&pendingDiffStorageWatcher, &wg)
	}
	wg.Wait()
}

type Exporter interface {
	Export() ([]event.TransformerInitializer, []storage.TransformerInitializer, []transformer.ContractTransformerInitializer)
}

func watchEthEvents(w *watcher.EventWatcher, wg *sync.WaitGroup) {
	defer wg.Done()
	// Execute over the EventTransformerInitializer set using the watcher
	LogWithCommand.Info("executing event transformers")
	var recheck constants.TransformerExecution
	if recheckHeadersArg {
		recheck = constants.HeaderRecheck
	} else {
		recheck = constants.HeaderUnchecked
	}
	err := w.Execute(recheck)
	if err != nil {
		LogWithCommand.Fatalf("error executing event watcher: %s", err.Error())
	}
}

func watchEthStorage(w watcher.IStorageWatcher, wg *sync.WaitGroup) {
	defer wg.Done()
	// Execute over the storage.TransformerInitializer set using the storage watcher
	LogWithCommand.Infof("executing %s storage transformers", w.StorageWatcherName())
	err := w.Execute()
	if err != nil {
		LogWithCommand.Fatalf("error executing storage watcher: %s", err.Error())
	}
}
