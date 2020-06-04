package old_geth_patches

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage/fetcher"
	"github.com/makerdao/vulcanizedb/libraries/shared/test_data"
)

var (
	storageWithSmallValue = []fetcher.StorageDiffOldPatch{{
		Key:   test_data.StorageKey,
		Value: test_data.SmallStorageValueRlp,
	}}
	storageWithLargeValue = []fetcher.StorageDiffOldPatch{{
		Key:   test_data.StorageKey,
		Value: test_data.LargeStorageValueRlp,
	}}

	testAccountDiff1 = fetcher.AccountDiffOldPatch{
		Key:     test_data.ContractLeafKey.Bytes(),
		Value:   test_data.AccountValueBytes,
		Storage: storageWithSmallValue,
	}
	testAccountDiff2 = fetcher.AccountDiffOldPatch{
		Key:     test_data.AnotherContractLeafKey.Bytes(),
		Value:   test_data.AccountValueBytes,
		Storage: storageWithLargeValue,
	}
	testAccountDiff3 = fetcher.AccountDiffOldPatch{
		Key:     test_data.AnotherContractLeafKey.Bytes(),
		Value:   test_data.AccountValueBytes,
		Storage: storageWithSmallValue,
	}

	MockStateDiff = fetcher.StateDiffOldPatch{
		BlockNumber:     test_data.BlockNumber,
		BlockHash:       common.HexToHash(test_data.BlockHash),
		UpdatedAccounts: []fetcher.AccountDiffOldPatch{testAccountDiff1},
		CreatedAccounts: []fetcher.AccountDiffOldPatch{testAccountDiff2},
		DeletedAccounts: []fetcher.AccountDiffOldPatch{testAccountDiff3},
	}
	MockStateDiffBytes, _ = rlp.EncodeToBytes(MockStateDiff)
	MockStatediffPayload  = filters.Payload{
		StateDiffRlp: MockStateDiffBytes,
	}

	storageWithBadValue = fetcher.StorageDiffOldPatch{
		Key:   test_data.StorageKey,
		Value: []byte{0, 1, 2},
		// this storage value will fail to be decoded as an RLP with the following error message:
		// "rlp: input contains more than one value"
	}
	testAccountDiffWithBadStorageValue = fetcher.AccountDiffOldPatch{
		Key:     test_data.ContractLeafKey.Bytes(),
		Value:   test_data.AccountValueBytes,
		Storage: []fetcher.StorageDiffOldPatch{storageWithBadValue},
	}
	StateDiffWithBadStorageValue = fetcher.StateDiffOldPatch{
		BlockNumber:     test_data.BlockNumber,
		BlockHash:       common.HexToHash(test_data.BlockHash),
		CreatedAccounts: []fetcher.AccountDiffOldPatch{testAccountDiffWithBadStorageValue},
	}
)
