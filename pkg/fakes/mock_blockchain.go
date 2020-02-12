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

package fakes

import (
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/makerdao/vulcanizedb/pkg/core"
	. "github.com/onsi/gomega"
)

type MockBlockChain struct {
	fetchContractDataErr               error
	fetchContractDataPassedAbi         string
	fetchContractDataPassedAddress     string
	fetchContractDataPassedMethod      string
	fetchContractDataPassedMethodArgs  []interface{}
	fetchContractDataPassedResult      interface{}
	fetchContractDataPassedBlockNumber int64
	GetTransactionsCalled              bool
	GetTransactionsError               error
	GetTransactionsPassedHashes        []common.Hash
	GetStorageAtPassedAccounts         []common.Address
	GetStorageAtPassedKeys             []common.Hash
	GetStorageAtPassedBlockNumber      *big.Int
	GetStorageAtError                  error
	storageValuesToReturn              map[common.Address][]byte
	logQuery                           ethereum.FilterQuery
	logQueryErr                        error
	logQueryReturnLogs                 []types.Log
	lastBlock                          *big.Int
	node                               core.Node
	Transactions                       []core.TransactionModel
}

func NewMockBlockChain() *MockBlockChain {
	return &MockBlockChain{
		node:                  core.Node{GenesisBlock: "GENESIS", NetworkID: 1, ID: "x123", ClientName: "Geth"},
		storageValuesToReturn: make(map[common.Address][]byte),
	}
}

func (blockChain *MockBlockChain) SetFetchContractDataErr(err error) {
	blockChain.fetchContractDataErr = err
}

func (blockChain *MockBlockChain) SetLastBlock(blockNumber *big.Int) {
	blockChain.lastBlock = blockNumber
}

func (blockChain *MockBlockChain) SetGetEthLogsWithCustomQueryErr(err error) {
	blockChain.logQueryErr = err
}

func (blockChain *MockBlockChain) SetGetEthLogsWithCustomQueryReturnLogs(logs []types.Log) {
	blockChain.logQueryReturnLogs = logs
}

func (blockChain *MockBlockChain) FetchContractData(abiJSON string, address string, method string, methodArgs []interface{}, result interface{}, blockNumber int64) error {
	blockChain.fetchContractDataPassedAbi = abiJSON
	blockChain.fetchContractDataPassedAddress = address
	blockChain.fetchContractDataPassedMethod = method
	blockChain.fetchContractDataPassedMethodArgs = methodArgs
	blockChain.fetchContractDataPassedResult = result
	blockChain.fetchContractDataPassedBlockNumber = blockNumber
	return blockChain.fetchContractDataErr
}

func (blockChain *MockBlockChain) GetEthLogsWithCustomQuery(query ethereum.FilterQuery) ([]types.Log, error) {
	blockChain.logQuery = query
	return blockChain.logQueryReturnLogs, blockChain.logQueryErr
}

func (blockChain *MockBlockChain) GetHeaderByNumber(blockNumber int64) (core.Header, error) {
	return core.Header{BlockNumber: blockNumber}, nil
}

func (blockChain *MockBlockChain) GetHeadersByNumbers(blockNumbers []int64) ([]core.Header, error) {
	var headers []core.Header
	for _, blockNumber := range blockNumbers {
		var header = core.Header{BlockNumber: blockNumber}
		headers = append(headers, header)
	}
	return headers, nil
}

func (blockChain *MockBlockChain) GetTransactions(transactionHashes []common.Hash) ([]core.TransactionModel, error) {
	blockChain.GetTransactionsCalled = true
	blockChain.GetTransactionsPassedHashes = transactionHashes
	return blockChain.Transactions, blockChain.GetTransactionsError
}

func (blockChain *MockBlockChain) CallContract(contractHash string, input []byte, blockNumber *big.Int) ([]byte, error) {
	return []byte{}, nil
}

func (blockChain *MockBlockChain) LastBlock() (*big.Int, error) {
	return blockChain.lastBlock, nil
}

func (blockChain *MockBlockChain) GetStorageAt(account common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	blockChain.GetStorageAtPassedAccounts = append(blockChain.GetStorageAtPassedAccounts, account)
	blockChain.GetStorageAtPassedKeys = append(blockChain.GetStorageAtPassedKeys, key)
	blockChain.GetStorageAtPassedBlockNumber = blockNumber

	storageToReturn := blockChain.storageValuesToReturn[account]
	return storageToReturn, blockChain.GetStorageAtError
}

func (blockChain *MockBlockChain) SetGetStorageAtError(err error) {
	blockChain.GetStorageAtError = err
}

func (blockChain *MockBlockChain) SetStorageValuesToReturn(address common.Address, value []byte) {
	blockChain.storageValuesToReturn[address] = value
}

func (blockChain *MockBlockChain) Node() core.Node {
	return blockChain.node
}

func (blockChain *MockBlockChain) AssertFetchContractDataCalledWith(abiJSON string, address string, method string, methodArgs []interface{}, result interface{}, blockNumber int64) {
	Expect(blockChain.fetchContractDataPassedAbi).To(Equal(abiJSON))
	Expect(blockChain.fetchContractDataPassedAddress).To(Equal(address))
	Expect(blockChain.fetchContractDataPassedMethod).To(Equal(method))
	if methodArgs != nil {
		Expect(blockChain.fetchContractDataPassedMethodArgs).To(Equal(methodArgs))
	}
	Expect(blockChain.fetchContractDataPassedResult).To(BeAssignableToTypeOf(result))
	Expect(blockChain.fetchContractDataPassedBlockNumber).To(Equal(blockNumber))
}

func (blockChain *MockBlockChain) AssertGetEthLogsWithCustomQueryCalledWith(query ethereum.FilterQuery) {
	Expect(blockChain.logQuery).To(Equal(query))
}
