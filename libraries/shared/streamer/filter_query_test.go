package streamer_test

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/makerdao/vulcanizedb/libraries/shared/streamer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/spf13/viper"
)

var _ = Describe("Helpers", func() {
	Context("CreateFilterQuery", func() {
		var contractAddresses = []common.Address{
			common.HexToAddress("0x78f2c2af65126834c51822f56be0d7469d7a523e"),
			common.HexToAddress("0xa5679C04fc3d9d8b0AaB1F0ab83555b301cA70Ea"),
		}

		It("creates a filter query with contract addresses from viper config", func() {
			viper.SetConfigType("toml")
			readConfigErr := viper.ReadConfig(bytes.NewBuffer(testConfig))
			Expect(readConfigErr).NotTo(HaveOccurred())

			filterQuery, filterQueryErr := streamer.CreateFilterQuery()
			Expect(filterQueryErr).NotTo(HaveOccurred())
			Expect(filterQuery.Addresses).To(ContainElements(contractAddresses))
		})

		It("returns an error if one of the addresses is empty", func() {
			viper.SetConfigType("toml")
			readConfigErr := viper.ReadConfig(bytes.NewBuffer(badTestConfig))
			Expect(readConfigErr).NotTo(HaveOccurred())

			filterQuery, filterQueryErr := streamer.CreateFilterQuery()
			Expect(filterQueryErr).To(HaveOccurred())
			Expect(filterQueryErr).To(MatchError("test_contract_1 not parsed properly into viper"))
			Expect(filterQuery.Addresses).NotTo(ContainElement(contractAddresses[1]))
		})
	})
})

var testConfig = []byte(`
[contract]
  [contract.TEST_CONTRACT_1_0_0]
    address = "0x78f2c2af65126834c51822f56be0d7469d7a523e"
    abi = "[{\"inputs\":[]}]"
    deployed = 1
  [contract.TEST_CONTRACT_1_1_0]
    address = "0xa5679C04fc3d9d8b0AaB1F0ab83555b301cA70Ea"
    abi = "[{\"inputs\":[]}]"
    deployed = 2
`)

var badTestConfig = []byte(`
[contract]
  [contract.TEST_CONTRACT_1_0_0]
    address = "0x78f2c2af65126834c51822f56be0d7469d7a523e"
    abi = "[{\"inputs\":[]}]"
    deployed = 1
  [contract.TEST_CONTRACT_1.1.0]
    address = "0xa5679C04fc3d9d8b0AaB1F0ab83555b301cA70Ea"
    abi = "[{\"inputs\":[]}]"
    deployed = 2
`)
