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
	"github.com/makerdao/vulcanizedb/libraries/shared/storage/types"
	"github.com/makerdao/vulcanizedb/libraries/shared/transformer"
	"github.com/makerdao/vulcanizedb/pkg/core"
	"github.com/makerdao/vulcanizedb/pkg/fakes"
	"github.com/makerdao/vulcanizedb/test_config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("getStorageValue Command", func() {
	var (
		bc                             *fakes.MockBlockChain
		db                             = test_config.NewTestDB(test_config.NewTestNode())
		keysLookupOne, keysLookupTwo   mocks.MockStorageKeysLookup
		runner                         cmd.StorageValueCommandRunner
		initializerOne, initializerTwo transformer.StorageTransformerInitializer
		initializers                   []transformer.StorageTransformerInitializer
		keyOne, keyTwo                 common.Hash
		valueOne, valueTwo             common.Hash
		addressOne, addressTwo         common.Address
		blockNumber                    int64
		bigIntBlockNumber              *big.Int
		fakeHeader                     core.Header
		headerRepo                     fakes.MockHeaderRepository
		diffRepo                       mocks.MockStorageDiffRepository
	)

	BeforeEach(func() {
		bc = fakes.NewMockBlockChain()

		keysLookupOne = mocks.MockStorageKeysLookup{}
		keyOne = common.Hash{1, 2, 3}
		addressOne = fakes.FakeAddress
		keysLookupOne.SetKeysToReturn([]common.Hash{keyOne})
		valueOne = common.BytesToHash([]byte{7, 8, 9})
		bc.SetStorageValuesToReturn(addressOne, valueOne[:])

		keysLookupTwo = mocks.MockStorageKeysLookup{}
		keyTwo = common.Hash{4, 5, 6}
		addressTwo = fakes.AnotherFakeAddress
		keysLookupTwo.SetKeysToReturn([]common.Hash{keyTwo})
		valueTwo = common.BytesToHash([]byte{10, 11, 12})
		bc.SetStorageValuesToReturn(addressTwo, valueTwo[:])

		initializerOne = storage.Transformer{
			Address:           addressOne,
			StorageKeysLookup: &keysLookupOne,
			Repository:        &mocks.MockStorageRepository{},
		}.NewTransformer

		initializerTwo = storage.Transformer{
			Address:           addressTwo,
			StorageKeysLookup: &keysLookupTwo,
			Repository:        &mocks.MockStorageRepository{},
		}.NewTransformer

		initializers = []transformer.StorageTransformerInitializer{initializerOne, initializerTwo}
		blockNumber = rand.Int63()
		bigIntBlockNumber = big.NewInt(blockNumber)

		runner = cmd.NewStorageValueCommandRunner(bc, db, initializers, blockNumber)

		diffRepo = mocks.MockStorageDiffRepository{}
		runner.StorageDiffRepo = &diffRepo

		headerRepo = fakes.MockHeaderRepository{}
		fakeHeader = fakes.FakeHeader
		fakeHeader.BlockNumber = blockNumber
		headerRepo.GetHeaderReturnHash = fakeHeader.Hash
		runner.HeaderRepo = &headerRepo
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

	It("returns an error if getting the keys from the KeysLookup fails", func() {
		keysLookupTwo.SetGetKeysError(fakes.FakeError)

		runnerErr := runner.Run()
		Expect(keysLookupOne.GetKeysCalled).To(BeTrue())
		Expect(runnerErr).To(HaveOccurred())
		Expect(runnerErr).To(Equal(fakes.FakeError))
	})

	It("fetches the header by the given block number", func() {
		runnerErr := runner.Run()
		Expect(runnerErr).NotTo(HaveOccurred())
		Expect(headerRepo.GetHeaderPassedBlockNumber).To(Equal(blockNumber))
	})

	It("returns an error if a header for the given block cannot be retrieved", func() {
		headerRepo.GetHeaderError = fakes.FakeError
		runnerErr := runner.Run()
		Expect(runnerErr).To(HaveOccurred())
		Expect(runnerErr).To(Equal(fakes.FakeError))
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

	It("returns an error if blockchain call to GetStorageAt fails", func() {
		keysLookupOne.SetKeysToReturn([]common.Hash{keyOne})
		bc.SetGetStorageAtError(fakes.FakeError)

		runnerErr := runner.Run()
		Expect(keysLookupOne.GetKeysCalled).To(BeTrue())
		Expect(runnerErr).To(HaveOccurred())
		Expect(runnerErr).To(Equal(fakes.FakeError))
	})

	It("persists the storage values for each transformer", func() {
		runnerErr := runner.Run()
		Expect(runnerErr).NotTo(HaveOccurred())

		trimmedHeaderHash := strings.TrimPrefix(fakeHeader.Hash, "0x")
		headerHashBytes := common.HexToHash(trimmedHeaderHash)
		expectedDiffOne :=	types.RawDiff{
			BlockHeight:   int(blockNumber),
			BlockHash:     headerHashBytes,
			HashedAddress: crypto.Keccak256Hash(addressOne[:]),
			StorageKey:    keyOne,
			StorageValue:  valueOne,
		}
		expectedDiffTwo := types.RawDiff{
			BlockHeight:   int(blockNumber),
			BlockHash:     headerHashBytes,
			HashedAddress: crypto.Keccak256Hash(addressTwo[:]),
			StorageKey:    keyTwo,
			StorageValue:  valueTwo,
		}

		Expect(diffRepo.CreatePassedRawDiffs).To(ConsistOf(expectedDiffOne, expectedDiffTwo))
	})

	It("ignores sql.ErrNoRows error for duplicate diffs", func() {
		diffRepo.SetCreateError(sql.ErrNoRows)
		runnerErr := runner.Run()
		Expect(runnerErr).NotTo(HaveOccurred())
	})

	It("returns an error if inserting a diff fails", func() {
		diffRepo.SetCreateError(fakes.FakeError)
		runnerErr := runner.Run()
		Expect(runnerErr).To(HaveOccurred())
	})
})
