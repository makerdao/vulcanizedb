package streamer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func CreateFilterQuery() (ethereum.FilterQuery, error) {
	addressStrings, addressStringErr := getAddressesFromViper()
	if addressStringErr != nil {
		return ethereum.FilterQuery{}, addressStringErr
	}
	logWatchedAddresses(addressStrings)
	addresses := convertAddressStrings(addressStrings)

	return ethereum.FilterQuery{Addresses: addresses}, nil
}

func logWatchedAddresses(watchedAddresses []string) {
	logrus.Infof("Creating a filter query for %d watched addresses", len(watchedAddresses))
	addressesToLog := strings.Join(watchedAddresses[:], ", ")
	logrus.Infof("Watched addresses: %s", addressesToLog)
}

func getAddressesFromViper() ([]string, error) {
	contracts := viper.GetStringMapString("contract")
	var addressStrings []string
	for contractName := range contracts {
		address, ok := viper.GetStringMapString("contract." + contractName)["address"]
		if !ok {
			parseErr := errors.New(fmt.Sprintf("%s not parsed properly into viper", contractName))
			return addressStrings, parseErr
		}
		addressStrings = append(addressStrings, address)
	}
	return addressStrings, nil
}

func convertAddressStrings(addressStrings []string) []common.Address {
	var addresses []common.Address
	for _, addressString := range addressStrings {
		addresses = append(addresses, common.HexToAddress(addressString))
	}

	return addresses
}
