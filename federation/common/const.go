package common

import (
	"github.com/vapor/errors"
)

const (
	CrossTxPendingStatus uint8 = iota
	CrossTxRejectedStatus
	CrossTxSubmittedStatus
	CrossTxCompletedStatus
)

const (
	CrossTxSignPendingStatus uint8 = iota
	CrossTxSignCompletedStatus
	CrossTxSignRejectedStatus
)

var ErrInconsistentDB = errors.New("inconsistent db status")
