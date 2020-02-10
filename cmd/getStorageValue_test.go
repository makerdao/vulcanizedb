package cmd_test

import (
	"math/big"
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/makerdao/vulcanizedb/cmd"
	"github.com/makerdao/vulcanizedb/libraries/shared/factories/storage"
	"github.com/makerdao/vulcanizedb/libraries/shared/mocks"
	"github.com/makerdao/vulcanizedb/libraries/shared/transformer"
	"github.com/makerdao/vulcanizedb/pkg/fakes"
	"github.com/makerdao/vulcanizedb/test_config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("getStorageValue Command", func() {
	var (
		bc                             = fakes.MockBlockChain{}
		db                             = test_config.NewTestDB(test_config.NewTestNode())
		keysLookupOne, keysLookupTwo   mocks.MockStorageKeysLookup
		repoOne, repoTwo               mocks.MockStorageRepository
		runner                         cmd.GetStorageValueRunner
		initializerOne, initializerTwo transformer.StorageTransformerInitializer
		initializers                   []transformer.StorageTransformerInitializer
		keyOne, keyTwo                 common.Hash
		addressOne, addressTwo         common.Address
		blockNumber *big.Int
	)

	BeforeEach(func() {
		runner = cmd.GetStorageValueRunner{}

		keysLookupOne = mocks.MockStorageKeysLookup{}
		keysLookupTwo = mocks.MockStorageKeysLookup{}
		repoOne = mocks.MockStorageRepository{}
		repoTwo = mocks.MockStorageRepository{}

		keyOne = common.Hash{1, 2, 3}
		keyTwo = common.Hash{4, 5, 6}

		addressOne = fakes.FakeAddress
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
		blockNumber = big.NewInt(rand.Int63())
	})

	It("gets the storage keys for each transformer", func() {
		runnerExecErr := runner.Execute(&bc, db, initializers, blockNumber)
		Expect(runnerExecErr).NotTo(HaveOccurred())

		Expect(keysLookupOne.GetKeysCalled).To(BeTrue())
		Expect(keysLookupTwo.GetKeysCalled).To(BeTrue())
	})

	It("gets the storage values for each of the transformer's keys", func() {
		keysLookupOne.SetKeysToReturn([]common.Hash{keyOne})
		keysLookupTwo.SetKeysToReturn([]common.Hash{keyTwo})

		runnerExecErr := runner.Execute(&bc, db, initializers, blockNumber)
		Expect(runnerExecErr).NotTo(HaveOccurred())
		Expect(keysLookupOne.GetKeysCalled).To(BeTrue())
		Expect(keysLookupTwo.GetKeysCalled).To(BeTrue())

		Expect(bc.GetStorageAtPassedBlockNumber).To(Equal(blockNumber))
		Expect(bc.GetStorageAtPassedAccounts).To(ConsistOf(addressOne, addressTwo))
		Expect(bc.GetStorageAtPassedKeys).To(ConsistOf(keyOne, keyTwo))
	})

	It("returns an error if getting the keys from the KeysLookup fails", func() {
		keysLookupTwo.SetGetKeysError(fakes.FakeError)

		runnerExecErr := runner.Execute(&bc, db, initializers, blockNumber)
		Expect(runnerExecErr).To(HaveOccurred())

		Expect(keysLookupOne.GetKeysCalled).To(BeTrue())
	})

	It("returns an error if blockchain call to GetStorageAt fails", func() {
		keysLookupOne.SetKeysToReturn([]common.Hash{keyOne})
		bc.SetGetStorageAtError(fakes.FakeError)

		runnerExecErr := runner.Execute(&bc, db, initializers, blockNumber)
		Expect(runnerExecErr).To(HaveOccurred())

		Expect(keysLookupOne.GetKeysCalled).To(BeTrue())
	})
})
