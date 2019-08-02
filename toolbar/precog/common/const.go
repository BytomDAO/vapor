package common

const (
	NodeUnknownStatus uint8 = iota
	NodeHealthyStatus
	NodeCongestedStatus
	NodeBusyStatus
	NodeOfflineStatus
)

var StatusMap = map[uint8]string{
	NodeUnknownStatus:   "unknown",
	NodeHealthyStatus:   "healthy",
	NodeCongestedStatus: "congested",
	NodeBusyStatus:      "busy",
	NodeOfflineStatus:   "offline",
}
