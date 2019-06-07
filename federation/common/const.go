package common

const (
	DepositDirection uint8 = iota
	WithdrawalDirection
)

const (
	CrossTxPendingStatus uint8 = iota
	CrossTxCompletedStatus
	CrossTxRejectedStatus
)

const (
	CrossTxSignPendingStatus uint8 = iota
	CrossTxSignCompletedStatus
	CrossTxSignRejectedStatus
)
