package cmd

import (
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres/repositories"
	"github.com/makerdao/vulcanizedb/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var deleteHeaderBlockNumber int64

// deleteHeaderCmd represents the deleteHeader command
var deleteHeaderCmd = &cobra.Command{
	Use:   "deleteHeader",
	Short: "Delete a header with a given block number from the headers table",
	Long: `Useful if a non-canonical header exists in the DB at a block that's earlier than what's being scanned
in the header validation window (by default, within 15 of the head of the chain).`,
	Run: func(cmd *cobra.Command, args []string) {
		SubCommand = cmd.CalledAs()
		LogWithCommand = *logrus.WithField("SubCommand", SubCommand)
		err := deleteHeader()
		if err != nil {
			LogWithCommand.Fatalf("failed to delete header: %s", err.Error())
		}
		LogWithCommand.Infof("successfully deleted header %d", deleteHeaderBlockNumber)
	},
}

func init() {
	rootCmd.AddCommand(deleteHeaderCmd)
	deleteHeaderCmd.Flags().Int64VarP(&deleteHeaderBlockNumber, blockNumberFlagName, "b", -1, "block number of the header to delete")
	deleteHeaderCmd.MarkFlagRequired(blockNumberFlagName)
}

func deleteHeader() error {
	blockChain := getBlockChain()
	db := utils.LoadPostgres(databaseConfig, blockChain.Node())
	headerRepository := repositories.NewHeaderRepository(&db)
	return headerRepository.DeleteHeader(deleteHeaderBlockNumber)
}
