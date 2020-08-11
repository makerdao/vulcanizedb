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

package types_test

import (
	"math/big"
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage/types"
	"github.com/makerdao/vulcanizedb/libraries/shared/test_data"
	"github.com/makerdao/vulcanizedb/pkg/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Storage row parsing", func() {
	Describe("FromParityCsvRow", func() {
		It("converts an array of strings to a row struct", func() {
			contract := "0x123"
			blockHash := "0x456"
			blockHeight := "789"
			storageKey := "0x987"
			storageValue := "0x654"
			data := []string{contract, blockHash, blockHeight, storageKey, storageValue}

			result, err := types.FromParityCsvRow(data)

			Expect(err).NotTo(HaveOccurred())
			expectedKeccakOfContractAddress := types.HexToKeccak256Hash(contract)
			Expect(result.HashedAddress).To(Equal(expectedKeccakOfContractAddress))
			Expect(result.BlockHash).To(Equal(common.HexToHash(blockHash)))
			Expect(result.BlockHeight).To(Equal(789))
			Expect(result.StorageKey).To(Equal(common.HexToHash(storageKey)))
			Expect(result.StorageValue).To(Equal(common.HexToHash(storageValue)))
		})

		It("returns an error if row is missing data", func() {
			_, err := types.FromParityCsvRow([]string{"0x123"})

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(types.ErrRowMalformed{Length: 1}))
		})

		It("returns error if block height malformed", func() {
			_, err := types.FromParityCsvRow([]string{"", "", "", "", ""})

			Expect(err).To(HaveOccurred())
		})
	})

	Describe("FromOldGethStateDiff", func() {
		var (
			accountDiff = filters.AccountDiff{Key: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}}
			stateDiff   = &filters.StateDiff{
				BlockNumber: big.NewInt(rand.Int63()),
				BlockHash:   fakes.FakeHash,
			}
		)

		It("adds relevant fields to diff", func() {
			storageValueBytes := []byte{3}
			storageValueRlp, encodeErr := rlp.EncodeToBytes(storageValueBytes)
			Expect(encodeErr).NotTo(HaveOccurred())

			storageDiff := filters.StorageDiff{
				Key:   []byte{0, 9, 8, 7, 6, 5, 4, 3, 2, 1},
				Value: storageValueRlp,
			}

			result, err := types.FromOldGethStateDiff(accountDiff, stateDiff, storageDiff)
			Expect(err).NotTo(HaveOccurred())

			expectedAddress := common.BytesToHash(accountDiff.Key)
			Expect(result.HashedAddress).To(Equal(expectedAddress))
			Expect(result.BlockHash).To(Equal(fakes.FakeHash))
			expectedBlockHeight := int(stateDiff.BlockNumber.Int64())
			Expect(result.BlockHeight).To(Equal(expectedBlockHeight))
			expectedStorageKey := common.BytesToHash(storageDiff.Key)
			Expect(result.StorageKey).To(Equal(expectedStorageKey))
			expectedStorageValue := common.BytesToHash(storageValueBytes)
			Expect(result.StorageValue).To(Equal(expectedStorageValue))
		})

		It("handles decoding large storage values from their RLP", func() {
			storageValueBytes := []byte{1, 2, 3, 4, 5, 0, 9, 8, 7, 6}
			storageValueRlp, encodeErr := rlp.EncodeToBytes(storageValueBytes)
			Expect(encodeErr).NotTo(HaveOccurred())

			storageDiff := filters.StorageDiff{
				Key:   []byte{0, 9, 8, 7, 6, 5, 4, 3, 2, 1},
				Value: storageValueRlp,
			}

			result, err := types.FromOldGethStateDiff(accountDiff, stateDiff, storageDiff)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.StorageValue).To(Equal(common.BytesToHash(storageValueBytes)))
		})

		It("returns an err if decoding the storage value Rlp fails", func() {
			_, err := types.FromOldGethStateDiff(accountDiff, stateDiff, test_data.StorageWithBadValue)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("rlp: input contains more than one value"))
		})
	})

	Describe("FromNewGethStateDiff", func() {
		var (
			accountDiff = filters.AccountDiff{Key: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 0}}
			stateDiff   = &filters.StateDiff{
				BlockNumber: big.NewInt(rand.Int63()),
				BlockHash:   fakes.FakeHash,
			}
		)

		It("adds relevant fields to diff", func() {
			storageValueBytes := []byte{3}
			storageValueRlp, encodeErr := rlp.EncodeToBytes(storageValueBytes)
			Expect(encodeErr).NotTo(HaveOccurred())

			storageDiff := filters.StorageDiff{
				Key:   []byte{0, 9, 8, 7, 6, 5, 4, 3, 2, 1},
				Value: storageValueRlp,
			}

			result, err := types.FromNewGethStateDiff(accountDiff, stateDiff, storageDiff)
			Expect(err).NotTo(HaveOccurred())

			expectedHashedAddress := crypto.Keccak256Hash(accountDiff.Key)
			Expect(result.HashedAddress).To(Equal(expectedHashedAddress))
			Expect(result.BlockHash).To(Equal(fakes.FakeHash))
			expectedBlockHeight := int(stateDiff.BlockNumber.Int64())
			Expect(result.BlockHeight).To(Equal(expectedBlockHeight))
			expectedStorageKey := common.BytesToHash(storageDiff.Key)
			Expect(result.StorageKey).To(Equal(expectedStorageKey))
			expectedStorageValue := common.BytesToHash(storageValueBytes)
			Expect(result.StorageValue).To(Equal(expectedStorageValue))
		})

		It("handles decoding large storage values from their RLP", func() {
			storageValueBytes := []byte{1, 2, 3, 4, 5, 0, 9, 8, 7, 6}
			storageValueRlp, encodeErr := rlp.EncodeToBytes(storageValueBytes)
			Expect(encodeErr).NotTo(HaveOccurred())

			storageDiff := filters.StorageDiff{
				Key:   []byte{0, 9, 8, 7, 6, 5, 4, 3, 2, 1},
				Value: storageValueRlp,
			}

			result, err := types.FromNewGethStateDiff(accountDiff, stateDiff, storageDiff)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.StorageValue).To(Equal(common.BytesToHash(storageValueBytes)))
		})

		It("returns an err if decoding the storage value Rlp fails", func() {
			_, err := types.FromNewGethStateDiff(accountDiff, stateDiff, test_data.StorageWithBadValue)
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError("rlp: input contains more than one value"))
		})
	})
})
