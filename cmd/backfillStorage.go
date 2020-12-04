package cmd

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/makerdao/vulcanizedb/libraries/shared/factories/storage"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage/backfill"
	"github.com/makerdao/vulcanizedb/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	backfillStorageAddress             string
	backfillStorageAddressFlag         = "backfill-storage-contract-address"
	backfillStorageEndBlockFlag        = "backfill-storage-end-block"
	backfillStorageEndBlockNumber      int64
	backfillStorageStartBlockFlag      = "backfill-storage-start-block"
	backfillStorageStartBlockNumber    int64
	backfillStorageContractAddressFlag = "backfill-storage-contract-address"
)

// backfillStorageCmd represents the backfillStorage command
var backfillStorageCmd = &cobra.Command{
	Use:   "backfillStorage",
	Short: "Backfill smart contract storage for a range of blocks",
	Long: fmt.Sprintf(`Run this command to fetch and persist contract storage for a range of blocks. Useful
if you have started watching diffs from a new contract but do not have storage data
from before you started running the transformer.

   -Requires a config file structured the same as it would be for running compose or 
    execute (to specify which addresses and storage slots must be back-filled).

   -Required CLI flags are %s (-s) and 
    %s (-e) to define range of block that need to be back-filled.

   -Optional CLI flag is %s (-a) to specify a single 
    contract address that needs to be back-filled (if not necessary for all transformers).

Before running this command, verify that you have run headerSync and execute for the 
desired blocks. Headers are required for generating queries for storage slots by hash,
and execute is required since the identifier for storage slots that represent mappings
and dynamic arrays depend on data derived from events.`, backfillStorageStartBlockFlag,
		backfillStorageEndBlockFlag, backfillStorageContractAddressFlag),
	RunE: func(cmd *cobra.Command, args []string) error {
		SubCommand = cmd.CalledAs()
		LogWithCommand = *logrus.WithField("SubCommand", SubCommand)
		return backfillStorage()
	},
}

func init() {
	rootCmd.AddCommand(backfillStorageCmd)
	backfillStorageCmd.Flags().StringVarP(&backfillStorageAddress, backfillStorageAddressFlag, "a", "", "address for which to back-fill storage")
	backfillStorageCmd.Flags().Int64VarP(&backfillStorageStartBlockNumber, backfillStorageStartBlockFlag, "s", -1, "starting block from which to back-fill storage")
	backfillStorageCmd.Flags().Int64VarP(&backfillStorageEndBlockNumber, backfillStorageEndBlockFlag, "e", -1, "ending block for back-filling storage")
}

func backfillStorage() error {
	validationErr := validateBackfillStorageArgs()
	if validationErr != nil {
		return validationErr
	}

	blockChain := getBlockChain()
	db := utils.LoadPostgres(databaseConfig, blockChain.Node())

	genConfig, configErr := prepConfig()
	if configErr != nil {
		LogWithCommand.Fatalf("SubCommand %v: failed to prepare config: %v", SubCommand, configErr)
		return configErr
	}

	_, storageInitializers, _, exportTransformersErr := exportTransformers(genConfig)
	if exportTransformersErr != nil {
		return fmt.Errorf("SubCommand %v: exporting transformers failed: %v", SubCommand, exportTransformersErr)
	}

	if len(storageInitializers) == 0 {
		logrus.Warn("not back-filling storage because no contracts configured for back-fill")
		return nil
	}

	var loader backfill.StorageValueLoader
	if backfillStorageAddress != "" {
		filteredInitializers, filterErr := filterByAddress(backfillStorageAddress, storageInitializers)
		if filterErr != nil {
			return filterErr
		}
		loader = backfill.NewStorageValueLoader(blockChain, &db, filteredInitializers, backfillStorageStartBlockNumber, backfillStorageEndBlockNumber)
	} else {
		loader = backfill.NewStorageValueLoader(blockChain, &db, storageInitializers, backfillStorageStartBlockNumber, backfillStorageEndBlockNumber)
	}

	LogWithCommand.Infof("Back-filling storage for blocks %d-%d", backfillStorageStartBlockNumber, backfillStorageEndBlockNumber)
	return loader.Run()
}

func validateBackfillStorageArgs() error {
	validateStartBlockErr := validateBlockNumberArg(backfillStorageStartBlockNumber, backfillStorageStartBlockFlag)
	if validateStartBlockErr != nil {
		return validateStartBlockErr
	}

	validateEndBlockErr := validateBlockNumberArg(backfillStorageEndBlockNumber, backfillStorageEndBlockFlag)
	if validateEndBlockErr != nil {
		return validateEndBlockErr
	}

	return nil
}

func filterByAddress(address string, initializers []storage.TransformerInitializer) ([]storage.TransformerInitializer, error) {
	for _, initializer := range initializers {
		if initializer(nil).GetContractAddress() == common.HexToAddress(address) {
			return []storage.TransformerInitializer{initializer}, nil
		}
	}
	return nil, fmt.Errorf("subcommand %v: no storage transformer found with address %v", SubCommand, address)
}
