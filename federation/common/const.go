package common

const (
	CrossTxInitiatedStatus uint8 = iota
	CrossTxPendingStatus
	CrossTxRejectedStatus
	CrossTxSubmittedStatus
	CrossTxCompletedStatus
)

const (
	CrossTxSignPendingStatus uint8 = iota
	CrossTxSignCompletedStatus
	CrossTxSignRejectedStatus
)

const (
	MainchainNameLabel = "bytom"
	SidechainNameLabel = "vapor"
)
