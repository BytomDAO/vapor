package dpos

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/go-loom/types"
	lru "github.com/hashicorp/golang-lru"
	log "github.com/sirupsen/logrus"
	"github.com/vapor/common"
	"github.com/vapor/config"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
)

const (
	inMemorySnapshots  = 128             // Number of recent vote snapshots to keep in memory
	inMemorySignatures = 4096            // Number of recent block signatures to keep in memory
	secondsPerYear     = 365 * 24 * 3600 // Number of seconds for one year
	checkpointInterval = 360             // About N hours if config.period is N
	module             = "dpos"
)

//delegated-proof-of-stake protocol constants.
var (
	SignerBlockReward                = big.NewInt(5e+18) // Block reward in wei for successfully mining a block first year
	defaultEpochLength               = uint64(3000000)   // Default number of blocks after which vote's period of validity
	defaultBlockPeriod               = uint64(3)         // Default minimum difference between two consecutive block's timestamps
	defaultMaxSignerCount            = uint64(21)        //
	defaultMinVoterBalance           = new(big.Int).Mul(big.NewInt(10000), big.NewInt(1e+18))
	extraVanity                      = 32            // Fixed number of extra-data prefix bytes reserved for signer vanity
	extraSeal                        = 65            // Fixed number of extra-data suffix bytes reserved for signer seal
	defaultDifficulty                = big.NewInt(1) // Default difficulty
	defaultLoopCntRecalculateSigners = uint64(10)    // Default loop count to recreate signers from top tally
	minerRewardPerThousand           = uint64(618)   // Default reward for miner in each block from block reward (618/1000)
	candidateNeedPD                  = false         // is new candidate need Proposal & Declare process
)

var (
	// errUnknownBlock is returned when the list of signers is requested for a block
	// that is not part of the local blockchain.
	errUnknownBlock = errors.New("unknown block")

	// errMissingVanity is returned if a block's extra-data section is shorter than
	// 32 bytes, which is required to store the signer vanity.
	errMissingVanity = errors.New("extra-data 32 byte vanity prefix missing")

	// errMissingSignature is returned if a block's extra-data section doesn't seem
	// to contain a 65 byte secp256k1 signature.
	errMissingSignature = errors.New("extra-data 65 byte suffix signature missing")

	// errInvalidMixDigest is returned if a block's mix digest is non-zero.
	errInvalidMixDigest = errors.New("non-zero mix digest")

	// errInvalidUncleHash is returned if a block contains an non-empty uncle list.
	errInvalidUncleHash = errors.New("non empty uncle hash")

	// ErrInvalidTimestamp is returned if the timestamp of a block is lower than
	// the previous block's timestamp + the minimum block period.
	ErrInvalidTimestamp = errors.New("invalid timestamp")

	// errInvalidVotingChain is returned if an authorization list is attempted to
	// be modified via out-of-range or non-contiguous headers.
	errInvalidVotingChain = errors.New("invalid voting chain")

	// errUnauthorized is returned if a header is signed by a non-authorized entity.
	errUnauthorized = errors.New("unauthorized")

	// errPunishedMissing is returned if a header calculate punished signer is wrong.
	errPunishedMissing = errors.New("punished signer missing")

	// errWaitTransactions is returned if an empty block is attempted to be sealed
	// on an instant chain (0 second period). It's important to refuse these as the
	// block reward is zero, so an empty block just bloats the chain... fast.
	errWaitTransactions = errors.New("waiting for transactions")

	// errUnclesNotAllowed is returned if uncles exists
	errUnclesNotAllowed = errors.New("uncles not allowed")

	// errCreateSignerQueueNotAllowed is returned if called in (block number + 1) % maxSignerCount != 0
	errCreateSignerQueueNotAllowed = errors.New("create signer queue not allowed")

	// errInvalidSignerQueue is returned if verify SignerQueue fail
	errInvalidSignerQueue = errors.New("invalid signer queue")

	// errSignerQueueEmpty is returned if no signer when calculate
	errSignerQueueEmpty = errors.New("signer queue is empty")
)

