package cmd

import (
	"github.com/makerdao/vulcanizedb/libraries/shared/storage"
	"github.com/makerdao/vulcanizedb/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var resetDiffsBlockNumber int64

// resetNonCanonicalDiffsToNewCmd represents the resetNonCanonicalDiffsToNew command
var resetNonCanonicalDiffsToNewCmd = &cobra.Command{
	Use:   "resetNonCanonicalDiffsToNew",
	Short: "Reset diffs with status of 'non-canonical' at given block to 'new'.",
	Long: `If the database has an invalid header at an older block (outside of the validation window),
then diffs from the valid header will be marked as 'non-canonical'. In that case,
we need to remove the invalid header and update the associated diffs' status so
that they can be re-checked and transformed if valid. This command handles updating
the diffs' status.`,
	Run: func(cmd *cobra.Command, args []string) {
		SubCommand = cmd.CalledAs()
		LogWithCommand = *logrus.WithField("SubCommand", SubCommand)
		err := resetNonCanonicalDiffsToNew()
		if err != nil {
			LogWithCommand.Fatalf("failed to reset non-canonical diffs to new: %s", err.Error())
		}
		LogWithCommand.Infof("successfully reset non-canonical diffs to new at block %d", resetDiffsBlockNumber)
	},
}

func init() {
	rootCmd.AddCommand(resetNonCanonicalDiffsToNewCmd)
	resetNonCanonicalDiffsToNewCmd.Flags().Int64VarP(&resetDiffsBlockNumber, blockNumberFlagName, "b", -1, "block number where non-canonical diffs should be marked new")
	resetNonCanonicalDiffsToNewCmd.MarkFlagRequired(blockNumberFlagName)
}

func resetNonCanonicalDiffsToNew() error {
	blockChain := getBlockChain()
	db := utils.LoadPostgres(databaseConfig, blockChain.Node())
	diffRepository := storage.NewDiffRepository(&db)
	return diffRepository.MarkNoncanonicalDiffsAsNew(resetDiffsBlockNumber)
}
