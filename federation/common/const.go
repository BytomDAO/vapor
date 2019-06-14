package common

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
