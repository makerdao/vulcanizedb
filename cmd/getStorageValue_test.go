package cmd_test

import (
	"database/sql"
	"math/big"
	"math/rand"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/makerdao/vulcanizedb/cmd"
	"github.com/makerdao/vulcanizedb/libraries/shared/factories/storage"
	"github.com/makerdao/vulcanizedb/libraries/shared/mocks"
	"github.com/makerdao/vulcanizedb/libraries/shared/transformer"
	"github.com/makerdao/vulcanizedb/pkg/core"
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres/repositories"
	"github.com/makerdao/vulcanizedb/pkg/fakes"
	"github.com/makerdao/vulcanizedb/test_config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("getStorageValue Command", func() {
	var (
		bc                             fakes.MockBlockChain
		db                             = test_config.NewTestDB(test_config.NewTestNode())
		keysLookupOne, keysLookupTwo   mocks.MockStorageKeysLookup
		repoOne, repoTwo               mocks.MockStorageRepository
		runner                         cmd.StorageValueCommandRunner
		initializerOne, initializerTwo transformer.StorageTransformerInitializer
		initializers                   []transformer.StorageTransformerInitializer
		keyOne, keyTwo                 common.Hash
		addressOne, addressTwo         common.Address
		blockNumber                    int64
		bigIntBlockNumber              *big.Int
		fakeHeader                     core.Header
	)

	BeforeEach(func() {
		bc = fakes.MockBlockChain{}

		keysLookupOne = mocks.MockStorageKeysLookup{}
		repoOne = mocks.MockStorageRepository{}
		keyOne = common.Hash{1, 2, 3}
		addressOne = fakes.FakeAddress

		keysLookupTwo = mocks.MockStorageKeysLookup{}
		repoTwo = mocks.MockStorageRepository{}
		keyTwo = common.Hash{4, 5, 6}
		addressTwo = fakes.AnotherFakeAddress

		initializerOne = storage.Transformer{
			Address:           addressOne,
			StorageKeysLookup: &keysLookupOne,
			Repository:        &repoOne,
		}.NewTransformer

		initializerTwo = storage.Transformer{
			Address:           addressTwo,
			StorageKeysLookup: &keysLookupTwo,
			Repository:        &repoTwo,
		}.NewTransformer

		initializers = []transformer.StorageTransformerInitializer{initializerOne, initializerTwo}
		blockNumber = rand.Int63()
		bigIntBlockNumber = big.NewInt(blockNumber)
		headerRepository := repositories.NewHeaderRepository(db)
		fakeHeader = fakes.FakeHeader
		fakeHeader.BlockNumber = blockNumber
		_, insertHeaderErr := headerRepository.CreateOrUpdateHeader(fakeHeader)
		Expect(insertHeaderErr).NotTo(HaveOccurred())

		runner = cmd.NewStorageValueCommandRunner(&bc, db, initializers, blockNumber)
	})

	AfterEach(func() {
		test_config.CleanTestDB(db)
	})

	It("gets the storage keys for each transformer", func() {
		runnerErr := runner.Run()
		Expect(runnerErr).NotTo(HaveOccurred())

		Expect(keysLookupOne.GetKeysCalled).To(BeTrue())
		Expect(keysLookupTwo.GetKeysCalled).To(BeTrue())
	})

	It("gets the storage values for each of the transformer's keys", func() {
		keysLookupOne.SetKeysToReturn([]common.Hash{keyOne})
		keysLookupTwo.SetKeysToReturn([]common.Hash{keyTwo})

		runnerErr := runner.Run()
		Expect(runnerErr).NotTo(HaveOccurred())
		Expect(keysLookupOne.GetKeysCalled).To(BeTrue())
		Expect(keysLookupTwo.GetKeysCalled).To(BeTrue())

		Expect(bc.GetStorageAtPassedBlockNumber).To(Equal(bigIntBlockNumber))
		Expect(bc.GetStorageAtPassedAccounts).To(ConsistOf(addressOne, addressTwo))
		Expect(bc.GetStorageAtPassedKeys).To(ConsistOf(keyOne, keyTwo))
	})

	It("returns an error if getting the keys from the KeysLookup fails", func() {
		keysLookupTwo.SetGetKeysError(fakes.FakeError)

		runnerErr := runner.Run()
		Expect(keysLookupOne.GetKeysCalled).To(BeTrue())
		Expect(runnerErr).To(HaveOccurred())
		Expect(runnerErr).To(Equal(fakes.FakeError))
	})

	It("returns an error if blockchain call to GetStorageAt fails", func() {
		keysLookupOne.SetKeysToReturn([]common.Hash{keyOne})
		bc.SetGetStorageAtError(fakes.FakeError)

		runnerErr := runner.Run()
		Expect(keysLookupOne.GetKeysCalled).To(BeTrue())
		Expect(runnerErr).To(HaveOccurred())
		Expect(runnerErr).To(Equal(fakes.FakeError))
	})

	It("persists the storage values for each transformer", func() {
		keysLookupOne.SetKeysToReturn([]common.Hash{keyOne})
		keysLookupTwo.SetKeysToReturn([]common.Hash{keyTwo})
		value1 := common.BytesToHash([]byte{7, 8, 9})
		value2 := common.BytesToHash([]byte{10, 11, 12})
		bc.SetStorageValuesToReturn([][]byte{value1[:], value2[:]})

		runnerErr := runner.Run()
		Expect(runnerErr).NotTo(HaveOccurred())

		var dbResults []dbDiffResult
		getDbResultsErr := db.Select(&dbResults, `SELECT block_height, block_hash, hashed_address, storage_key, storage_value FROM public.storage_diff`)
		Expect(getDbResultsErr).NotTo(HaveOccurred())

		trimmedHeaderHash := strings.TrimPrefix(fakeHeader.Hash, "0x")
		headerHashBytes := common.Hex2Bytes(trimmedHeaderHash)
		expectedResultOne := dbDiffResult{
			BlockHeight:   int(blockNumber),
			BlockHash:     headerHashBytes,
			HashedAddress: crypto.Keccak256Hash(addressOne[:]).Bytes(),
			StorageKey:    keyOne[:],
			StorageValue:  value1[:],
		}
		expectedResultTwo := dbDiffResult{
			BlockHeight:   int(blockNumber),
			BlockHash:     headerHashBytes,
			HashedAddress: crypto.Keccak256Hash(addressTwo[:]).Bytes(),
			StorageKey:    keyTwo[:],
			StorageValue:  value2[:],
		}

		Expect(dbResults).To(ConsistOf(expectedResultOne, expectedResultTwo))
	})

	It("ignore duplicate diffs", func() {
		keysLookupOne.SetKeysToReturn([]common.Hash{keyOne})
		value1 := common.BytesToHash([]byte{7, 8, 9})
		//Simulating requesting the same key from the blockChain twice
		bc.SetStorageValuesToReturn([][]byte{value1[:], value1[:]})

		initializers := []transformer.StorageTransformerInitializer{initializerOne}
		runner = cmd.NewStorageValueCommandRunner(&bc, db, initializers, blockNumber)
		runnerErr := runner.Run()
		Expect(runnerErr).NotTo(HaveOccurred())

		var dbResults []dbDiffResult
		getDbResultsErr := db.Select(&dbResults, `SELECT block_height, block_hash, hashed_address, storage_key, storage_value FROM public.storage_diff`)
		Expect(getDbResultsErr).NotTo(HaveOccurred())

		trimmedHeaderHash := strings.TrimPrefix(fakeHeader.Hash, "0x")
		headerHashBytes := common.Hex2Bytes(trimmedHeaderHash)
		expectedDiffResult := dbDiffResult{
			BlockHeight:   int(blockNumber),
			BlockHash:     headerHashBytes,
			HashedAddress: crypto.Keccak256Hash(addressOne[:]).Bytes(),
			StorageKey:    keyOne[:],
			StorageValue:  value1[:],
		}
		Expect(len(dbResults)).To(Equal(1))
		Expect(dbResults).To(ConsistOf(expectedDiffResult))

		//Run the command again with the same storage info
		runnerErrTwo := runner.Run()
		Expect(runnerErrTwo).NotTo(HaveOccurred())

		var dbResultsTwo []dbDiffResult
		getDbResultsErrTwo := db.Select(&dbResultsTwo, `SELECT block_height, block_hash, hashed_address, storage_key, storage_value FROM public.storage_diff`)
		Expect(getDbResultsErrTwo).NotTo(HaveOccurred())
		Expect(len(dbResults)).To(Equal(1))
		Expect(dbResults).To(ConsistOf(expectedDiffResult))
	})

	It("returns an error if a header for the given block cannot be retrieved", func() {
		runner := cmd.NewStorageValueCommandRunner(&bc, db, initializers, blockNumber+1)
		runnerErr := runner.Run()
		Expect(runnerErr).To(HaveOccurred())
		Expect(runnerErr).To(Equal(sql.ErrNoRows))
	})
})

type dbDiffResult struct {
	BlockHeight   int    `db:"block_height"`
	BlockHash     []byte `db:"block_hash"`
	HashedAddress []byte `db:"hashed_address"`
	StorageKey    []byte `db:"storage_key"`
	StorageValue  []byte `db:"storage_value"`
}