type Dpos struct {
	config     *config.DposConfig // Consensus engine configuration parameters
	store      protocol.Store     // Database to store and retrieve snapshot checkpoints
	recents    *lru.ARCCache      // Snapshots for recent block to speed up reorgs
	signatures *lru.ARCCache      // Signatures of recent blocks to speed up mining
	signer     common.Address     // Ethereum address of the signing key
	signFn     SignerFn           // Signer function to authorize hashes with
	signTxFn   SignTxFn           // Sign transaction function to sign tx
	lock       sync.RWMutex       // Protects the signer fields
	lcsc       uint64             // Last confirmed side chain
}

// SignerFn is a signer callback function to request a hash to be signed by a backing account.
type SignerFn func(common.Address, []byte) ([]byte, error)

// SignTxFn is a signTx
type SignTxFn func(common.Address, *types.Transaction, *big.Int) (*types.Transaction, error)

//
func ecrecover(header *types.BlockHeader, sigcache *lru.ARCCache) (common.Address, error) {
	return nil, nil
}

func sigHash(header *types.BlockHeader) (hash bc.Hash) {
	return bc.Hash{}
}

//
func New(config *config.DposConfig, store protocol.Store) *Dpos {
	conf := *config
	if conf.Epoch == 0 {
		conf.Epoch = defaultEpochLength
	}
	if conf.Period == 0 {
		conf.Period = defaultBlockPeriod
	}
	if conf.MaxSignerCount == 0 {
		conf.MaxSignerCount = defaultMaxSignerCount
	}
	if conf.MinVoterBalance.Uint64() == 0 {
		conf.MinVoterBalance = defaultMinVoterBalance
	}

	// Allocate the snapshot caches and create the engine
	recents, _ := lru.NewARC(inMemorySnapshots)
	signatures, _ := lru.NewARC(inMemorySignatures)
	return &Dpos{
		config:     &conf,
		store:      store,
		recents:    recents,
		signatures: signatures,
	}
}

// 从BLockHeader中获取到地址
func (d *Dpos) Author(header *types.BlockHeader) (common.Address, error) {
	return ecrecover(header, d.signatures)
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the header for running the transactions on top.
func (d *Dpos) Prepare(c *protocol.Chain, header *bc.BlockHeader) error {
	if d.config.GenesisTimestamp < uint64(time.Now().Unix()) {
		return nil
	}

	if header.Height == 1 {
		for {
			delay := time.Unix(int64(d.config.GenesisTimestamp-2), 0).Sub(time.Now())
			if delay <= time.Duration(0) {
				log.WithFields(log.Fields{"module": module, "time": time.Now()}).Info("Ready for seal block")
				break
			} else if delay > time.Duration(d.config.Period)*time.Second {
				delay = time.Duration(d.config.Period) * time.Second
			}
			log.WithFields(log.Fields{"module": module, "delay": time.Duration(time.Unix(int64(d.config.GenesisTimestamp-2), 0).Sub(time.Now()))}).Info("Waiting for seal block")
			select {
			case <-time.After(delay):
				continue
			}
		}
	}
	return nil
}

func (d *Dpos) Finalize(c *protocol.Chain, header *bc.BlockHeader, txs []*types.Transaction) (*bc.Block, error) {
	height := c.BestBlockHeight()
	fmt.Println(height)
	parent, err := c.GetHeaderByHeight(height - 1)
	if parent == nil {
		return nil, err
	}
	//header.Timestamp
	t := new(big.Int).Add(new(big.Int).SetUint64(parent.Timestamp), new(big.Int).SetUint64(d.config.Period))
	header.Timestamp = t.Uint64()

	if header.Timestamp < uint64(time.Now().Unix()) {
		header.Timestamp = uint64(time.Now().Unix())
	}
	

	return nil, nil
}

func (d *Dpos) Seal() {

}
