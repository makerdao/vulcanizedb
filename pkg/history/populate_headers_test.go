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

package history_test

import (
	"math/big"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/makerdao/vulcanizedb/pkg/fakes"
	"github.com/makerdao/vulcanizedb/pkg/history"
)

var _ = Describe("Populating headers", func() {
	var (
		headerRepository *fakes.MockHeaderRepository
		statusWriter     fakes.MockStatusWriter
		startingBlock,
		validationWindowSize int64
	)

	BeforeEach(func() {
		headerRepository = fakes.NewMockHeaderRepository()
		statusWriter = fakes.MockStatusWriter{}
		startingBlock = 1
		validationWindowSize = 15
	})

	It("returns number of headers added", func() {
		blockChain := fakes.NewMockBlockChain()
		blockChain.SetChainHead(big.NewInt(startingBlock + 1))
		headerRepository.SetMissingBlockNumbers([]int64{startingBlock + 1})

		numHeadersAdded, err := history.PopulateMissingHeaders(blockChain, headerRepository, startingBlock, validationWindowSize)

		Expect(err).NotTo(HaveOccurred())
		Expect(numHeadersAdded).To(Equal(1))
	})

	It("adds missing headers to the db", func() {
		blockChain := fakes.NewMockBlockChain()
		blockChain.SetChainHead(big.NewInt(startingBlock + 1))
		headerRepository.SetMissingBlockNumbers([]int64{startingBlock + 1})

		_, err := history.PopulateMissingHeaders(blockChain, headerRepository, startingBlock, validationWindowSize)

		Expect(err).NotTo(HaveOccurred())
		headerRepository.AssertCreateOrUpdateHeaderCallCountAndPassedBlockNumbers(1, []int64{2})
	})

	It("queries headers table for missing headers until beginning validation window (not chain head)", func() {
		blockChain := fakes.NewMockBlockChain()
		blockChain.SetChainHead(big.NewInt(startingBlock + validationWindowSize))
		_, err := history.PopulateMissingHeaders(blockChain, headerRepository, startingBlock, validationWindowSize)

		Expect(err).NotTo(HaveOccurred())
		Expect(headerRepository.MissingBlockNumbersPassedStartingBlock).To(Equal(startingBlock))
		Expect(headerRepository.MissingBlockNumbersPassedEndingBlock).To(Equal(startingBlock))
	})

	It("doesn't query for numbers less than starting block", func() {
		blockChain := fakes.NewMockBlockChain()
		blockChain.SetChainHead(big.NewInt(startingBlock))
		_, err := history.PopulateMissingHeaders(blockChain, headerRepository, startingBlock, validationWindowSize)

		Expect(err).NotTo(HaveOccurred())
		Expect(headerRepository.MissingBlockNumbersPassedStartingBlock).To(Equal(startingBlock))
		Expect(headerRepository.MissingBlockNumbersPassedEndingBlock).To(Equal(startingBlock))
	})

	It("returns early if the db is already synced up to the beginning of the validation window", func() {
		blockChain := fakes.NewMockBlockChain()
		blockChain.SetChainHead(big.NewInt(startingBlock))
		headersAdded, err := history.PopulateMissingHeaders(blockChain, headerRepository, startingBlock, validationWindowSize)

		Expect(err).NotTo(HaveOccurred())
		Expect(headersAdded).To(Equal(0))
	})

	It("does not write a healthcheck file when the call to get the last block fails", func() {
		blockChain := fakes.NewMockBlockChain()
		blockChain.SetChainHeadError(fakes.FakeError)

		_, err := history.PopulateMissingHeaders(blockChain, headerRepository, startingBlock, validationWindowSize)

		Expect(err).To(HaveOccurred())
		Expect(err).To(MatchError(fakes.FakeError))
		Expect(statusWriter.WriteCalled).To(BeFalse())
	})

	Describe("RetrieveAndUpdateHeaders", func() {
		It("returns an error if getting headers from the blockchain fails", func(){
			blockChain := fakes.NewMockBlockChain()
			blockChain.GetHeadersByNumbersErr = fakes.FakeError

			_, err := history.RetrieveAndUpdateHeaders(blockChain, headerRepository, []int64{})

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))
		})

		It("returns an error when Creating or updating the header fails", func() {
			blockChain := fakes.NewMockBlockChain()
			headerRepository.SetCreateOrUpdateHeaderReturnErr(fakes.FakeError)
			_, err := history.RetrieveAndUpdateHeaders(blockChain, headerRepository, []int64{startingBlock})

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))
		})
	})

})
