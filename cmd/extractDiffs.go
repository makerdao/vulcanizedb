package cmd

import (
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage/fetcher"
	"github.com/makerdao/vulcanizedb/libraries/shared/streamer"
	"github.com/makerdao/vulcanizedb/pkg/fs"
	"github.com/makerdao/vulcanizedb/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	watchedStorageAddresses         []string
	watchedStorageAddressesFlagName = "watchedStorageAddresses"
)

// extractDiffsCmd represents the extractDiffs command
var extractDiffsCmd = &cobra.Command{
	Use:   "extractDiffs",
	Short: "Extract storage diffs from a node and write them to postgres",
	Long: `Reads storage diffs from either a CSV or JSON RPC subscription.
	Configure which with the STORAGEDIFFS_SOURCE flag. Received diffs are
	written to public.storage_diff.`,
	Run: func(cmd *cobra.Command, args []string) {
		SubCommand = cmd.CalledAs()
		LogWithCommand = *logrus.WithField("SubCommand", SubCommand)
		extractDiffs()
	},
}

func init() {
	extractDiffsCmd.Flags().StringSliceVarP(&watchedStorageAddresses, watchedStorageAddressesFlagName, "w", []string{}, "contract addresses to subscribe to for storage diffs")
	rootCmd.AddCommand(extractDiffsCmd)
}

func extractDiffs() {
	// Setup bc and db objects
	blockChain := getBlockChain()
	db := utils.LoadPostgres(databaseConfig, blockChain.Node())

	// initialize fetcher
	var storageFetcher fetcher.IStorageFetcher
	logrus.Debug("fetching storage diffs from geth")
	switch storageDiffsSource {
	case "geth":
		logrus.Info("Using original geth patch")
		rpcClient, _ := getClients()
		stateDiffStreamer := streamer.NewStateDiffStreamer(rpcClient)
		payloadChan := make(chan filters.Payload)
		storageFetcher = fetcher.NewGethRpcStorageFetcher(&stateDiffStreamer, payloadChan, fetcher.OldGethPatch)
	case "new-geth":
		logrus.Info("Using new geth patch with statediff service")
		rpcClient, _ := getClients()
		stateDiffStreamer := streamer.NewStateDiffStreamer(rpcClient)
		payloadChan := make(chan filters.Payload)
		storageFetcher = fetcher.NewGethRpcStorageFetcher(&stateDiffStreamer, payloadChan, fetcher.NewGethPatchWithService)
	case "new-geth-with-filter":
		logrus.Info("Using new geth patch with filters event system")
		_, ethClient := getClients()
		filterQuery := createFilterQuery()
		stateDiffStreamer := streamer.NewEthStateChangeStreamer(ethClient, filterQuery)
		payloadChan := make(chan filters.Payload)
		storageFetcher = fetcher.NewGethRpcStorageFetcher(&stateDiffStreamer, payloadChan, fetcher.NewGethPatchWithFilter)
	default:
		logrus.Debug("fetching storage diffs from csv")
		tailer := fs.FileTailer{Path: storageDiffsPath}
		storageFetcher = fetcher.NewCsvTailStorageFetcher(tailer)
	}

	// extract diffs
	extractor := storage.NewDiffExtractor(storageFetcher, &db)
	err := extractor.ExtractDiffs()
	if err != nil {
		LogWithCommand.Fatalf("extracting diffs failed: %s", err.Error())
	}
}

func createFilterQuery() ethereum.FilterQuery {
	var addresses []common.Address
	for _, addressString := range watchedStorageAddresses {
		addresses = append(addresses, common.HexToAddress(addressString))
	}

	return ethereum.FilterQuery{
		Addresses: addresses,
	}
}
