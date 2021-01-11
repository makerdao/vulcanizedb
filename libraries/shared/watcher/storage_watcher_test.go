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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/makerdao/vulcanizedb/libraries/shared/factories/storage"
	"github.com/makerdao/vulcanizedb/libraries/shared/mocks"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage/types"
	"github.com/makerdao/vulcanizedb/libraries/shared/test_data"
	"github.com/makerdao/vulcanizedb/libraries/shared/watcher"
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres"
	"github.com/makerdao/vulcanizedb/pkg/fakes"
	"github.com/makerdao/vulcanizedb/test_config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const ThrottleTime = time.Duration(30)

var _ = Describe("Storage Watcher", func() {
	var statusWriter fakes.MockStatusWriter
	Describe("AddTransformer", func() {
		It("adds transformers", func() {
			fakeAddress := fakes.FakeAddress
			fakeTransformer := &mocks.MockStorageTransformer{Address: fakeAddress}
			w := watcher.NewStorageWatcher(test_config.NewTestDB(test_config.NewTestNode()), -1, &statusWriter, ThrottleTime)

			w.AddTransformers([]storage.TransformerInitializer{fakeTransformer.FakeTransformerInitializer})

			Expect(w.AddressTransformers[fakeAddress]).To(Equal(fakeTransformer))
		})
	})

	Describe("Execute", func() {
		When("a watcher is configured to watches 'new' storage diffs", func() {
			statusWriter := fakes.MockStatusWriter{}
			storageWatcher := watcher.NewStorageWatcher(test_config.NewTestDB(test_config.NewTestNode()), -1, &statusWriter, ThrottleTime)
			input := ExecuteInput{
				watcher:      &storageWatcher,
				statusWriter: &statusWriter,
			}
			SharedExecuteBehavior(&input)
		})

		When("a watcher is configured to watches 'unrecognized' storage diffs", func() {
			statusWriter := fakes.MockStatusWriter{}
			storageWatcher := watcher.UnrecognizedStorageWatcher(test_config.NewTestDB(test_config.NewTestNode()), -1, &statusWriter, ThrottleTime)
			input := ExecuteInput{
				watcher:      &storageWatcher,
				statusWriter: &statusWriter,
			}
			SharedExecuteBehavior(&input)
		})

		When("a watcher is configured to watch 'pending' storage diffs", func() {
			statusWriter := fakes.MockStatusWriter{}
			storageWatcher := watcher.PendingStorageWatcher(test_config.NewTestDB(test_config.NewTestNode()), -1, &statusWriter, ThrottleTime)
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

type MockThrottler struct {
	sleepTime  time.Duration
	calledWith watcher.Callback
}

func (throttler *MockThrottler) throttle(sleep time.Duration, f watcher.Callback) error {
	throttler.sleepTime = sleep
	throttler.calledWith = f
	return f()
}

func SharedExecuteBehavior(input *ExecuteInput) {
	var (
		mockDiffsRepository  *mocks.MockStorageDiffRepository
		mockHeaderRepository *fakes.MockHeaderRepository
		statusWriter         = input.statusWriter
		storageWatcher       = input.watcher
		contractAddress      common.Address
		mockTransformer      *mocks.MockStorageTransformer
		mockThrottler        = MockThrottler{}
	)

	BeforeEach(func() {
		mockDiffsRepository = &mocks.MockStorageDiffRepository{}
		mockHeaderRepository = &fakes.MockHeaderRepository{}
		contractAddress = test_data.FakeAddress()
		mockTransformer = &mocks.MockStorageTransformer{Address: contractAddress}
		storageWatcher.Throttler = mockThrottler.throttle
		storageWatcher.HeaderRepository = mockHeaderRepository
		storageWatcher.StorageDiffRepository = mockDiffsRepository
		storageWatcher.AddTransformers([]storage.TransformerInitializer{mockTransformer.FakeTransformerInitializer})
	})

	Describe("Execute", func() {
		It("creates file for health check", func() {
			setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{fakes.FakeError})

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))
			Expect(statusWriter.WriteCalled).To(BeTrue())
		})

		It("fetches diffs with results limit", func() {
			setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{fakes.FakeError})

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))
			assertGetDiffsLimits(storageWatcher.DiffStatus, mockDiffsRepository, watcher.ResultsLimit)
		})

		It("throttles the transform calls to the passed in diffs", func() {
			setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{fakes.FakeError})

			storageWatcher.Execute()

			Expect(mockThrottler.sleepTime).To(Equal(ThrottleTime))
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
			setDiffsToReturn(storageWatcher.DiffStatus, mockDiffsRepository, diffs)
			setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil, fakes.FakeError})

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))
			assertGetDiffsMinIDs(storageWatcher.DiffStatus, mockDiffsRepository, []int{0, diffID})
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
			setDiffsToReturn(storageWatcher.DiffStatus, mockDiffsRepository, diffs)
			setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil, fakes.FakeError})

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))
			assertGetDiffsMinIDs(storageWatcher.DiffStatus, mockDiffsRepository, []int{0, 0})
		})

		It("marks diff as 'unwatched' if no transformer is watching its address", func() {
			unwatchedAddress := test_data.FakeAddress()
			unwatchedDiff := types.PersistedDiff{
				RawDiff: types.RawDiff{
					Address: unwatchedAddress,
				},
				ID: rand.Int63(),
			}
			setDiffsToReturn(storageWatcher.DiffStatus, mockDiffsRepository, []types.PersistedDiff{unwatchedDiff})
			setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil, fakes.FakeError})

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))
			Expect(mockDiffsRepository.MarkUnwatchedPassedID).To(Equal(unwatchedDiff.ID))
		})

		It("does not change a diff's status if there's no header for the given block number", func() {
			diffWithoutHeader := types.PersistedDiff{
				RawDiff: types.RawDiff{
					Address:     contractAddress,
					BlockHash:   test_data.FakeHash(),
					BlockHeight: rand.Int(),
				},
				ID: rand.Int63(),
			}
			setDiffsToReturn(storageWatcher.DiffStatus, mockDiffsRepository, []types.PersistedDiff{diffWithoutHeader})
			setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil, fakes.FakeError})
			mockHeaderRepository.GetHeaderByBlockNumberError = postgres.ErrHeaderDoesNotExist

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))
			Expect(mockDiffsRepository.MarkTransformedPassedID).NotTo(Equal(diffWithoutHeader.ID))
		})

		It("does not return postgres.ErrHeaderDoesNotExist if header for diff not found", func() {
			diffWithoutHeader := types.PersistedDiff{
				RawDiff: types.RawDiff{
					Address:     contractAddress,
					BlockHash:   test_data.FakeHash(),
					BlockHeight: rand.Int(),
				},
				ID: rand.Int63(),
			}
			setDiffsToReturn(storageWatcher.DiffStatus, mockDiffsRepository, []types.PersistedDiff{diffWithoutHeader})
			setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil, fakes.FakeError})
			mockHeaderRepository.GetHeaderByBlockNumberError = postgres.ErrHeaderDoesNotExist

			err := storageWatcher.Execute()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))
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
				setDiffsToReturn(storageWatcher.DiffStatus, mockDiffsRepository, diffs)
				setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil, fakes.FakeError})

				expectedFirstMinDiffID := int(diffs[0].ID - 1)
				expectedSecondMinDiffID := int(diffs[len(diffs)-1].ID)

				err := storageWatcher.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fakes.FakeError))
				Expect(mockDiffsRepository.GetFirstDiffBlockHeightPassed).To(Equal(headerBlockNumber - numberOfBlocksFromHeadOfChain))
				assertGetDiffsMinIDs(storageWatcher.DiffStatus, mockDiffsRepository, []int{expectedFirstMinDiffID, expectedSecondMinDiffID})
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
				setDiffsToReturn(storageWatcher.DiffStatus, mockDiffsRepository, diffs)
				setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil, fakes.FakeError})

				expectedFirstMinDiffID := int(diffs[0].ID - 1)

				err := storageWatcher.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fakes.FakeError))
				Expect(mockDiffsRepository.GetFirstDiffBlockHeightPassed).To(Equal(headerBlockNumber - numberOfBlocksFromHeadOfChain))
				assertGetDiffsMinIDs(storageWatcher.DiffStatus, mockDiffsRepository, []int{expectedFirstMinDiffID, expectedFirstMinDiffID})
			})

			It("sets minID to 0 if there are no headers with the given block height", func() {
				mockHeaderRepository.MostRecentHeaderBlockNumberErr = sql.ErrNoRows
				setDiffsToReturn(storageWatcher.DiffStatus, mockDiffsRepository, diffs)
				setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil, fakes.FakeError})
				err := storageWatcher.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fakes.FakeError))

				expectedFirstMinDiffID := 0
				expectedSecondMinDiffID := int(diffs[len(diffs)-1].ID)
				assertGetDiffsMinIDs(storageWatcher.DiffStatus, mockDiffsRepository, []int{expectedFirstMinDiffID, expectedSecondMinDiffID})
			})

			It("sets minID to 0 if there are no diffs with given block range", func() {
				mockDiffsRepository.GetFirstDiffIDErr = sql.ErrNoRows
				setDiffsToReturn(storageWatcher.DiffStatus, mockDiffsRepository, diffs)
				setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil, fakes.FakeError})
				err := storageWatcher.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fakes.FakeError))

				expectedFirstMinDiffID := 0
				expectedSecondMinDiffID := int(diffs[len(diffs)-1].ID)
				assertGetDiffsMinIDs(storageWatcher.DiffStatus, mockDiffsRepository, []int{expectedFirstMinDiffID, expectedSecondMinDiffID})
			})
		})

		Describe("When a header with a non-matching hash is found", func() {
			var (
				blockNumber       int
				fakePersistedDiff types.PersistedDiff
				fakeRawDiff       types.RawDiff
			)

			BeforeEach(func() {
				blockNumber = rand.Int()
				fakeRawDiff = types.RawDiff{
					Address:      contractAddress,
					BlockHash:    test_data.FakeHash(),
					BlockHeight:  blockNumber,
					StorageKey:   test_data.FakeHash(),
					StorageValue: test_data.FakeHash(),
				}
				mockHeaderRepository.GetHeaderByBlockNumberReturnID = int64(blockNumber)
				mockHeaderRepository.GetHeaderByBlockNumberReturnHash = test_data.FakeHash().Hex()

				fakePersistedDiff = types.PersistedDiff{
					RawDiff: fakeRawDiff,
					ID:      rand.Int63(),
				}
				setDiffsToReturn(storageWatcher.DiffStatus, mockDiffsRepository, []types.PersistedDiff{fakePersistedDiff})
			})

			It("does not change a diff's status if getting max known block height fails", func() {
				maxHeaderErr := errors.New("getting max header failed")
				mockHeaderRepository.MostRecentHeaderBlockNumberErr = maxHeaderErr
				setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil})

				err := storageWatcher.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(maxHeaderErr))
				Expect(mockDiffsRepository.MarkTransformedPassedID).NotTo(Equal(fakePersistedDiff.ID))
			})

			It("marks diff 'noncanonical' if block height less than max known block height minus reorg window", func() {
				mockHeaderRepository.MostRecentHeaderBlockNumber = int64(blockNumber + watcher.ReorgWindow + 1)
				setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil, fakes.FakeError})

				err := storageWatcher.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fakes.FakeError))
				Expect(mockDiffsRepository.MarkNoncanonicalPassedID).To(Equal(fakePersistedDiff.ID))
			})

			It("marks diff as 'pending' if block height is within reorg window - because we aren't sure its noncanonical yet", func() {
				mockHeaderRepository.MostRecentHeaderBlockNumber = int64(blockNumber + watcher.ReorgWindow)
				setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil, fakes.FakeError})

				err := storageWatcher.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fakes.FakeError))
				Expect(mockDiffsRepository.MarkPendingPassedID).To(Equal(fakePersistedDiff.ID))
			})

			It("doesn't 're-mark' a diff as 'pending' if block height is within reorg window and it's already pending", func() {
				fakePersistedDiff = types.PersistedDiff{
					RawDiff: fakeRawDiff,
					ID:      rand.Int63(),
					Status:  "pending",
				}
				setDiffsToReturn(storageWatcher.DiffStatus, mockDiffsRepository, []types.PersistedDiff{fakePersistedDiff})
				mockHeaderRepository.MostRecentHeaderBlockNumber = int64(blockNumber + watcher.ReorgWindow)
				setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil, fakes.FakeError})

				err := storageWatcher.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fakes.FakeError))
				Expect(mockDiffsRepository.MarkPendingPassedID).To(Equal(int64(0)))
			})
		})

		Describe("When a header with a matching hash exists", func() {
			var fakePersistedDiff types.PersistedDiff

			BeforeEach(func() {
				fakeBlockHash := test_data.FakeHash()
				fakeRawDiff := types.RawDiff{
					Address:   contractAddress,
					BlockHash: fakeBlockHash,
				}

				mockHeaderRepository.GetHeaderByBlockNumberReturnID = rand.Int63()
				mockHeaderRepository.GetHeaderByBlockNumberReturnHash = fakeBlockHash.Hex()

				fakePersistedDiff = types.PersistedDiff{
					RawDiff: fakeRawDiff,
					ID:      rand.Int63(),
				}
				setDiffsToReturn(storageWatcher.DiffStatus, mockDiffsRepository, []types.PersistedDiff{fakePersistedDiff})
			})

			It("does not change diff's status if transformer execution fails", func() {
				executeErr := errors.New("execute failed")
				mockTransformer.ExecuteErr = executeErr
				setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil})

				err := storageWatcher.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(executeErr))
				Expect(mockDiffsRepository.MarkTransformedPassedID).NotTo(Equal(fakePersistedDiff.ID))
			})

			It("marks diff as 'unrecognized' when transforming the diff returns a ErrKeyNotFound error", func() {
				mockTransformer.ExecuteErr = types.ErrKeyNotFound
				setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil, fakes.FakeError})

				err := storageWatcher.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fakes.FakeError))
				Expect(mockDiffsRepository.MarkUnrecognizedPassedID).To(Equal(fakePersistedDiff.ID))
			})

			It("does not return ErrKeyNotFound if storage diff key not recognized", func() {
				mockTransformer.ExecuteErr = types.ErrKeyNotFound
				setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil, fakes.FakeError})

				err := storageWatcher.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fakes.FakeError))
			})

			It("marks diff transformed if transformer execution doesn't fail", func() {
				setDiffsToReturn(storageWatcher.DiffStatus, mockDiffsRepository, []types.PersistedDiff{fakePersistedDiff})
				setGetDiffsErrors(storageWatcher.DiffStatus, mockDiffsRepository, []error{nil, fakes.FakeError})

				err := storageWatcher.Execute()

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fakes.FakeError))
				Expect(mockDiffsRepository.MarkTransformedPassedID).To(Equal(fakePersistedDiff.ID))
			})
		})

	})
}

