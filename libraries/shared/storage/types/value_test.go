package types_test

import (
	"github.com/makerdao/vulcanizedb/libraries/shared/storage/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Storage value metadata getter", func() {
	It("returns storage value metadata for a single storage variable", func() {
		metadataName := "fake_name"
		metadataKeys := map[types.Key]string{"key": "value"}
		metadataType := types.Uint256

		expectedMetadata := types.ValueMetadata{
			Name: metadataName,
			Keys: metadataKeys,
			Type: metadataType,
		}
		Expect(types.GetValueMetadata(metadataName, metadataKeys, metadataType)).To(Equal(expectedMetadata))
	})

	Describe("metadata for a packed storage slot", func() {
		It("returns metadata for multiple storage variables", func() {
			metadataName := "fake_name"
			metadataKeys := map[types.Key]string{"key": "value"}
			metadataType := types.PackedSlot
			metadataPackedNames := map[int]string{0: "name"}
			metadataPackedTypes := map[int]types.ValueType{0: types.Uint48}

			expectedMetadata := types.ValueMetadata{
				Name:        metadataName,
				Keys:        metadataKeys,
				Type:        metadataType,
				PackedTypes: metadataPackedTypes,
				PackedNames: metadataPackedNames,
			}
			Expect(types.GetValueMetadataForPackedSlot(metadataName, metadataKeys, metadataType, metadataPackedNames, metadataPackedTypes)).To(Equal(expectedMetadata))
		})

		It("panics if PackedTypes are nil when the type is PackedSlot", func() {
			metadataName := "fake_name"
			metadataKeys := map[types.Key]string{"key": "value"}
			metadataType := types.PackedSlot
			metadataPackedNames := map[int]string{0: "name"}

			getMetadata := func() {
				types.GetValueMetadataForPackedSlot(metadataName, metadataKeys, metadataType, metadataPackedNames, nil)
			}
			Expect(getMetadata).To(Panic())
		})

		It("panics if PackedNames are nil when the type is PackedSlot", func() {
			metadataName := "fake_name"
			metadataKeys := map[types.Key]string{"key": "value"}
			metadataType := types.PackedSlot
			metadataPackedTypes := map[int]types.ValueType{0: types.Uint48}

			getMetadata := func() {
				types.GetValueMetadataForPackedSlot(metadataName, metadataKeys, metadataType, nil, metadataPackedTypes)
			}
			Expect(getMetadata).To(Panic())
		})

		It("panics if valueType is not PackedSlot if PackedNames is populated", func() {
			metadataName := "fake_name"
			metadataKeys := map[types.Key]string{"key": "value"}
			metadataType := types.Uint48
			metadataPackedNames := map[int]string{0: "name"}
			metadataPackedTypes := map[int]types.ValueType{0: types.Uint48}

			getMetadata := func() {
				types.GetValueMetadataForPackedSlot(metadataName, metadataKeys, metadataType, metadataPackedNames, metadataPackedTypes)
			}
			Expect(getMetadata).To(Panic())
		})
	})
})
