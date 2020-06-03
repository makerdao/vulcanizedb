package old_geth_patches

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/makerdao/vulcanizedb/libraries/shared/test_data"
)
var (
	storageWithSmallValue = []StorageDiffOldPatch{{
		Key:   test_data.StorageKey,
		Value: test_data.SmallStorageValueRlp,
	}}
	storageWithLargeValue = []StorageDiffOldPatch{{
		Key:   test_data.StorageKey,
		Value: test_data.LargeStorageValueRlp,
	}}

	testAccountDiff1 = AccountDiffOldPatch{
		Key:     test_data.ContractLeafKey.Bytes(),
		Value:   test_data.AccountValueBytes,
		Storage: storageWithSmallValue,
	}
	testAccountDiff2 = AccountDiffOldPatch{
		Key:     test_data.AnotherContractLeafKey.Bytes(),
		Value:   test_data.AccountValueBytes,
		Storage: storageWithLargeValue,
	}
	testAccountDiff3 = AccountDiffOldPatch{
		Key:     test_data.AnotherContractLeafKey.Bytes(),
		Value:   test_data.AccountValueBytes,
		Storage: storageWithSmallValue,
	}

	MockStateDiff = StateDiffOldPatch{
		BlockNumber:     test_data.BlockNumber,
		BlockHash:       common.HexToHash(test_data.BlockHash),
		UpdatedAccounts: []AccountDiffOldPatch{testAccountDiff1},
		CreatedAccounts: []AccountDiffOldPatch{testAccountDiff2},
		DeletedAccounts: []AccountDiffOldPatch{testAccountDiff3},
	}
	MockStateDiffBytes, _ = rlp.EncodeToBytes(MockStateDiff)
	MockStatediffPayload  = filters.Payload{
		StateDiffRlp: MockStateDiffBytes,
	}

	storageWithBadValue = StorageDiffOldPatch{
		Key:   test_data.StorageKey,
		Value: []byte{0, 1, 2},
		// this storage value will fail to be decoded as an RLP with the following error message:
		// "rlp: input contains more than one value"
	}
	testAccountDiffWithBadStorageValue = AccountDiffOldPatch{
		Key:     test_data.ContractLeafKey.Bytes(),
		Value:   test_data.AccountValueBytes,
		Storage: []StorageDiffOldPatch{storageWithBadValue},
	}
	StateDiffWithBadStorageValue = StateDiffOldPatch{
		BlockNumber:     test_data.BlockNumber,
		BlockHash:       common.HexToHash(test_data.BlockHash),
		CreatedAccounts: []AccountDiffOldPatch{testAccountDiffWithBadStorageValue},
	}
)



//Types for old patches

//This is a bit confusing... VDB only knows about filters.StateDiff, so it passes that in as the channel type to the statediff subscription
// however, the OLD geth version's payload types included BlockRLP, ReceiptsRLP, encoded and err fields.
//For some reason, this still works - maybe because we were never including the BlockRLP or ReceiptsRLP?

type StateDiffOldPatch struct {
	BlockNumber     *big.Int              `json:"blockNumber"     gencodec:"required"`
	BlockHash       common.Hash           `json:"blockHash"       gencodec:"required"`
	CreatedAccounts []AccountDiffOldPatch `json:"createdAccounts"`
	DeletedAccounts []AccountDiffOldPatch `json:"deletedAccounts"`
	UpdatedAccounts []AccountDiffOldPatch `json:"updatedAccounts" gencodec:"required"`

	encoded []byte
	err     error
}

type AccountDiffOldPatch struct {
	Leaf    bool                  `json:"leaf"`
	Key     []byte                `json:"key"         gencodec:"required"`
	Value   []byte                `json:"value"       gencodec:"required"`
	Proof   [][]byte              `json:"proof"`
	Path    []byte                `json:"path"`
	Storage []StorageDiffOldPatch `json:"storage"     gencodec:"required"`
}

type StorageDiffOldPatch struct {
	Leaf  bool     `json:"leaf"`
	Key   []byte   `json:"key"         gencodec:"required"`
	Value []byte   `json:"value"       gencodec:"required"`
	Proof [][]byte `json:"proof"`
	Path  []byte   `json:"path"`
}