func setGetDiffsErrors(diffStatus watcher.DiffStatusToWatch, mockDiffsRepo *mocks.MockStorageDiffRepository, diffErrors []error) {
	switch diffStatus {
	case watcher.New:
		mockDiffsRepo.GetNewDiffsErrors = diffErrors
	case watcher.Unrecognized:
		mockDiffsRepo.GetUnrecognizedDiffsErrors = diffErrors
	case watcher.Pending:
		mockDiffsRepo.GetPendingDiffsErrors = diffErrors
	}
}
func setDiffsToReturn(diffStatus watcher.DiffStatusToWatch, mockDiffsRepo *mocks.MockStorageDiffRepository, diffs []types.PersistedDiff) {
	switch diffStatus {
	case watcher.New:
		mockDiffsRepo.GetNewDiffsToReturn = diffs
	case watcher.Unrecognized:
		mockDiffsRepo.GetUnrecognizedDiffsToReturn = diffs
	case watcher.Pending:
		mockDiffsRepo.GetPendingDiffsToReturn = diffs
	}
}

func assertGetDiffsLimits(diffStatus watcher.DiffStatusToWatch, mockDiffRepo *mocks.MockStorageDiffRepository, watcherLimit int) {
	switch diffStatus {
	case watcher.New:
		Expect(mockDiffRepo.GetNewDiffsPassedLimits).To(ConsistOf(watcherLimit))
	case watcher.Unrecognized:
		Expect(mockDiffRepo.GetUnrecognizedDiffsPassedLimits).To(ConsistOf(watcherLimit))
	case watcher.Pending:
		Expect(mockDiffRepo.GetPendingDiffsPassedLimits).To(ConsistOf(watcherLimit))
	}
}

func assertGetDiffsMinIDs(diffStatus watcher.DiffStatusToWatch, mockDiffRepo *mocks.MockStorageDiffRepository, passedMinIDs []int) {
	switch diffStatus {
	case watcher.New:
		Expect(mockDiffRepo.GetNewDiffsPassedMinIDs).To(ConsistOf(passedMinIDs))
	case watcher.Unrecognized:
		Expect(mockDiffRepo.GetUnrecognizedDiffsPassedMinIDs).To(ConsistOf(passedMinIDs))
	case watcher.Pending:
		Expect(mockDiffRepo.GetPendingDiffsPassedMinIDs).To(ConsistOf(passedMinIDs))
	}
}
