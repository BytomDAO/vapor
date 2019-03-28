package protocolbc

import "github.com/vapor/protocol/bc"

// Block is block struct in bc level
type Block struct {
	*bc.BytomBlockHeader
	ID           bc.Hash
	Transactions []*Tx
}
