package consensus

import (
	"github.com/vapor/chain"
	"github.com/vapor/common"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

type AddressBalance struct {
	Address string
	Balance int64
}

type DelegateInterface interface {
	ConsensusName() string
}

// Engine is an algorithm agnostic consensus engine.
type Engine interface {
	Init(c chain.Chain, delegateNumber, intervalTime, blockHeight uint64, blockHash bc.Hash) error
	Finish() error
	IsMining(address common.Address, t uint64) (interface{}, error)
	ProcessRegister(delegateAddress string, delegateName string, hash bc.Hash, height uint64) bool
	ProcessVote(voterAddress string, delegates []string, hash bc.Hash, height uint64) bool
	ProcessCancelVote(voterAddress string, delegates []string, hash bc.Hash, height uint64) bool
	UpdateAddressBalance(addressBalance []AddressBalance)
	CheckBlockHeader(header types.BlockHeader) error
	CheckBlock(block types.Block, fIsCheckDelegateInfo bool) error
	IsValidBlockCheckIrreversibleBlock(height uint64, hash bc.Hash) error
	GetOldBlockHeight() uint64
	GetOldBlockHash() bc.Hash
}
