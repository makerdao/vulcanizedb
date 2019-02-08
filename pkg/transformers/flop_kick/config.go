// VulcanizeDB
// Copyright © 2018 Vulcanize

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

package flop_kick

import (
	"github.com/vulcanize/vulcanizedb/pkg/transformers/shared"
	"github.com/vulcanize/vulcanizedb/pkg/transformers/shared/constants"
)

func GetFlopKickConfig() shared.TransformerConfig {
	return shared.TransformerConfig{
		TransformerName:     constants.FlopKickLabel,
		ContractAddresses:   []string{constants.FlopperContractAddress()},
		ContractAbi:         constants.FlopperABI(),
		Topic:               constants.GetFlopKickSignature(),
		StartingBlockNumber: constants.FlopperDeploymentBlock(),
		EndingBlockNumber:   -1,
	}
}