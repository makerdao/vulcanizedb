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

package repositories_test

import (
	"database/sql"
	"math/big"
	"math/rand"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/makerdao/vulcanizedb/pkg/core"
	"github.com/makerdao/vulcanizedb/pkg/datastore"
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres"
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres/repositories"
	"github.com/makerdao/vulcanizedb/pkg/fakes"
	"github.com/makerdao/vulcanizedb/test_config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Block header repository", func() {
	var (
		db     = test_config.NewTestDB(test_config.NewTestNode())
		repo   datastore.HeaderRepository
		header core.Header
	)

	BeforeEach(func() {
		test_config.CleanTestDB(db)
		repo = repositories.NewHeaderRepository(db)
		header = fakes.GetFakeHeader(rand.Int63n(50000000))
	})

	Describe("creating or updating a header", func() {
		BeforeEach(func() {
			_, createErr := repo.CreateOrUpdateHeader(header)
			Expect(createErr).NotTo(HaveOccurred())
		})

		It("adds a header", func() {
			var dbHeader core.Header
			readErr := db.Get(&dbHeader, `SELECT block_number, hash, raw, block_timestamp FROM public.headers WHERE block_number = $1`, header.BlockNumber)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(dbHeader.BlockNumber).To(Equal(header.BlockNumber))
			Expect(dbHeader.Hash).To(Equal(header.Hash))
			Expect(dbHeader.Raw).To(MatchJSON(header.Raw))
			Expect(dbHeader.Timestamp).To(Equal(header.Timestamp))
		})

		It("adds node data to header", func() {
			var ethNodeId int64
			readErr := db.Get(&ethNodeId, `SELECT eth_node_id FROM public.headers WHERE block_number = $1`, header.BlockNumber)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(ethNodeId).To(Equal(db.NodeID))
		})

		It("does not duplicate headers", func() {
			_, createTwoErr := repo.CreateOrUpdateHeader(header)
			Expect(createTwoErr).NotTo(HaveOccurred())

			var count int
			readErr := db.Get(&count, `SELECT COUNT(*) FROM public.headers WHERE block_number = $1`, header.BlockNumber)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(count).To(Equal(1))
		})

		It("replaces header if hash is different and block number is within 15 of the max block number in the db", func() {
			headerTwo := fakes.GetFakeHeader(header.BlockNumber)

			_, createTwoErr := repo.CreateOrUpdateHeader(headerTwo)

			Expect(createTwoErr).NotTo(HaveOccurred())
			var dbHeaderHash string
			readErr := db.Get(&dbHeaderHash, `SELECT hash FROM public.headers WHERE block_number = $1`, header.BlockNumber)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(dbHeaderHash).To(Equal(headerTwo.Hash))
		})

		It("does not replace header if block number if greater than 15 back from the max block number in the db", func() {
			chainHeadHeader := fakes.GetFakeHeader(header.BlockNumber + 15)
			_, createHeadErr := repo.CreateOrUpdateHeader(chainHeadHeader)
			Expect(createHeadErr).NotTo(HaveOccurred())

			oldConflictingHeader := fakes.GetFakeHeader(header.BlockNumber)
			_, createConflictErr := repo.CreateOrUpdateHeader(oldConflictingHeader)
			Expect(createConflictErr).NotTo(HaveOccurred())

			var dbHeaderHash string
			readErr := db.Get(&dbHeaderHash, `SELECT hash FROM public.headers WHERE block_number = $1`, header.BlockNumber)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(dbHeaderHash).To(Equal(header.Hash))
		})

		It("does not duplicate headers with different hashes", func() {
			headerTwo := fakes.GetFakeHeader(header.BlockNumber)

			_, createTwoErr := repo.CreateOrUpdateHeader(headerTwo)
			Expect(createTwoErr).NotTo(HaveOccurred())

			var dbHeaderHashes []string
			readErr := db.Select(&dbHeaderHashes, `SELECT hash FROM public.headers WHERE block_number = $1`, header.BlockNumber)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(len(dbHeaderHashes)).To(Equal(1))
			Expect(dbHeaderHashes[0]).To(Equal(headerTwo.Hash))
		})

		It("replaces header if hash is different (even from different node)", func() {
			dbTwo := test_config.NewTestDB(test_config.NewTestNode())

			repoTwo := repositories.NewHeaderRepository(dbTwo)
			headerTwo := fakes.GetFakeHeader(header.BlockNumber)

			_, createTwoErr := repoTwo.CreateOrUpdateHeader(headerTwo)

			Expect(createTwoErr).NotTo(HaveOccurred())
			var dbHeaderHashes []string
			readErr := dbTwo.Select(&dbHeaderHashes, `SELECT hash FROM headers WHERE block_number = $1`, header.BlockNumber)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(len(dbHeaderHashes)).To(Equal(1))
			Expect(dbHeaderHashes[0]).To(Equal(headerTwo.Hash))
		})
	})

	Describe("creating a transaction", func() {
		var (
			headerID     int64
			transactions []core.TransactionModel
		)

		BeforeEach(func() {
			var err error
			headerID, err = repo.CreateOrUpdateHeader(header)
			Expect(err).NotTo(HaveOccurred())
			fromAddress := common.HexToAddress("0x1234")
			toAddress := common.HexToAddress("0x5678")
			txHash := common.HexToHash("0x9876")
			txHashTwo := common.HexToHash("0x5432")
			txIndex := big.NewInt(123)
			transactions = []core.TransactionModel{{
				Data:     []byte{},
				From:     fromAddress.Hex(),
				GasLimit: 0,
				GasPrice: 0,
				Hash:     txHash.Hex(),
				Nonce:    0,
				Raw:      []byte{},
				To:       toAddress.Hex(),
				TxIndex:  txIndex.Int64(),
				Value:    "0",
				Receipt:  core.Receipt{},
			}, {
				Data:     []byte{},
				From:     fromAddress.Hex(),
				GasLimit: 1,
				GasPrice: 1,
				Hash:     txHashTwo.Hex(),
				Nonce:    1,
				Raw:      []byte{},
				To:       toAddress.Hex(),
				TxIndex:  1,
				Value:    "1",
			}}

			insertErr := repo.CreateTransactions(headerID, transactions)
			Expect(insertErr).NotTo(HaveOccurred())
		})

		It("adds transactions", func() {
			var dbTransactions []core.TransactionModel
			readErr := db.Select(&dbTransactions,
				`SELECT hash, gas_limit, gas_price, input_data, nonce, raw, tx_from, tx_index, tx_to, "value"
				FROM public.transactions WHERE header_id = $1`, headerID)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(dbTransactions).To(ConsistOf(transactions))
		})

		It("silently ignores duplicate inserts", func() {
			insertTwoErr := repo.CreateTransactions(headerID, transactions)
			Expect(insertTwoErr).NotTo(HaveOccurred())

			var dbTransactions []core.TransactionModel
			readErr := db.Select(&dbTransactions,
				`SELECT hash, gas_limit, gas_price, input_data, nonce, raw, tx_from, tx_index, tx_to, "value"
				FROM public.transactions WHERE header_id = $1`, headerID)
			Expect(readErr).NotTo(HaveOccurred())
			Expect(len(dbTransactions)).To(Equal(2))
		})
	})

	Describe("creating a transaction in a sqlx tx", func() {
		It("adds a transaction", func() {
			headerID, err := repo.CreateOrUpdateHeader(header)
			Expect(err).NotTo(HaveOccurred())
			fromAddress := common.HexToAddress("0x1234")
			toAddress := common.HexToAddress("0x5678")
			txHash := common.HexToHash("0x9876")
			txIndex := big.NewInt(123)
			transaction := core.TransactionModel{
				Data:     []byte{},
				From:     fromAddress.Hex(),
				GasLimit: 0,
				GasPrice: 0,
				Hash:     txHash.Hex(),
				Nonce:    0,
				Raw:      []byte{1, 2, 3},
				To:       toAddress.Hex(),
				TxIndex:  txIndex.Int64(),
				Value:    "0",
			}

			tx, err := db.Beginx()
			Expect(err).ToNot(HaveOccurred())
			_, insertErr := repo.CreateTransactionInTx(tx, headerID, transaction)
			Expect(insertErr).NotTo(HaveOccurred())
			commitErr := tx.Commit()
			Expect(commitErr).ToNot(HaveOccurred())

			var dbTransaction core.TransactionModel
			err = db.Get(&dbTransaction,
				`SELECT hash, gas_limit, gas_price, input_data, nonce, raw, tx_from, tx_index, tx_to, "value"
				FROM public.transactions WHERE header_id = $1`, headerID)
			Expect(err).NotTo(HaveOccurred())
			Expect(dbTransaction).To(Equal(transaction))
		})

		It("silently upserts", func() {
			headerID, err := repo.CreateOrUpdateHeader(header)
			Expect(err).NotTo(HaveOccurred())
			fromAddress := common.HexToAddress("0x1234")
			toAddress := common.HexToAddress("0x5678")
			txHash := common.HexToHash("0x9876")
			txIndex := big.NewInt(123)
			transaction := core.TransactionModel{
				Data:     []byte{},
				From:     fromAddress.Hex(),
				GasLimit: 0,
				GasPrice: 0,
				Hash:     txHash.Hex(),
				Nonce:    0,
				Raw:      []byte{},
				Receipt:  core.Receipt{},
				To:       toAddress.Hex(),
				TxIndex:  txIndex.Int64(),
				Value:    "0",
			}

			tx1Err := repo.CreateTransactions(headerID, []core.TransactionModel{transaction})
			Expect(tx1Err).NotTo(HaveOccurred())

			tx2Err := repo.CreateTransactions(headerID, []core.TransactionModel{transaction})
			Expect(tx2Err).NotTo(HaveOccurred())

			var dbTransactions []core.TransactionModel
			err = db.Select(&dbTransactions,
				`SELECT hash, gas_limit, gas_price, input_data, nonce, raw, tx_from, tx_index, tx_to, "value"
				FROM public.transactions WHERE header_id = $1`, headerID)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(dbTransactions)).To(Equal(1))
		})
	})

	Describe("deleting a header", func() {
		It("returns error if header does not exist", func() {
			err := repo.DeleteHeader(rand.Int63())

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(postgres.ErrHeaderDoesNotExist))
		})

		It("deletes header if it exists", func() {
			blockNumber := rand.Int63()
			header := fakes.GetFakeHeader(blockNumber)
			_, insertErr := repo.CreateOrUpdateHeader(header)
			Expect(insertErr).NotTo(HaveOccurred())

			err := repo.DeleteHeader(blockNumber)

			Expect(err).NotTo(HaveOccurred())
			persistedHeader, getErr := repo.GetHeaderByBlockNumber(blockNumber)
			Expect(persistedHeader).To(BeZero())
			Expect(getErr).To(HaveOccurred())
			Expect(getErr).To(MatchError(postgres.ErrHeaderDoesNotExist))
		})
	})

	Describe("Getting a header by block number", func() {
		It("returns header if it exists", func() {
			_, createErr := repo.CreateOrUpdateHeader(header)
			Expect(createErr).NotTo(HaveOccurred())

			dbHeader, err := repo.GetHeaderByBlockNumber(header.BlockNumber)

			Expect(err).NotTo(HaveOccurred())
			Expect(dbHeader.Id).NotTo(BeZero())
			Expect(dbHeader.BlockNumber).To(Equal(header.BlockNumber))
			Expect(dbHeader.Hash).To(Equal(header.Hash))
			Expect(dbHeader.Raw).To(MatchJSON(header.Raw))
			Expect(dbHeader.Timestamp).To(Equal(header.Timestamp))
		})

		It("returns header from any node", func() {
			_, createErr := repo.CreateOrUpdateHeader(header)
			Expect(createErr).NotTo(HaveOccurred())

			dbTwo := test_config.NewTestDB(test_config.NewTestNode())
			repoTwo := repositories.NewHeaderRepository(dbTwo)

			result, readErr := repoTwo.GetHeaderByBlockNumber(header.BlockNumber)

			Expect(readErr).NotTo(HaveOccurred())
			Expect(result.Raw).To(MatchJSON(header.Raw))
		})
	})

	Describe("Getting a header by ID", func() {
		It("returns header with associated ID", func() {
			wantedHeader := core.Header{
				BlockNumber: rand.Int63(),
				Hash:        fakes.RandomString(64),
				Raw:         nil,
				Timestamp:   strconv.Itoa(rand.Int()),
			}
			var wantedHeaderID int64
			wantedHeaderErr := db.Get(&wantedHeaderID, `
				INSERT INTO public.headers (block_number, hash, block_timestamp, eth_node_id) VALUES ($1, $2, $3, $4)
				RETURNING id`, wantedHeader.BlockNumber, wantedHeader.Hash, wantedHeader.Timestamp, db.NodeID)
			Expect(wantedHeaderErr).NotTo(HaveOccurred())
			wantedHeader.Id = wantedHeaderID

			_, anotherHeaderErr := db.Exec(`INSERT INTO public.headers (block_number, hash, block_timestamp,
                            eth_node_id) VALUES ($1, $2, $3, $4) RETURNING id`, rand.Int()-1, fakes.RandomString(64),
				strconv.Itoa(rand.Int()), db.NodeID)
			Expect(anotherHeaderErr).NotTo(HaveOccurred())

			header, err := repo.GetHeaderByID(wantedHeaderID)

			Expect(err).NotTo(HaveOccurred())
			Expect(header).To(Equal(wantedHeader))
		})
	})

	Describe("Getting headers in range", func() {
		var blockTwo int64

		BeforeEach(func() {
			_, headerErrOne := repo.CreateOrUpdateHeader(header)
			Expect(headerErrOne).NotTo(HaveOccurred())
			blockTwo = header.BlockNumber + 1
			headerTwo := core.Header{
				BlockNumber: blockTwo,
				Hash:        common.BytesToHash([]byte{5, 4, 3, 2, 1}).Hex(),
				Raw:         header.Raw,
				Timestamp:   header.Timestamp,
			}
			_, headerErrTwo := repo.CreateOrUpdateHeader(headerTwo)
			Expect(headerErrTwo).NotTo(HaveOccurred())
		})

		It("returns all headers in range in ascending order", func() {
			dbHeaders, err := repo.GetHeadersInRange(header.BlockNumber, blockTwo)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(dbHeaders)).To(Equal(2))
			Expect(dbHeaders[0].BlockNumber).To(Equal(header.BlockNumber))
		})

		It("does not return header outside of block range", func() {
			dbHeaders, err := repo.GetHeadersInRange(header.BlockNumber, header.BlockNumber)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(dbHeaders)).To(Equal(1))
		})
	})

	Describe("Getting missing headers", func() {
		It("returns block numbers for headers not in the db", func() {
			_, createOneErr := repo.CreateOrUpdateHeader(fakes.GetFakeHeader(1))
			Expect(createOneErr).NotTo(HaveOccurred())

			_, createTwoErr := repo.CreateOrUpdateHeader(fakes.GetFakeHeader(3))
			Expect(createTwoErr).NotTo(HaveOccurred())

			_, createThreeErr := repo.CreateOrUpdateHeader(fakes.GetFakeHeader(5))
			Expect(createThreeErr).NotTo(HaveOccurred())

			missingBlockNumbers, err := repo.MissingBlockNumbers(1, 5)
			Expect(err).NotTo(HaveOccurred())

			Expect(missingBlockNumbers).To(ConsistOf([]int64{2, 4}))
		})

		It("treats headers created by _any_ node as not missing", func() {
			_, createOneErr := repo.CreateOrUpdateHeader(fakes.GetFakeHeader(1))
			Expect(createOneErr).NotTo(HaveOccurred())

			_, createTwoErr := repo.CreateOrUpdateHeader(fakes.GetFakeHeader(3))
			Expect(createTwoErr).NotTo(HaveOccurred())

			_, createThreeErr := repo.CreateOrUpdateHeader(fakes.GetFakeHeader(5))
			Expect(createThreeErr).NotTo(HaveOccurred())

			dbTwo := test_config.NewTestDB(test_config.NewTestNode())
			repoTwo := repositories.NewHeaderRepository(dbTwo)

			missingBlockNumbers, err := repoTwo.MissingBlockNumbers(1, 5)
			Expect(err).NotTo(HaveOccurred())

			Expect(missingBlockNumbers).To(ConsistOf([]int64{2, 4}))
		})
	})

	Describe("GetMostRecentHeaderBlockNumber", func() {
		It("gets the most recent header block number", func() {
			_, createHeader1Err := repo.CreateOrUpdateHeader(header)
			Expect(createHeader1Err).NotTo(HaveOccurred())

			header2BlockNumber := header.BlockNumber + int64(1)
			header2 := fakes.GetFakeHeader(header2BlockNumber)
			_, createHeader2Err := repo.CreateOrUpdateHeader(header2)
			Expect(createHeader2Err).NotTo(HaveOccurred())

			mostRecentHeaderBlock, err := repo.GetMostRecentHeaderBlockNumber()
			Expect(err).NotTo(HaveOccurred())
			Expect(mostRecentHeaderBlock).To(Equal(header2BlockNumber))
		})

		It("returns an error if it fails to get the most recent header", func() {
			_, err := repo.GetMostRecentHeaderBlockNumber()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(sql.ErrNoRows))
		})
	})
})
