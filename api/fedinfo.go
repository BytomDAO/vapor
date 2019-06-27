package api

import (
	"context"
	"net"

	"github.com/vapor/errors"
	"github.com/vapor/netsync/peers"
	"github.com/vapor/p2p"
	"github.com/vapor/version"
)

type FedInfo struct{}

// GetFedInfo return federation information
func (a *API) GetFedInfo() *FedInfo {
	return a.fedInfo
}
