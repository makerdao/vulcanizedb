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

package watcher_test

import (
	"database/sql"
	"errors"
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/makerdao/vulcanizedb/libraries/shared/factories/storage"
	"github.com/makerdao/vulcanizedb/libraries/shared/mocks"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage/types"
	"github.com/makerdao/vulcanizedb/libraries/shared/test_data"
	"github.com/makerdao/vulcanizedb/libraries/shared/watcher"
	"github.com/makerdao/vulcanizedb/pkg/fakes"
	"github.com/makerdao/vulcanizedb/test_config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Storage Watcher", func() {
	var statusWriter fakes.MockStatusWriter
	Describe("AddTransformer", func() {
		It("adds transformers", func() {
			fakeAddress := fakes.FakeAddress
			fakeTransformer := &mocks.MockStorageTransformer{Address: fakeAddress}
			w := watcher.NewStorageWatcher(test_config.NewTestDB(test_config.NewTestNode()), -1, &statusWriter, watcher.New)

			w.AddTransformers([]storage.TransformerInitializer{fakeTransformer.FakeTransformerInitializer})

			Expect(w.AddressTransformers[fakeAddress]).To(Equal(fakeTransformer))
		})
	})

	Describe("Execute", func() {
		When("a watcher is configured to watches 'new' storage diffs", func() {
			statusWriter := fakes.MockStatusWriter{}
			storageWatcher := watcher.NewStorageWatcher(test_config.NewTestDB(test_config.NewTestNode()), -1, &statusWriter, watcher.New)
			input := ExecuteInput{
				watcher:      &storageWatcher,
				statusWriter: &statusWriter,
			}
			SharedExecuteBehavior(&input)
		})

		When("a watcher is configured to watches 'unrecognized' storage diffs", func() {
			statusWriter := fakes.MockStatusWriter{}
			storageWatcher := watcher.NewStorageWatcher(test_config.NewTestDB(test_config.NewTestNode()), -1, &statusWriter, watcher.Unrecognized)
			input := ExecuteInput{
				watcher:      &storageWatcher,
				statusWriter: &statusWriter,
			}
			SharedExecuteBehavior(&input)
		})
	})
})

type ExecuteInput struct {
	watcher      *watcher.StorageWatcher
	statusWriter *fakes.MockStatusWriter
}

