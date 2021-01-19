package history

import (
	"math/big"
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/makerdao/vulcanizedb/libraries/shared/mocks"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage/types"
	"github.com/makerdao/vulcanizedb/libraries/shared/test_data"
	"github.com/makerdao/vulcanizedb/pkg/core"
	"github.com/makerdao/vulcanizedb/pkg/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Storage Diff Validator", func() {
	var (
		diffRepository mocks.MockStorageDiffRepository
		blockChain     *fakes.MockBlockChain
		windowSize     int
		chainHead      *big.Int
	)

	BeforeEach(func() {
		diffRepository = mocks.MockStorageDiffRepository{}
		blockChain = fakes.NewMockBlockChain()
		windowSize = 2
		chainHead = big.NewInt(rand.Int63())
		blockChain.SetChainHead(chainHead)
	})

	It("gets all diffs with blocks heights in the validation window", func() {
		validationWindowStartingBlock := chainHead.Int64() - int64(windowSize)
		validationWindowEndingBlock := chainHead.Int64()

		validator := NewStorageDiffValidator(blockChain, &diffRepository, windowSize)
		err := validator.ValidateDiffs()
		Expect(err).NotTo(HaveOccurred())

		Expect(diffRepository.GetDiffsForRangeStartingHeightPassed).To(Equal(validationWindowStartingBlock))
		Expect(diffRepository.GetDiffsForRangeEndingHeightPassed).To(Equal(validationWindowEndingBlock))
	})

	It("returns an error if getting headers in the validation window fails", func() {
		blockChain.GetHeadersByNumbersError = fakes.FakeError

		validator := NewStorageDiffValidator(blockChain, &diffRepository, windowSize)
		err := validator.ValidateDiffs()
		Expect(err).To(HaveOccurred())
		Expect(err).To(Equal(fakes.FakeError))
	})

	It("returns an error if getting diffs for block range fails", func() {
		diffRepository.GetDiffsForRangeErr = fakes.FakeError

		validator := NewStorageDiffValidator(blockChain, &diffRepository, windowSize)
		err := validator.ValidateDiffs()
		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(fakes.FakeError))
	})

	Context("comparing diff hash and header hash to determine if the diff is canonical", func() {
		var (
			blockHeight                      int64
			headerHash, noncanonicalDiffHash common.Hash
			noncanonicalDiffID               int64
		)

		BeforeEach(func() {
			blockHeight = chainHead.Int64()
			headerHash = test_data.FakeHash()
			noncanonicalDiffHash = test_data.FakeHash()
			Expect(headerHash).NotTo(Equal(noncanonicalDiffHash))

			header := core.Header{BlockNumber: blockHeight, Hash: headerHash.Hex()}
			blockChain.GetHeadersByNumbersToReturn = []core.Header{header}

			noncanonicalDiffID = rand.Int63()
			rawDiff := types.RawDiff{BlockHash: noncanonicalDiffHash, BlockHeight: int(blockHeight)}
			diff := types.PersistedDiff{
				RawDiff: rawDiff,
				Status:  storage.Transformed,
				ID:      noncanonicalDiffID,
			}
			diffRepository.GetDiffsForRangeToReturn = []types.PersistedDiff{diff}
		})

		It("marks the diff as PENDING if it is noncanonical and was previously TRANSFORMED", func() {
			validator := NewStorageDiffValidator(blockChain, &diffRepository, windowSize)
			err := validator.ValidateDiffs()
			Expect(err).NotTo(HaveOccurred())
			Expect(diffRepository.MarkPendingPassedID).To(Equal(noncanonicalDiffID))
		})

		It("does not change the diff status if it is noncanoical and is still NEW", func() {
			rawDiff := types.RawDiff{BlockHash: noncanonicalDiffHash, BlockHeight: int(blockHeight)}
			diff := types.PersistedDiff{
				RawDiff: rawDiff,
				Status:  storage.New,
				ID:      noncanonicalDiffID,
			}
			diffRepository.GetDiffsForRangeToReturn = []types.PersistedDiff{diff}

			validator := NewStorageDiffValidator(blockChain, &diffRepository, windowSize)
			err := validator.ValidateDiffs()
			Expect(err).NotTo(HaveOccurred())
			Expect(diffRepository.MarkPendingPassedID).NotTo(Equal(noncanonicalDiffID))
		})

		It("returns an error if marking the diff as pending fails", func() {
			diffRepository.MarkPendingError = fakes.FakeError

			validator := NewStorageDiffValidator(blockChain, &diffRepository, windowSize)
			err := validator.ValidateDiffs()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))
		})

		It("doesn't change the diff status if it is canonical", func() {
			canonicalDiffID := rand.Int63()
			rawDiff := types.RawDiff{BlockHash: headerHash, BlockHeight: int(blockHeight)}
			diff := types.PersistedDiff{ID: noncanonicalDiffID, RawDiff: rawDiff}
			diffRepository.GetDiffsForRangeToReturn = []types.PersistedDiff{diff}

			validator := NewStorageDiffValidator(blockChain, &diffRepository, windowSize)
			err := validator.ValidateDiffs()
			Expect(err).NotTo(HaveOccurred())
			Expect(diffRepository.MarkPendingPassedID).NotTo(Equal(canonicalDiffID))
		})
	})
})
