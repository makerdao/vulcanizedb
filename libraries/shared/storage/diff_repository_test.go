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
	"database/sql"
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage/types"
	"github.com/makerdao/vulcanizedb/libraries/shared/test_data"
	"github.com/makerdao/vulcanizedb/pkg/datastore/postgres"
	"github.com/makerdao/vulcanizedb/test_config"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

var _ = Describe("Storage diffs repository", func() {
	var (
		db              = test_config.NewTestDB(test_config.NewTestNode())
		repo            storage.DiffRepository
		fakeStorageDiff types.RawDiff
	)

	BeforeEach(func() {
		test_config.CleanTestDB(db)
		repo = storage.NewDiffRepository(db)
		fakeStorageDiff = createFakeRawDiff(rand.Int())
	})

	type dbStorageDiff struct {
		Created string
		Updated string
	}

	Describe("CreateStorageDiff", func() {
		It("adds a storage diff to the db, returning id", func() {
			id, createErr := repo.CreateStorageDiff(fakeStorageDiff)

			Expect(createErr).NotTo(HaveOccurred())
			Expect(id).NotTo(BeZero())
			var persisted types.PersistedDiff
			getErr := db.Get(&persisted, `SELECT id, address, block_hash, block_height, storage_key, storage_value, status FROM public.storage_diff`)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(persisted.ID).To(Equal(id))
			Expect(persisted.Address).To(Equal(fakeStorageDiff.Address))
			Expect(persisted.BlockHash).To(Equal(fakeStorageDiff.BlockHash))
			Expect(persisted.BlockHeight).To(Equal(fakeStorageDiff.BlockHeight))
			Expect(persisted.StorageKey).To(Equal(fakeStorageDiff.StorageKey))
			Expect(persisted.StorageValue).To(Equal(fakeStorageDiff.StorageValue))
			Expect(persisted.Status).To(Equal(storage.New))
		})

		It("does not duplicate storage diffs", func() {
			_, createErr := repo.CreateStorageDiff(fakeStorageDiff)
			Expect(createErr).NotTo(HaveOccurred())

			_, createTwoErr := repo.CreateStorageDiff(fakeStorageDiff)
			Expect(createTwoErr).To(HaveOccurred())
			Expect(createTwoErr).To(MatchError(sql.ErrNoRows))

			var count int
			getErr := db.Get(&count, `SELECT count(*) FROM public.storage_diff`)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(count).To(Equal(1))
		})

		It("indicates when a record was created or updated", func() {
			id, createErr := repo.CreateStorageDiff(fakeStorageDiff)
			Expect(createErr).NotTo(HaveOccurred())

			var storageDiffUpdatedRes dbStorageDiff
			initialStorageErr := db.Get(&storageDiffUpdatedRes, `SELECT created, updated FROM public.storage_diff`)
			Expect(initialStorageErr).NotTo(HaveOccurred())
			Expect(storageDiffUpdatedRes.Created).To(Equal(storageDiffUpdatedRes.Updated))

			_, updateErr := db.Exec(`UPDATE public.storage_diff SET block_hash = '{"new_block_hash"}' where id = $1`, id)
			Expect(updateErr).NotTo(HaveOccurred())
			updatedDiffErr := db.Get(&storageDiffUpdatedRes, `SELECT created, updated FROM public.storage_diff`)
			Expect(updatedDiffErr).NotTo(HaveOccurred())
			Expect(storageDiffUpdatedRes.Created).NotTo(Equal(storageDiffUpdatedRes.Updated))
		})
	})

	Describe("CreateBackFilledStorageValue", func() {
		It("creates a storage diff", func() {
			createErr := repo.CreateBackFilledStorageValue(fakeStorageDiff)

			Expect(createErr).NotTo(HaveOccurred())
			var persisted types.PersistedDiff
			getErr := db.Get(&persisted, `SELECT address, block_hash, block_height, storage_key, storage_value, status FROM public.storage_diff`)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(persisted.Address).To(Equal(fakeStorageDiff.Address))
			Expect(persisted.BlockHash).To(Equal(fakeStorageDiff.BlockHash))
			Expect(persisted.BlockHeight).To(Equal(fakeStorageDiff.BlockHeight))
			Expect(persisted.StorageKey).To(Equal(fakeStorageDiff.StorageKey))
			Expect(persisted.StorageValue).To(Equal(fakeStorageDiff.StorageValue))
			Expect(persisted.Status).To(Equal("new"))
		})

		It("marks diff as back-filled", func() {
			createErr := repo.CreateBackFilledStorageValue(fakeStorageDiff)

			Expect(createErr).NotTo(HaveOccurred())
			var fromBackfill bool
			checkedErr := db.Get(&fromBackfill, `SELECT from_backfill FROM public.storage_diff`)
			Expect(checkedErr).NotTo(HaveOccurred())
			Expect(fromBackfill).To(BeTrue())
		})

		It("does not duplicate storage values in the same block", func() {
			_, createErr := repo.CreateStorageDiff(fakeStorageDiff)
			Expect(createErr).NotTo(HaveOccurred())

			createTwoErr := repo.CreateBackFilledStorageValue(fakeStorageDiff)
			Expect(createTwoErr).NotTo(HaveOccurred())

			var count int
			getErr := db.Get(&count, `SELECT count(*) FROM public.storage_diff`)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(count).To(Equal(1))
		})

		It("does not duplicate storage values across subsequent blocks", func() {
			_, createErr := repo.CreateStorageDiff(fakeStorageDiff)
			Expect(createErr).NotTo(HaveOccurred())

			duplicateDiff := fakeStorageDiff
			duplicateDiff.BlockHeight = fakeStorageDiff.BlockHeight + 1
			createTwoErr := repo.CreateBackFilledStorageValue(duplicateDiff)
			Expect(createTwoErr).NotTo(HaveOccurred())

			var count int
			getErr := db.Get(&count, `SELECT count(*) FROM public.storage_diff`)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(count).To(Equal(1))
		})

		It("does duplicate storage value if same value only exists at a later block", func() {
			_, createErr := repo.CreateStorageDiff(fakeStorageDiff)
			Expect(createErr).NotTo(HaveOccurred())

			duplicateDiff := fakeStorageDiff
			duplicateDiff.BlockHeight = fakeStorageDiff.BlockHeight - 1
			createTwoErr := repo.CreateBackFilledStorageValue(duplicateDiff)
			Expect(createTwoErr).NotTo(HaveOccurred())

			var count int
			getErr := db.Get(&count, `SELECT count(*) FROM public.storage_diff`)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(count).To(Equal(2))
		})

		It("inserts zero-valued storage if there's a previous diff", func() {
			_, createOneErr := repo.CreateStorageDiff(fakeStorageDiff)
			Expect(createOneErr).NotTo(HaveOccurred())
			emptyStorageValue := fakeStorageDiff
			emptyStorageValue.StorageValue = common.HexToHash("0x0")

			createTwoErr := repo.CreateBackFilledStorageValue(emptyStorageValue)

			Expect(createTwoErr).NotTo(HaveOccurred())
			var count int
			getErr := db.Get(&count, `SELECT count(*) FROM public.storage_diff`)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(count).To(Equal(2))
		})

		It("does not insert zero-valued storage if there's no previous diff", func() {
			emptyStorageValue := fakeStorageDiff
			emptyStorageValue.StorageValue = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")

			createErr := repo.CreateBackFilledStorageValue(emptyStorageValue)

			Expect(createErr).NotTo(HaveOccurred())
			var count int
			getErr := db.Get(&count, `SELECT count(*) FROM public.storage_diff`)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(count).To(BeZero())
		})

		It("does not insert zero-valued storage derived from bytes if there's no previous diff", func() {
			emptyStorageValue := fakeStorageDiff
			emptyStorageValue.StorageValue = common.BytesToHash([]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})

			createErr := repo.CreateBackFilledStorageValue(emptyStorageValue)

			Expect(createErr).NotTo(HaveOccurred())
			var count int
			getErr := db.Get(&count, `SELECT count(*) FROM public.storage_diff`)
			Expect(getErr).NotTo(HaveOccurred())

		})
	})

	Describe("GetDiffs", func() {

		SkipStatus := func(status string) TableEntry {
			return Entry("skip", status, false)
		}

		KeepStatus := func(status string) TableEntry {
			return Entry("keep", status, true)
		}

		DescribeTable("GetNewDiffs",
			func(status string, present bool) {
				fakePersistedDiff := createFakePersistedDiff(fakeStorageDiff, status, db.NodeID)
				insertTestDiff(fakePersistedDiff, db)

				diffs, err := repo.GetNewDiffs(0, 1)
				Expect(err).NotTo(HaveOccurred())
				if present {
					Expect(diffs).To(ConsistOf(fakePersistedDiff))
				} else {
					Expect(diffs).To(BeEmpty())
				}
			},
			KeepStatus(storage.New),
			SkipStatus(storage.Pending),
			SkipStatus(storage.Noncanonical),
			SkipStatus(storage.Transformed),
			SkipStatus(storage.Unrecognized),
			SkipStatus(storage.Unwatched),
		)

		DescribeTable("GetPendingDiffs",
			func(status string, present bool) {
				fakePersistedDiff := createFakePersistedDiff(fakeStorageDiff, status, db.NodeID)
				insertTestDiff(fakePersistedDiff, db)

				diffs, err := repo.GetPendingDiffs(0, 1)
				Expect(err).NotTo(HaveOccurred())
				if present {
					Expect(diffs).To(ConsistOf(fakePersistedDiff))
				} else {
					Expect(diffs).To(BeEmpty())
				}
			},
			KeepStatus(storage.Pending),
			SkipStatus(storage.New),
			SkipStatus(storage.Noncanonical),
			SkipStatus(storage.Transformed),
			SkipStatus(storage.Unrecognized),
			SkipStatus(storage.Unwatched),
		)

		DescribeTable("GetUnrecognizedDiffs",
			func(status string, present bool) {
				fakePersistedDiff := createFakePersistedDiff(fakeStorageDiff, status, db.NodeID)
				insertTestDiff(fakePersistedDiff, db)

				diffs, err := repo.GetUnrecognizedDiffs(0, 1)
				Expect(err).NotTo(HaveOccurred())
				if present {
					Expect(diffs).To(ConsistOf(fakePersistedDiff))
				} else {
					Expect(diffs).To(BeEmpty())
				}
			},
			KeepStatus(storage.Unrecognized),
			SkipStatus(storage.New),
			SkipStatus(storage.Pending),
			SkipStatus(storage.Noncanonical),
			SkipStatus(storage.Transformed),
			SkipStatus(storage.Unwatched),
		)

		// Surely there is a better way for me to setup these calls
		type queryFunc func(int, int) ([]types.PersistedDiff, error)

		getNewDiffs := func(minID, limit int) ([]types.PersistedDiff, error) {
			return repo.GetNewDiffs(minID, limit)
		}

		getPendingDiffs := func(minID, limit int) ([]types.PersistedDiff, error) {
			return repo.GetPendingDiffs(minID, limit)
		}

		getUnrecognizedDiffs := func(minID, limit int) ([]types.PersistedDiff, error) {
			return repo.GetUnrecognizedDiffs(minID, limit)
		}

		DescribeTable("seeking diffs with greater ID",
			func(query queryFunc, status string) {
				blockZero := rand.Int()
				for i := 0; i < 2; i++ {
					rawDiff := createFakeRawDiff(blockZero + i)
					persistedDiff := createFakePersistedDiff(rawDiff, status, db.NodeID)
					insertTestDiff(persistedDiff, db)
				}

				minID := 0
				limit := 1
				diffsOne, errOne := query(minID, limit)
				Expect(errOne).NotTo(HaveOccurred())
				Expect(len(diffsOne)).To(Equal(1))

				nextID := int(diffsOne[0].ID)
				diffsTwo, errTwo := query(nextID, limit)
				Expect(errTwo).NotTo(HaveOccurred())
				Expect(len(diffsTwo)).To(Equal(1))
				Expect(int(diffsTwo[0].ID) > nextID).To(BeTrue())
			},
			Entry("new diffs", getNewDiffs, storage.New),
			Entry("pending diffs", getPendingDiffs, storage.Pending),
			Entry("unrecognized diffs", getUnrecognizedDiffs, storage.Unrecognized),
		)
	})

	Describe("Changing the diff status", func() {
		var fakePersistedDiff types.PersistedDiff
		BeforeEach(func() {
			fakePersistedDiff = createFakePersistedDiff(fakeStorageDiff, storage.New, db.NodeID)
			insertTestDiff(fakePersistedDiff, db)
		})

		It("marks a diff as transformed", func() {
			err := repo.MarkTransformed(fakePersistedDiff.ID)

			Expect(err).NotTo(HaveOccurred())
			var status string
			getStatusErr := db.Get(&status, `SELECT status FROM public.storage_diff WHERE id = $1`, fakePersistedDiff.ID)
			Expect(getStatusErr).NotTo(HaveOccurred())
			Expect(status).To(Equal(storage.Transformed))
		})

		It("marks a diff as unrecognized", func() {
			err := repo.MarkUnrecognized(fakePersistedDiff.ID)

			Expect(err).NotTo(HaveOccurred())
			var status string
			getStatusErr := db.Get(&status, `SELECT status FROM public.storage_diff WHERE id = $1`, fakePersistedDiff.ID)
			Expect(getStatusErr).NotTo(HaveOccurred())
			Expect(status).To(Equal(storage.Unrecognized))
		})

		It("marks a diff as noncanonical", func() {
			err := repo.MarkNoncanonical(fakePersistedDiff.ID)

			Expect(err).NotTo(HaveOccurred())
			var status string
			getStatusErr := db.Get(&status, `SELECT status FROM public.storage_diff WHERE id = $1`, fakePersistedDiff.ID)
			Expect(getStatusErr).NotTo(HaveOccurred())
			Expect(status).To(Equal(storage.Noncanonical))
		})

		It("marks a diff as unwatched", func() {
			err := repo.MarkUnwatched(fakePersistedDiff.ID)

			Expect(err).NotTo(HaveOccurred())
			var status string
			getStatusErr := db.Get(&status, `SELECT status FROM public.storage_diff WHERE id = $1`, fakePersistedDiff.ID)
			Expect(getStatusErr).NotTo(HaveOccurred())
			Expect(status).To(Equal(storage.Unwatched))
		})

		It("marks a diff as pending", func() {
			err := repo.MarkPending(fakePersistedDiff.ID)
			Expect(err).NotTo(HaveOccurred())

			var status string
			getStatusErr := db.Get(&status, `SELECT status FROM public.storage_diff WHERE id = $1`, fakePersistedDiff.ID)

			Expect(getStatusErr).NotTo(HaveOccurred())
			Expect(status).To(Equal(storage.Pending))
		})
	})

	Describe("Marking non-canonical diffs as new", func() {
		It("does nothing if no diffs in block are marked non-canonical", func() {
			blockHeight := rand.Int()
			numDiffs := 3
			for i := 0; i < numDiffs; i++ {
				rawDiff := createFakeRawDiff(blockHeight)
				persistedDiff := createFakePersistedDiff(rawDiff, storage.Transformed, db.NodeID)
				insertTestDiff(persistedDiff, db)
			}

			err := repo.MarkNoncanonicalDiffsAsNew(int64(blockHeight))

			Expect(err).NotTo(HaveOccurred())
			var count int
			getErr := db.Get(&count, `SELECT COUNT(*) FROM public.storage_diff WHERE status = $1`, storage.Transformed)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(count).To(Equal(numDiffs))
		})

		It("updates diffs marked as non-canonical to 'new'", func() {
			blockHeight := rand.Int()
			rawDiff := createFakeRawDiff(blockHeight)
			persistedDiff := createFakePersistedDiff(rawDiff, storage.Noncanonical, db.NodeID)
			insertTestDiff(persistedDiff, db)

			err := repo.MarkNoncanonicalDiffsAsNew(int64(blockHeight))

			Expect(err).NotTo(HaveOccurred())
			var status string
			getErr := db.Get(&status, `SELECT status FROM public.storage_diff`)
			Expect(getErr).NotTo(HaveOccurred())
			Expect(status).To(Equal(storage.New))
		})
	})

	Describe("GetFirstDiffIDForBlockHeight", func() {
		It("sends first diff for a given block height", func() {
			blockHeight := fakeStorageDiff.BlockHeight
			fakeStorageDiff2 := createFakeRawDiff(blockHeight)

			id1, create1Err := repo.CreateStorageDiff(fakeStorageDiff)
			Expect(create1Err).NotTo(HaveOccurred())
			_, create2Err := repo.CreateStorageDiff(fakeStorageDiff2)
			Expect(create2Err).NotTo(HaveOccurred())

			diffID, diffErr := repo.GetFirstDiffIDForBlockHeight(int64(blockHeight))
			Expect(diffErr).NotTo(HaveOccurred())
			Expect(diffID).To(Equal(id1))
		})

		It("sends a diff for the next block height if one doesn't exist for the block passed in", func() {
			blockHeight := fakeStorageDiff.BlockHeight
			fakeStorageDiff2 := createFakeRawDiff(blockHeight)

			id1, create1Err := repo.CreateStorageDiff(fakeStorageDiff)
			Expect(create1Err).NotTo(HaveOccurred())
			_, create2Err := repo.CreateStorageDiff(fakeStorageDiff2)
			Expect(create2Err).NotTo(HaveOccurred())

			blockBeforeDiffBlockHeight := int64(blockHeight - 1)
			diffID, diffErr := repo.GetFirstDiffIDForBlockHeight(blockBeforeDiffBlockHeight)
			Expect(diffErr).NotTo(HaveOccurred())
			Expect(diffID).To(Equal(id1))
		})

		It("won't fail if all of the diffs within the id range are already checked", func() {
			fakePersistedDiff := createFakePersistedDiff(fakeStorageDiff, storage.Transformed, db.NodeID)
			insertTestDiff(fakePersistedDiff, db)

			var insertedDiffID int64
			getInsertedDiffIDErr := db.Get(&insertedDiffID, `SELECT id FROM storage_diff LIMIT 1`)
			Expect(getInsertedDiffIDErr).NotTo(HaveOccurred())

			blockBeforeDiffBlockHeight := int64(fakeStorageDiff.BlockHeight - 1)
			diffID, diffErr := repo.GetFirstDiffIDForBlockHeight(blockBeforeDiffBlockHeight)
			Expect(diffErr).NotTo(HaveOccurred())
			Expect(diffID).To(Equal(insertedDiffID))
		})

		It("returns an error if getting the diff fails", func() {
			_, diffErr := repo.GetFirstDiffIDForBlockHeight(0)
			Expect(diffErr).To(HaveOccurred())
			Expect(diffErr).To(MatchError(sql.ErrNoRows))
		})
	})
})

