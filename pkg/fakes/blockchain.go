package fakes

import (
	"github.com/8thlight/vulcanizedb/pkg/core"
	"sort"
)

type Blockchain struct {
	blocks             map[int64]core.Block
	contractAttributes map[string]map[string]string
	blocksChannel      chan core.Block
	WasToldToStop      bool
}

func (blockchain *Blockchain) GetContractAttributes(contractHash string) (core.ContractAttributes, error) {
	var contractAttributes core.ContractAttributes
	attributes, ok := blockchain.contractAttributes[contractHash]
	if ok {
		for key, _ := range attributes {
			contractAttributes = append(contractAttributes, core.ContractAttribute{Name: key, Type: "string"})
		}
	}
	sort.Sort(contractAttributes)
	return contractAttributes, nil
}

func (blockchain *Blockchain) GetContractStateAttribute(contractHash string, attributeName string) (*string, error) {
	result := new(string)
	*result = blockchain.contractAttributes[contractHash][attributeName]
	return result, nil
}

func NewBlockchain() *Blockchain {
	return &Blockchain{
		blocks:             make(map[int64]core.Block),
		contractAttributes: make(map[string]map[string]string),
	}
}

func NewBlockchainWithBlocks(blocks []core.Block) *Blockchain {
	blockNumberToBlocks := make(map[int64]core.Block)
	for _, block := range blocks {
		blockNumberToBlocks[block.Number] = block
	}
	return &Blockchain{
		blocks: blockNumberToBlocks,
	}
}

func (blockchain *Blockchain) GetBlockByNumber(blockNumber int64) core.Block {
	return blockchain.blocks[blockNumber]
}

func (blockchain *Blockchain) SubscribeToBlocks(outputBlocks chan core.Block) {
	blockchain.blocksChannel = outputBlocks
}

func (blockchain *Blockchain) AddBlock(block core.Block) {
	blockchain.blocks[block.Number] = block
	blockchain.blocksChannel <- block
}

func (*Blockchain) StartListening() {}

func (blockchain *Blockchain) StopListening() {
	blockchain.WasToldToStop = true
}

func (blockchain *Blockchain) SetContractStateAttribute(contractHash string, attributeName string, attributeValue string) {
	contractStateAttributes := blockchain.contractAttributes[contractHash]
	if contractStateAttributes == nil {
		blockchain.contractAttributes[contractHash] = make(map[string]string)
	}
	blockchain.contractAttributes[contractHash][attributeName] = attributeValue
}