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

package storage_test

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/makerdao/vulcanizedb/libraries/shared/factories/storage"
	"github.com/makerdao/vulcanizedb/libraries/shared/mocks"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage/types"
	"github.com/makerdao/vulcanizedb/libraries/shared/test_data"
	"github.com/makerdao/vulcanizedb/pkg/fakes"
	"github.com/makerdao/vulcanizedb/test_config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Storage keys lookup", func() {
	var (
		fakeMetadata = types.GetValueMetadata("name", map[types.Key]string{}, types.Uint256)
		lookup       storage.KeysLookup
		loader       *mocks.MockStorageKeysLoader
	)

	BeforeEach(func() {
		loader = &mocks.MockStorageKeysLoader{}
		lookup = storage.NewKeysLookup(loader)
	})

	Describe("Lookup", func() {
		Describe("when key not found", func() {
			It("refreshes keys", func() {
				fakeKey := test_data.FakeHash()
				loader.StorageKeyMappings = map[common.Hash]types.ValueMetadata{fakeKey: fakeMetadata}
				_, err := lookup.Lookup(fakeKey)

				Expect(err).NotTo(HaveOccurred())
				Expect(loader.LoadMappingsCallCount).To(Equal(1))
			})

			It("returns error if refreshing keys fails", func() {
				loader.LoadMappingsError = fakes.FakeError

				_, err := lookup.Lookup(test_data.FakeHash())

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(fakes.FakeError))
			})
		})

		Describe("when key found", func() {
			var fakeKey common.Hash

			BeforeEach(func() {
				fakeKey = test_data.FakeHash()
				loader.StorageKeyMappings = map[common.Hash]types.ValueMetadata{fakeKey: fakeMetadata}
				_, err := lookup.Lookup(fakeKey)
				Expect(err).NotTo(HaveOccurred())
				Expect(loader.LoadMappingsCallCount).To(Equal(1))
			})

			It("does not refresh keys", func() {
				_, err := lookup.Lookup(fakeKey)

				Expect(err).NotTo(HaveOccurred())
				Expect(loader.LoadMappingsCallCount).To(Equal(1))
			})
		})

		It("returns metadata for loaded static key", func() {
			fakeKey := test_data.FakeHash()
			loader.StorageKeyMappings = map[common.Hash]types.ValueMetadata{fakeKey: fakeMetadata}

			metadata, err := lookup.Lookup(fakeKey)

			Expect(err).NotTo(HaveOccurred())
			Expect(metadata).To(Equal(fakeMetadata))
		})

		It("returns key not found error if key not found", func() {
			fakeKey := test_data.FakeHash()
			_, err := lookup.Lookup(fakeKey)

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(types.ErrKeyNotFound))
		})
	})

	Describe("SetDB", func() {
		It("sets the db on the loader", func() {
			lookup.SetDB(test_config.NewTestDB(test_config.NewTestNode()))

			Expect(loader.SetDBCalled).To(BeTrue())
		})
	})

	Describe("GetKeys", func() {
		var (
			mappings = make(map[common.Hash]types.ValueMetadata)
			keyOne   = test_data.FakeHash()
			keyTwo   = test_data.FakeHash()
		)

		BeforeEach(func() {
			mappings[keyOne] = fakeMetadata
			mappings[keyTwo] = fakeMetadata
		})

		It("gets the keys that were loaded with the loader", func() {
			loader.StorageKeyMappings = mappings

			keys, err := lookup.GetKeys()

			Expect(err).NotTo(HaveOccurred())
			Expect(keys).To(ContainElement(keyOne))
			Expect(keys).To(ContainElement(keyTwo))
		})

		It("returns an error if GetKeys fails", func() {
			loader.LoadMappingsError = fakes.FakeError

			_, err := lookup.GetKeys()

			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(fakes.FakeError))
		})
	})
})
