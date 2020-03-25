package consensus

import (
	"encoding/binary"
	"fmt"

	"github.com/bytom/vapor/protocol/bc"
)

// basic constant
const (
	BTMAlias = "BTM"

	PayToWitnessPubKeyHashDataSize = 20
	PayToWitnessScriptHashDataSize = 32

	_ = iota
	SoftFork001
)

// BTMAssetID is BTM's asset id, the soul asset of Bytom
var BTMAssetID = &bc.AssetID{
	V0: binary.BigEndian.Uint64([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
	V1: binary.BigEndian.Uint64([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
	V2: binary.BigEndian.Uint64([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
	V3: binary.BigEndian.Uint64([]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}),
}

// BTMDefinitionMap is the ....
var BTMDefinitionMap = map[string]interface{}{
	"name":        BTMAlias,
	"symbol":      BTMAlias,
	"decimals":    8,
	"description": `Bytom Official Issue`,
}

// BasicConfig indicate the basic config
type BasicConfig struct {
	// gas config
	MaxBlockGas      uint64 // the max used gas for all transactions of a block
	MaxGasAmount     int64  // the max gas for a transaction
	DefaultGasCredit int64  // the max default credit gas for a transaction with non-BTM asset
	VMGasRate        int64  // the gas rate for VM
	StorageGasRate   int64  // the gas rate for storage

	// utxo config
	VotePendingBlockNumber     uint64 // the valid block interval for vote utxo after the vote transaction is confirmed
	CoinbasePendingBlockNumber uint64 // the valid block interval for coinbase utxo after the coinbase transaction is confirmed
	CoinbaseArbitrarySizeLimit int    // the max size for coinbase arbitrary
}

// DPOSConfig indicate the dpos consensus config
type DPOSConfig struct {
	NumOfConsensusNode      int64  // the number of consensus node
	BlockNumEachNode        uint64 // the number of generated continuous blocks for each node
	RoundVoteBlockNums      uint64 // the block interval which count the vote result in a round
	MinConsensusNodeVoteNum uint64 // the min BTM amount for becoming consensus node(the unit is neu)
	MinVoteOutputAmount     uint64 // the min BTM amount for voting output in a transaction(the unit is neu)
	BlockTimeInterval       uint64 // the block time interval for producting a block
	MaxTimeOffsetMs         uint64 // the max number of seconds a block time is allowed to be ahead of the current time
}

// Checkpoint identifies a known good point in the block chain.  Using
// checkpoints allows a few optimizations for old blocks during initial download
// and also prevents forks from old blocks.
type Checkpoint struct {
	Height uint64
	Hash   bc.Hash
}

// ProducerSubsidy is a subsidy to the producer of the generated block
type ProducerSubsidy struct {
	BeginBlock uint64
	EndBlock   uint64
	Subsidy    uint64
}

// MovRewardProgram is a reward address corresponding to the range of the specified block height when matching transactions
type MovRewardProgram struct {
	BeginBlock uint64
	EndBlock   uint64
	Program    string
}

// Params store the config for different network
type Params struct {
	// Name defines a human-readable identifier for the network.
	Name string

	// Bech32HRPSegwit defines the prefix of address for the network
	Bech32HRPSegwit string

	// DefaultPort defines the default peer-to-peer port for the network.
	DefaultPort string

	// BasicConfig defines the gas and utxo relatived paramters.
	BasicConfig

	// DPOSConfig defines the dpos consensus paramters.
	DPOSConfig

	// DNSSeeds defines a list of DNS seeds for the network that are used
	// as one method to discover peers.
	DNSSeeds []string

	// Checkpoints defines the checkpoint blocks
	Checkpoints []Checkpoint

	// ProducerSubsidys defines the producer subsidy by block height
	ProducerSubsidys []ProducerSubsidy

	SoftForkPoint map[uint64]uint64

	// Mov will only start when the block height is greater than this value
	MovStartHeight uint64

	// Used to receive rewards for matching transactions
	MovRewardPrograms []MovRewardProgram
}

// ActiveNetParams is the active NetParams
var ActiveNetParams = MainNetParams

// NetParams is the correspondence between chain_id and Params
var NetParams = map[string]Params{
	"mainnet": MainNetParams,
	"testnet": TestNetParams,
	"solonet": SoloNetParams,
}

// MainNetParams is the config for vapor-mainnet
var MainNetParams = Params{
	Name:            "main",
	Bech32HRPSegwit: "vp",
	DefaultPort:     "56656",
	DNSSeeds:        []string{"www.mainnetseed.vapor.io"},
	BasicConfig: BasicConfig{
		MaxBlockGas:                uint64(10000000),
		MaxGasAmount:               int64(640000),
		DefaultGasCredit:           int64(160000),
		StorageGasRate:             int64(1),
		VMGasRate:                  int64(200),
		VotePendingBlockNumber:     uint64(3456000),
		CoinbasePendingBlockNumber: uint64(7200),
		CoinbaseArbitrarySizeLimit: 128,
	},
	DPOSConfig: DPOSConfig{
		NumOfConsensusNode:      10,
		BlockNumEachNode:        12,
		MinConsensusNodeVoteNum: uint64(100000000000000),
		MinVoteOutputAmount:     uint64(100000000),
		BlockTimeInterval:       500,
		RoundVoteBlockNums:      1200,
		MaxTimeOffsetMs:         2000,
	},
	Checkpoints: []Checkpoint{},
	ProducerSubsidys: []ProducerSubsidy{
		{BeginBlock: 1, EndBlock: 63072000, Subsidy: 9512938},
	},
	SoftForkPoint:  map[uint64]uint64{SoftFork001: 10461600},
	MovStartHeight: 42884800,
	MovRewardPrograms: []MovRewardProgram{
		{
			BeginBlock: 1,
			EndBlock:   126144000,
			Program:    "00141d00f85e220e35a23282cfc7f91fe7b34bf6dc18",
		},
	},
}

// TestNetParams is the config for vapor-testnet
var TestNetParams = Params{
	Name:            "test",
	Bech32HRPSegwit: "tp",
	DefaultPort:     "56657",
	DNSSeeds:        []string{"www.testnetseed.vapor.io"},
	BasicConfig: BasicConfig{
		MaxBlockGas:                uint64(10000000),
		MaxGasAmount:               int64(640000),
		DefaultGasCredit:           int64(160000),
		StorageGasRate:             int64(1),
		VMGasRate:                  int64(200),
		VotePendingBlockNumber:     uint64(3456000),
		CoinbasePendingBlockNumber: uint64(7200),
		CoinbaseArbitrarySizeLimit: 128,
	},
	DPOSConfig: DPOSConfig{
		NumOfConsensusNode:      10,
		BlockNumEachNode:        12,
		MinConsensusNodeVoteNum: uint64(100000000000000),
		MinVoteOutputAmount:     uint64(100000000),
		BlockTimeInterval:       500,
		RoundVoteBlockNums:      1200,
		MaxTimeOffsetMs:         2000,
	},
	Checkpoints: []Checkpoint{},
	ProducerSubsidys: []ProducerSubsidy{
		{BeginBlock: 1, EndBlock: 63072000, Subsidy: 15000000},
	},
}

// SoloNetParams is the config for vapor solonet
var SoloNetParams = Params{
	Name:            "solo",
	Bech32HRPSegwit: "sp",
	DefaultPort:     "56658",
	BasicConfig: BasicConfig{
		MaxBlockGas:                uint64(10000000),
		MaxGasAmount:               int64(200000),
		DefaultGasCredit:           int64(160000),
		StorageGasRate:             int64(1),
		VMGasRate:                  int64(200),
		VotePendingBlockNumber:     uint64(10000),
		CoinbasePendingBlockNumber: uint64(1200),
		CoinbaseArbitrarySizeLimit: 128,
	},
	DPOSConfig: DPOSConfig{
		NumOfConsensusNode:      10,
		BlockNumEachNode:        12,
		MinConsensusNodeVoteNum: uint64(100000000000000),
		MinVoteOutputAmount:     uint64(100000000),
		BlockTimeInterval:       500,
		RoundVoteBlockNums:      1200,
		MaxTimeOffsetMs:         2000,
	},
	Checkpoints: []Checkpoint{},
	ProducerSubsidys: []ProducerSubsidy{
		{BeginBlock: 0, EndBlock: 0, Subsidy: 24},
		{BeginBlock: 1, EndBlock: 840000, Subsidy: 24},
		{BeginBlock: 840001, EndBlock: 1680000, Subsidy: 12},
		{BeginBlock: 1680001, EndBlock: 3360000, Subsidy: 6},
	},
}

// BlockSubsidy calculate the coinbase rewards on given block height
func BlockSubsidy(height uint64) uint64 {
	for _, subsidy := range ActiveNetParams.ProducerSubsidys {
		if height >= subsidy.BeginBlock && height <= subsidy.EndBlock {
			return subsidy.Subsidy
		}
	}
	return 0
}

// BytomMainNetParams is the config for bytom mainnet
func BytomMainNetParams(vaporParam *Params) *Params {
	bech32HRPSegwit := "sm"
	switch vaporParam.Name {
	case "main":
		bech32HRPSegwit = "bm"
	case "test":
		bech32HRPSegwit = "tm"
	}
	return &Params{Bech32HRPSegwit: bech32HRPSegwit}
}

// InitActiveNetParams load the config by chain ID
func InitActiveNetParams(chainID string) error {
	var exist bool
	if ActiveNetParams, exist = NetParams[chainID]; !exist {
		return fmt.Errorf("chain_id[%v] don't exist", chainID)
	}
	return nil
}