func createFakeRawDiff(blockHeight int) types.RawDiff {
	return types.RawDiff{
		Address:      test_data.FakeAddress(),
		BlockHash:    test_data.FakeHash(),
		BlockHeight:  blockHeight,
		StorageKey:   test_data.FakeHash(),
		StorageValue: test_data.FakeHash(),
	}
}

func createFakePersistedDiff(rawDiff types.RawDiff, status string, nodeID int64) types.PersistedDiff {
	return types.PersistedDiff{
		RawDiff:   rawDiff,
		ID:        rand.Int63(),
		Status:    status,
		EthNodeID: nodeID,
	}
}

func insertTestDiff(persistedDiff types.PersistedDiff, db *postgres.DB) {
	rawDiff := persistedDiff.RawDiff
	_, insertErr := db.Exec(`INSERT INTO public.storage_diff (id, block_height, block_hash,
				address, storage_key, storage_value, status, eth_node_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		persistedDiff.ID, rawDiff.BlockHeight, rawDiff.BlockHash.Bytes(),
		rawDiff.Address.Bytes(), rawDiff.StorageKey.Bytes(), rawDiff.StorageValue.Bytes(),
		persistedDiff.Status, persistedDiff.EthNodeID)
	Expect(insertErr).NotTo(HaveOccurred())
}
