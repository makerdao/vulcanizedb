// Copyright 2018 Vulcanize
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test_data

import (
	"math/big"
	"math/rand"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	BlockNumber             = big.NewInt(rand.Int63())
	BlockHash               = "0xfa40fbe2d98d98b3363a778d52f2bcd29d6790b9b3f3cab2b167fd12d3550f73"
	CodeHash                = common.Hex2Bytes("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470")
	NewNonceValue           = rand.Uint64()
	NewBalanceValue         = rand.Int63()
	ContractRoot            = common.HexToHash("0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")
	StoragePath             = common.HexToHash("0xc5d2460186f7233c927e7db2dcc703c0e500b653ca82273b7bfad8045d85a470").Bytes()
	StorageKey              = common.HexToHash("0000000000000000000000000000000000000000000000000000000000000001").Bytes()
	SmallStorageValue       = common.Hex2Bytes("03")
	SmallStorageValueRlp, _ = rlp.EncodeToBytes(SmallStorageValue)
	storageWithSmallValue   = []filters.StorageDiff{{
		Key:   StorageKey,
		Value: SmallStorageValueRlp,
	}}
	LargeStorageValue            = common.Hex2Bytes("00191b53778c567b14b50ba0000")
	LargeStorageValueRlp, rlpErr = rlp.EncodeToBytes(LargeStorageValue)
	storageWithLargeValue        = []filters.StorageDiff{{
		Key:   StorageKey,
		Value: LargeStorageValueRlp,
	}}
	EmptyStorage        = make([]filters.StorageDiff, 0)
	StorageWithBadValue = filters.StorageDiff{
		Key:   StorageKey,
		Value: []byte{0, 1, 2},
		// this storage value will fail to be decoded as an RLP with the following error message:
		// "input contains more than one value"
	}
	contractAddress        = common.HexToAddress("0xaE9BEa628c4Ce503DcFD7E305CaB4e29E7476592")
	ContractLeafKey        = crypto.Keccak256Hash(contractAddress[:])
	anotherContractAddress = common.HexToAddress("0xaE9BEa628c4Ce503DcFD7E305CaB4e29E7476593")
	AnotherContractLeafKey = crypto.Keccak256Hash(anotherContractAddress[:])

	testAccount = state.Account{
		Nonce:    NewNonceValue,
		Balance:  big.NewInt(NewBalanceValue),
		Root:     ContractRoot,
		CodeHash: CodeHash,
	}
	AccountValueBytes, _    = rlp.EncodeToBytes(testAccount)
	testAccountDiff1 = filters.AccountDiff{
		Key:     ContractLeafKey.Bytes(),
		Value:   AccountValueBytes,
		Storage: storageWithSmallValue,
	}
	testAccountDiff2 = filters.AccountDiff{
		Key:     AnotherContractLeafKey.Bytes(),
		Value:   AccountValueBytes,
		Storage: storageWithLargeValue,
	}
	testAccountDiff3 = filters.AccountDiff{
		Key:     AnotherContractLeafKey.Bytes(),
		Value:   AccountValueBytes,
		Storage: storageWithSmallValue,
	}
	CreatedAccountDiffs = []filters.AccountDiff{testAccountDiff1}
	UpdatedAccountDiffs = []filters.AccountDiff{testAccountDiff1, testAccountDiff2, testAccountDiff3}
	DeletedAccountDiffs = []filters.AccountDiff{testAccountDiff3}

	MockStateDiff = filters.StateDiff{
		BlockNumber:     BlockNumber,
		BlockHash:       common.HexToHash(BlockHash),
		UpdatedAccounts: UpdatedAccountDiffs,
	}
	MockStateDiffBytes, _ = rlp.EncodeToBytes(MockStateDiff)

	mockTransaction1 = types.NewTransaction(0, common.HexToAddress("0x0"), big.NewInt(1000), 50, big.NewInt(100), nil)
	mockTransaction2 = types.NewTransaction(1, common.HexToAddress("0x1"), big.NewInt(2000), 100, big.NewInt(200), nil)
	MockTransactions = types.Transactions{mockTransaction1, mockTransaction2}

	mockReceipt1 = types.NewReceipt(common.HexToHash("0x0").Bytes(), false, 50)
	mockReceipt2 = types.NewReceipt(common.HexToHash("0x1").Bytes(), false, 100)
	MockReceipts = types.Receipts{mockReceipt1, mockReceipt2}

	MockHeader = types.Header{
		Time:        0,
		Number:      BlockNumber,
		Root:        common.HexToHash("0x0"),
		TxHash:      common.HexToHash("0x0"),
		ReceiptHash: common.HexToHash("0x0"),
	}
	MockBlock       = types.NewBlock(&MockHeader, MockTransactions, nil, MockReceipts)
	MockBlockRlp, _ = rlp.EncodeToBytes(MockBlock)

	MockStatediffPayload = filters.Payload{
		StateDiffRlp: MockStateDiffBytes,
	}
)
