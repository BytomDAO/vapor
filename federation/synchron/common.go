package synchron

import (
	"bytes"

	btmTypes "github.com/bytom/protocol/bc/types"

	vaporCfg "github.com/vapor/config"
	"github.com/vapor/errors"
)

var (
	ErrInconsistentDB = errors.New("inconsistent db status")
)

func isDepositFromMainchain(tx *btmTypes.Tx) bool {
	fedProg := vaporCfg.FederationProgrom(vaporCfg.CommonConfig)
	for _, output := range tx.Outputs {
		if bytes.Equal(output.OutputCommitment.ControlProgram, fedProg) {
			return true
		}
	}
	return false
}

// func isWithdrawalToMainchain(tx *btmTypes.Tx) bool {
// 	for _, input := range tx.Inputs {
// 		if bytes.Equal(input.ControlProgram(), fedProg) {
// 			return true
// 		}
// 	}
// 	return false
// }
