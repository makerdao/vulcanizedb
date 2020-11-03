package cmd

import (
	"fmt"

	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/makerdao/vulcanizedb/cmd/helpers"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage/fetcher"
	"github.com/makerdao/vulcanizedb/libraries/shared/streamer"
	"github.com/makerdao/vulcanizedb/pkg/fs"
	"github.com/makerdao/vulcanizedb/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	storageDiffsPathFlag   = "fileSystem-storageDiffsPath"
	storageDiffsPath       string
	storageDiffsSourceFlag = "storageDiffs-source"
	storageDiffsSource     string
)

// extractDiffsCmd represents the extractDiffs command
var extractDiffsCmd = &cobra.Command{
	Use:   "extractDiffs",
	Short: "Extract storage diffs from a node and write them to postgres",
	Long: fmt.Sprintf(`Run this command to reads storage diffs from either a CSV or JSON RPC subscription.
Configure which with the %s flag. Received diffs are written to public.storage_diff.`, storageDiffsSourceFlag),
	Run: func(cmd *cobra.Command, args []string) {
		SubCommand = cmd.CalledAs()
		LogWithCommand = *logrus.WithField("SubCommand", SubCommand)
		extractDiffs()
	},
}

func init() {
	rootCmd.AddCommand(extractDiffsCmd)
	extractDiffsCmd.Flags().StringVarP(&storageDiffsSource, storageDiffsSourceFlag, "s", "csv", "where to get the state diffs: csv or geth")
	extractDiffsCmd.Flags().StringVarP(&storageDiffsPath, storageDiffsPathFlag, "p", "", "location of storage diffs csv file")
}

func extractDiffs() {
	// Setup bc and db objects
	blockChain := getBlockChain()
	db := utils.LoadPostgres(databaseConfig, blockChain.Node())

	healthCheckFile := "/tmp/connection"
	msg := []byte("geth storage fetcher connection established\n")
	gethStatusWriter := fs.NewStatusWriter(healthCheckFile, msg)

	// initialize fetcher
	var storageFetcher fetcher.IStorageFetcher
	logrus.Debug("fetching storage diffs from geth")
	switch storageDiffsSource {
	case "geth":
		logrus.Info("Using new geth patch with filters event system")
		_, ethClient := getClients()
		filterQuery, filterQueryErr := helpers.CreateFilterQuery()
		if filterQueryErr != nil {
			LogWithCommand.Fatalf("Error creating filter query from config file: %s", filterQueryErr)
		}
		stateDiffStreamer := streamer.NewEthStateChangeStreamer(ethClient, filterQuery)
		payloadChan := make(chan filters.Payload)
		storageFetcher = fetcher.NewGethRpcStorageFetcher(&stateDiffStreamer, payloadChan, gethStatusWriter)
	default:
		logrus.Debug("fetching storage diffs from csv")
		tailer := fs.FileTailer{Path: storageDiffsPath}
		msg := []byte("csv tail storage fetcher connection established\n")
		statusWriter := fs.NewStatusWriter(healthCheckFile, msg)

		storageFetcher = fetcher.NewCsvTailStorageFetcher(tailer, statusWriter)
	}

	// extract diffs
	extractor := storage.NewDiffExtractor(storageFetcher, &db)
	err := extractor.ExtractDiffs()
	if err != nil {
		LogWithCommand.Fatalf("extracting diffs failed: %s", err.Error())
	}
}
