package common

const (
	_ uint8 = iota
	NodeHealthyStatus
	NodeCongestedStatus
	NodeBusyStatus
	NodeOfflineStatus
)

var StatusMap = map[uint8]string{
	NodeHealthyStatus:   "healthy",
	NodeCongestedStatus: "congested",
	NodeBusyStatus:      "busy",
	NodeOfflineStatus:   "offline",
}
