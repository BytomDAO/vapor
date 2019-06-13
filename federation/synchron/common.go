package synchron

import (
	btmTypes "github.com/bytom/protocol/bc/types"

	"github.com/vapor/errors"
)

var ErrInconsistentDB = errors.New("inconsistent db status")

func isWithdrawalToMainchain(tx *btmTypes.Tx) bool {
	return true
}