func SharedExecuteBehavior(input *ExecuteInput) {
	var (
		mockDiffsRepository  *mocks.MockStorageDiffRepository
		mockHeaderRepository *fakes.MockHeaderRepository
		statusWriter         = input.statusWriter
		storageWatcher       = input.watcher
		contractAddress        common.Address
		mockTransformer      *mocks.MockStorageTransformer
	)

	BeforeEach(func() {
		mockDiffsRepository = &mocks.MockStorageDiffRepository{}
		mockHeaderRepository = &fakes.MockHeaderRepository{}
		contractAddress = test_data.FakeAddress()
		mockTransformer = &mocks.MockStorageTransformer{Address: contractAddress}
		storageWatcher.HeaderRepository = mockHeaderRepository
		storageWatcher.StorageDiffRepository = mockDiffsRepository
		storageWatcher.AddTransformers([]storage.TransformerInitializer{mockTransformer.FakeTransformerInitializer})
	})

	Describe("Execute", func() {
		It("creates file for health check", func() {
			setGetDiffsErrors(mockDiffsRepository, []error{fakes.FakeError})

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(statusWriter.WriteCalled).To(BeTrue())
		})

		It("fetches diffs with results limit", func() {
			setGetDiffsErrors(mockDiffsRepository, []error{fakes.FakeError})

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(mockDiffsRepository.GetNewDiffsPassedLimits).To(ConsistOf(watcher.ResultsLimit))
		})

		It("fetches diffs with min ID from subsequent queries when previous query returns max results", func() {
			var diffs []types.PersistedDiff
			diffID := rand.Int()
			for i := 0; i < watcher.ResultsLimit; i++ {
				diffID = diffID + i
				diff := types.PersistedDiff{
					RawDiff: types.RawDiff{
						Address: test_data.FakeAddress(),
					},
					ID: int64(diffID),
				}
				diffs = append(diffs, diff)
			}
			setDiffsToReturn(mockDiffsRepository, diffs)
			setGetDiffsErrors(mockDiffsRepository, []error{nil, fakes.FakeError})

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(mockDiffsRepository.GetNewDiffsPassedMinIDs).To(ConsistOf(0, diffID))
		})
	})

	It("resets min ID to zero when previous query returns fewer than max results", func() {
		var diffs []types.PersistedDiff
		diffID := rand.Int()
		for i := 0; i < watcher.ResultsLimit-1; i++ {
			diffID = diffID + i
			diff := types.PersistedDiff{
				RawDiff: types.RawDiff{
					Address: test_data.FakeAddress(),
				},
				ID: int64(diffID),
			}
			diffs = append(diffs, diff)
		}
		setDiffsToReturn(mockDiffsRepository, diffs)
		setGetDiffsErrors(mockDiffsRepository, []error{nil, fakes.FakeError})

		err := storageWatcher.Execute()

		Expect(err).To(HaveOccurred())
		Expect(mockDiffsRepository.GetNewDiffsPassedMinIDs).To(ConsistOf(0, 0))
	})

	It("marks diff as 'unwatched' if no transformer is watching its address", func() {
		unwatchedAddress := test_data.FakeAddress()
		unwatchedDiff := types.PersistedDiff{
			RawDiff: types.RawDiff{
				Address: unwatchedAddress,
			},
			ID: rand.Int63(),
		}
		setDiffsToReturn(mockDiffsRepository, []types.PersistedDiff{unwatchedDiff})
		setGetDiffsErrors(mockDiffsRepository, []error{nil, fakes.FakeError})

		err := storageWatcher.Execute()

		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(fakes.FakeError))
		Expect(mockDiffsRepository.MarkUnwatchedPassedID).To(Equal(unwatchedDiff.ID))
	})

	It("does not change a diff's status if there's no header for the given block number", func() {
		diffWithoutHeader := types.PersistedDiff{
			RawDiff: types.RawDiff{
				Address: contractAddress,
				BlockHash:     test_data.FakeHash(),
				BlockHeight:   rand.Int(),
			},
			ID: rand.Int63(),
		}
		setDiffsToReturn(mockDiffsRepository, []types.PersistedDiff{diffWithoutHeader})
		setGetDiffsErrors(mockDiffsRepository, []error{nil, fakes.FakeError})
		mockHeaderRepository.GetHeaderByBlockNumberError = sql.ErrNoRows

		err := storageWatcher.Execute()

		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(sql.ErrNoRows))
		Expect(mockDiffsRepository.MarkTransformedPassedID).NotTo(Equal(diffWithoutHeader.ID))
	})

	Describe("When the watcher is configured to skip old diffs", func() {
		var diffs []types.PersistedDiff
		var numberOfBlocksFromHeadOfChain = int64(500)

		BeforeEach(func() {
			storageWatcher.AddressTransformers = map[common.Address]storage.ITransformer{}
			storageWatcher.DiffBlocksFromHeadOfChain = numberOfBlocksFromHeadOfChain

			diffID := rand.Int()
			for i := 0; i < watcher.ResultsLimit; i++ {
				diffID = diffID + i
				diff := types.PersistedDiff{
					RawDiff: types.RawDiff{
						Address: test_data.FakeAddress(),
					},
					ID: int64(diffID),
				}
				diffs = append(diffs, diff)
			}
		})

		It("skips diffs that are from a block more than n from the head of the chain", func() {
			headerBlockNumber := rand.Int63()
			mockHeaderRepository.MostRecentHeaderBlockNumber = headerBlockNumber

			mockDiffsRepository.GetFirstDiffIDToReturn = diffs[0].ID
			setDiffsToReturn(mockDiffsRepository, diffs)
			setGetDiffsErrors(mockDiffsRepository, []error{nil, fakes.FakeError})

			expectedFirstMinDiffID := int(diffs[0].ID - 1)
			expectedSecondMinDiffID := int(diffs[len(diffs)-1].ID)

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))
			Expect(mockDiffsRepository.GetFirstDiffBlockHeightPassed).To(Equal(headerBlockNumber - numberOfBlocksFromHeadOfChain))
			Expect(mockDiffsRepository.GetNewDiffsPassedMinIDs).To(ConsistOf(expectedFirstMinDiffID, expectedSecondMinDiffID))
		})

		It("resets min ID back to new min diff when previous query returns fewer than max results", func() {
			var diffs []types.PersistedDiff
			diffID := rand.Int()
			for i := 0; i < watcher.ResultsLimit-1; i++ {
				diffID = diffID + i
				diff := types.PersistedDiff{
					RawDiff: types.RawDiff{
						Address: test_data.FakeAddress(),
					},
					ID: int64(diffID),
				}
				diffs = append(diffs, diff)
			}

			headerBlockNumber := rand.Int63()
			mockHeaderRepository.MostRecentHeaderBlockNumber = headerBlockNumber

			mockDiffsRepository.GetFirstDiffIDToReturn = diffs[0].ID
			setDiffsToReturn(mockDiffsRepository, diffs)
			setGetDiffsErrors(mockDiffsRepository, []error{nil, fakes.FakeError})

			expectedFirstMinDiffID := int(diffs[0].ID - 1)

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))
			Expect(mockDiffsRepository.GetFirstDiffBlockHeightPassed).To(Equal(headerBlockNumber - numberOfBlocksFromHeadOfChain))
			Expect(mockDiffsRepository.GetNewDiffsPassedMinIDs).To(ConsistOf(expectedFirstMinDiffID, expectedFirstMinDiffID))
		})

		It("sets minID to 0 if there are no headers with the given block height", func() {
			mockHeaderRepository.MostRecentHeaderBlockNumberErr = sql.ErrNoRows
			setDiffsToReturn(mockDiffsRepository, diffs)
			setGetDiffsErrors(mockDiffsRepository, []error{nil, fakes.FakeError})
			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))

			expectedFirstMinDiffID := 0
			expectedSecondMinDiffID := int(diffs[len(diffs)-1].ID)
			Expect(mockDiffsRepository.GetNewDiffsPassedMinIDs).To(ConsistOf(expectedFirstMinDiffID, expectedSecondMinDiffID))
		})

		It("sets minID to 0 if there are no diffs with given block range", func() {
			mockDiffsRepository.GetFirstDiffIDErr = sql.ErrNoRows
			setDiffsToReturn(mockDiffsRepository, diffs)
			setGetDiffsErrors(mockDiffsRepository, []error{nil, fakes.FakeError})
			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))

			expectedFirstMinDiffID := 0
			expectedSecondMinDiffID := int(diffs[len(diffs)-1].ID)
			Expect(mockDiffsRepository.GetNewDiffsPassedMinIDs).To(ConsistOf(expectedFirstMinDiffID, expectedSecondMinDiffID))
		})
	})

	Describe("When a header with a non-matching hash is found", func() {
		var (
			blockNumber       int
			fakePersistedDiff types.PersistedDiff
		)

		BeforeEach(func() {
			blockNumber = rand.Int()
			fakeRawDiff := types.RawDiff{
				Address: contractAddress,
				BlockHash:     test_data.FakeHash(),
				BlockHeight:   blockNumber,
				StorageKey:    test_data.FakeHash(),
				StorageValue:  test_data.FakeHash(),
			}
			mockHeaderRepository.GetHeaderByBlockNumberReturnID = int64(blockNumber)
			mockHeaderRepository.GetHeaderByBlockNumberReturnHash = test_data.FakeHash().Hex()

			fakePersistedDiff = types.PersistedDiff{
				RawDiff: fakeRawDiff,
				ID:      rand.Int63(),
			}
			mockDiffsRepository.GetNewDiffsDiffs = []types.PersistedDiff{fakePersistedDiff}
		})

		It("does not change a diff's status if getting max known block height fails", func() {
			maxHeaderErr := errors.New("getting max header failed")
			mockHeaderRepository.MostRecentHeaderBlockNumberErr = maxHeaderErr
			setGetDiffsErrors(mockDiffsRepository, []error{nil})

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(maxHeaderErr))
			Expect(mockDiffsRepository.MarkTransformedPassedID).NotTo(Equal(fakePersistedDiff.ID))
		})

		It("marks diff 'noncanonical' if block height less than max known block height minus reorg window", func() {
			mockHeaderRepository.MostRecentHeaderBlockNumber = int64(blockNumber + watcher.ReorgWindow + 1)
			setGetDiffsErrors(mockDiffsRepository, []error{nil, fakes.FakeError})

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))
			Expect(mockDiffsRepository.MarkNoncanonicalPassedID).To(Equal(fakePersistedDiff.ID))
		})

		It("does not mark diff as 'noncanonical' if block height is within reorg window", func() {
			mockHeaderRepository.MostRecentHeaderBlockNumber = int64(blockNumber + watcher.ReorgWindow)
			setGetDiffsErrors(mockDiffsRepository, []error{nil, fakes.FakeError})

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))
			Expect(mockDiffsRepository.MarkTransformedPassedID).NotTo(Equal(fakePersistedDiff.ID))
		})
	})

	Describe("When a header with a matching hash exists", func() {
		var fakePersistedDiff types.PersistedDiff

		BeforeEach(func() {
			fakeBlockHash := test_data.FakeHash()
			fakeRawDiff := types.RawDiff{
				Address: contractAddress,
				BlockHash:     fakeBlockHash,
			}

			mockHeaderRepository.GetHeaderByBlockNumberReturnID = rand.Int63()
			mockHeaderRepository.GetHeaderByBlockNumberReturnHash = fakeBlockHash.Hex()

			fakePersistedDiff = types.PersistedDiff{
				RawDiff: fakeRawDiff,
				ID:      rand.Int63(),
			}
			setDiffsToReturn(mockDiffsRepository, []types.PersistedDiff{fakePersistedDiff})
		})

		It("does not change diff's status if transformer execution fails", func() {
			executeErr := errors.New("execute failed")
			mockTransformer.ExecuteErr = executeErr
			setGetDiffsErrors(mockDiffsRepository, []error{nil})

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(executeErr))
			Expect(mockDiffsRepository.MarkTransformedPassedID).NotTo(Equal(fakePersistedDiff.ID))
		})

		It("marks diff as 'unrecognized' when transforming the diff returns a ErrKeyNotFound error", func() {
			mockTransformer.ExecuteErr = types.ErrKeyNotFound
			setGetDiffsErrors(mockDiffsRepository, []error{nil, types.ErrKeyNotFound})

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(types.ErrKeyNotFound))
			Expect(mockDiffsRepository.MarkUnrecognizedPassedID).To(Equal(fakePersistedDiff.ID))
		})

		It("marks diff transformed if transformer execution doesn't fail", func() {
			setDiffsToReturn(mockDiffsRepository, []types.PersistedDiff{fakePersistedDiff})
			setGetDiffsErrors(mockDiffsRepository, []error{nil, fakes.FakeError})

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))
			Expect(mockDiffsRepository.MarkTransformedPassedID).To(Equal(fakePersistedDiff.ID))
		})
	})
}

func setGetDiffsErrors(mockDiffsRepo *mocks.MockStorageDiffRepository, diffErrors []error) {
	mockDiffsRepo.GetNewDiffsErrors = diffErrors
}
func setDiffsToReturn(mockDiffsRepo *mocks.MockStorageDiffRepository, diffs []types.PersistedDiff) {
	mockDiffsRepo.GetNewDiffsDiffs = diffs
}
