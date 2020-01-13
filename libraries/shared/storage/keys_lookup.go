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

package storage

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/makerdao/vulcanizedb/libraries/shared/storage/types"
)

func AddHashedKeys(currentMappings map[common.Hash]types.ValueMetadata) map[common.Hash]types.ValueMetadata {
	copyOfCurrentMappings := make(map[common.Hash]types.ValueMetadata)
	for k, v := range currentMappings {
		copyOfCurrentMappings[k] = v
	}
	for k, v := range copyOfCurrentMappings {
		currentMappings[hashKey(k)] = v
	}
	return currentMappings
}

func hashKey(key common.Hash) common.Hash {
	return crypto.Keccak256Hash(key.Bytes())
}
