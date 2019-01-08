package dpos

import (
	"bytes"
	"encoding/json"
	"errors"
	"math/big"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru"
	log "github.com/sirupsen/logrus"
	"github.com/vapor/chain"
	"github.com/vapor/common"
	"github.com/vapor/config"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm/vmutil"
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
	SignerBlockReward     = big.NewInt(5e+18) // Block reward in wei for successfully mining a block first year
	defaultEpochLength    = uint64(3000000)   // Default number of blocks after which vote's period of validity
	defaultBlockPeriod    = uint64(3)         // Default minimum difference between two consecutive block's timestamps
	defaultMaxSignerCount = uint64(21)        //
	//defaultMinVoterBalance           = new(big.Int).Mul(big.NewInt(10000), big.NewInt(1e+18))
	defaultMinVoterBalance           = uint64(0)
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
	signer     string             // Ethereum address of the signing key
	signFn     SignerFn           // Signer function to authorize hashes with
	signTxFn   SignTxFn           // Sign transaction function to sign tx
	lock       sync.RWMutex       // Protects the signer fields
	lcsc       uint64             // Last confirmed side chain
}

// SignerFn is a signer callback function to request a hash to be signed by a backing account.
type SignerFn func(string, []byte) ([]byte, error)

// SignTxFn is a signTx
type SignTxFn func(string, *bc.Tx, *big.Int) (*bc.Tx, error)

//
func ecrecover(header *types.BlockHeader, sigcache *lru.ARCCache, c chain.Chain) (string, error) {

	xpub := &chainkd.XPub{}
	xpub.UnmarshalText(header.Coinbase)
	derivedPK := xpub.PublicKey()
	pubHash := crypto.Ripemd160(derivedPK)
	address, err := common.NewAddressWitnessPubKeyHash(pubHash, &consensus.ActiveNetParams)
	if err != nil {
		return "", err
	}

	return address.EncodeAddress(), nil
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
	if conf.MinVoterBalance == 0 {
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

// Authorize injects a private key into the consensus engine to mint new blocks with.
func (d *Dpos) Authorize(signer string /*, signFn SignerFn*/) {
	d.lock.Lock()
	defer d.lock.Unlock()

	d.signer = signer
	//d.signFn = signFn
}

// 从BLockHeader中获取到地址
func (d *Dpos) Author(header *types.BlockHeader, c chain.Chain) (string, error) {
	return ecrecover(header, d.signatures, c)
}

func (d *Dpos) VerifyHeader(c chain.Chain, header *types.BlockHeader, seal bool) error {
	return d.verifyCascadingFields(c, header, nil)
}

func (d *Dpos) VerifyHeaders(c chain.Chain, headers []*types.BlockHeader, seals []bool) (chan<- struct{}, <-chan error) {
	return nil, nil
}

func (d *Dpos) VerifySeal(c chain.Chain, header *types.BlockHeader) error {
	return nil
}

func (d *Dpos) verifyHeader(c chain.Chain, header *types.BlockHeader, parents []*types.BlockHeader) error {
	return nil
}

func (d *Dpos) verifyCascadingFields(c chain.Chain, header *types.BlockHeader, parents []*types.BlockHeader) error {
	// The genesis block is the always valid dead-end
	height := header.Height
	if height == 0 {
		return nil
	}

	var (
		parent *types.BlockHeader
		err    error
	)

	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent, err = c.GetHeaderByHeight(height - 1)
		if err != nil {
			return err
		}
	}

	if parent == nil {
		return errors.New("unknown ancestor")
	}

	if _, err = d.snapshot(c, height-1, header.PreviousBlockHash, parents, nil, defaultLoopCntRecalculateSigners); err != nil {
		return err
	}

	return d.verifySeal(c, header, parents)
}

// verifySeal checks whether the signature contained in the header satisfies the
// consensus protocol requirements. The method accepts an optional list of parent
// headers that aren't yet part of the local blockchain to generate the snapshots
// from.
func (d *Dpos) verifySeal(c chain.Chain, header *types.BlockHeader, parents []*types.BlockHeader) error {
	height := header.Height
	if height == 0 {
		return errUnknownBlock
	}
	// Retrieve the snapshot needed to verify this header and cache it
	snap, err := d.snapshot(c, height-1, header.PreviousBlockHash, parents, nil, defaultLoopCntRecalculateSigners)
	if err != nil {
		return err
	}

	// Resolve the authorization key and check against signers
	signer, err := ecrecover(header, d.signatures, c)
	if err != nil {
		return err
	}

	if height > d.config.MaxSignerCount {
		var (
			parent *types.BlockHeader
			err    error
		)
		if len(parents) > 0 {
			parent = parents[len(parents)-1]
		} else {
			if parent, err = c.GetHeaderByHeight(height - 1); err != nil {
				return err
			}
		}

		//parent
		xpub := &chainkd.XPub{}
		xpub.UnmarshalText(parent.Coinbase)
		derivedPK := xpub.PublicKey()
		pubHash := crypto.Ripemd160(derivedPK)
		parentCoinbase, err := common.NewAddressWitnessPubKeyHash(pubHash, &consensus.ActiveNetParams)
		if err != nil {
			return err
		}

		//current
		xpub.UnmarshalText(header.Coinbase)
		derivedPK = xpub.PublicKey()
		pubHash = crypto.Ripemd160(derivedPK)
		currentCoinbase, err := common.NewAddressWitnessPubKeyHash(pubHash, &consensus.ActiveNetParams)
		if err != nil {
			return err
		}

		parentHeaderExtra := HeaderExtra{}
		if err = json.Unmarshal(parent.Extra[extraVanity:len(parent.Extra)-extraSeal], &parentHeaderExtra); err != nil {
			return err
		}

		currentHeaderExtra := HeaderExtra{}
		if err = json.Unmarshal(header.Extra[extraVanity:len(header.Extra)-extraSeal], &currentHeaderExtra); err != nil {
			return err
		}

		// verify signerqueue
		if height%d.config.MaxSignerCount == 0 {
			err := snap.verifySignerQueue(currentHeaderExtra.SignerQueue)
			if err != nil {
				return err
			}

		} else {
			for i := 0; i < int(d.config.MaxSignerCount); i++ {
				if parentHeaderExtra.SignerQueue[i] != currentHeaderExtra.SignerQueue[i] {
					return errInvalidSignerQueue
				}
			}
		}
		// verify missing signer for punish
		parentSignerMissing := getSignerMissing(parentCoinbase.EncodeAddress(), currentCoinbase.EncodeAddress(), parentHeaderExtra)
		if len(parentSignerMissing) != len(currentHeaderExtra.SignerMissing) {
			return errPunishedMissing
		}
		for i, signerMissing := range currentHeaderExtra.SignerMissing {
			if parentSignerMissing[i] != signerMissing {
				return errPunishedMissing
			}
		}

	}

	if !snap.inturn(signer, header.Timestamp) {
		return errUnauthorized
	}

	return nil
}

// Prepare implements consensus.Engine, preparing all the consensus fields of the header for running the transactions on top.
func (d *Dpos) Prepare(c chain.Chain, header *types.BlockHeader) error {
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

func (d *Dpos) Finalize(c chain.Chain, header *types.BlockHeader, txs []*bc.Tx) error {
	height := header.Height
	parent, err := c.GetHeaderByHeight(height - 1)
	if parent == nil {
		return err
	}
	//parent
	var xpub chainkd.XPub
	xpub.UnmarshalText(parent.Coinbase)
	pubHash := crypto.Ripemd160(xpub.PublicKey())
	parentCoinbase, err := common.NewAddressWitnessPubKeyHash(pubHash, &consensus.ActiveNetParams)
	if err != nil {
		return err
	}

	//current
	xpub.UnmarshalText(header.Coinbase)
	pubHash = crypto.Ripemd160(xpub.PublicKey())
	currentCoinbase, err := common.NewAddressWitnessPubKeyHash(pubHash, &consensus.ActiveNetParams)
	if err != nil {
		return err
	}

	//header.Timestamp
	t := new(big.Int).Add(new(big.Int).SetUint64(parent.Timestamp), new(big.Int).SetUint64(d.config.Period))
	header.Timestamp = t.Uint64()

	if header.Timestamp < uint64(time.Now().Unix()) {
		header.Timestamp = uint64(time.Now().Unix())
	}

	if len(header.Extra) < extraVanity {
		header.Extra = append(header.Extra, bytes.Repeat([]byte{0x00}, extraVanity-len(header.Extra))...)
	}

	header.Extra = header.Extra[:extraVanity]
	// genesisVotes write direct into snapshot, which number is 1
	var genesisVotes []*Vote
	parentHeaderExtra := HeaderExtra{}
	currentHeaderExtra := HeaderExtra{}
	if height == 1 {
		alreadyVote := make(map[string]struct{})
		for _, voter := range d.config.SelfVoteSigners {
			if _, ok := alreadyVote[voter]; !ok {
				genesisVotes = append(genesisVotes, &Vote{
					Voter:     voter,
					Candidate: voter,
					Stake:     0,
					//Stake:     state.GetBalance(voter),
				})
				alreadyVote[voter] = struct{}{}
			}
		}
	} else {
		parentHeaderExtraByte := parent.Extra[extraVanity : len(parent.Extra)-extraSeal]
		if err := json.Unmarshal(parentHeaderExtraByte, &parentHeaderExtra); err != nil {
			return err
		}
		currentHeaderExtra.ConfirmedBlockNumber = parentHeaderExtra.ConfirmedBlockNumber
		currentHeaderExtra.SignerQueue = parentHeaderExtra.SignerQueue
		currentHeaderExtra.LoopStartTime = parentHeaderExtra.LoopStartTime
		currentHeaderExtra.SignerMissing = getSignerMissing(parentCoinbase.EncodeAddress(), currentCoinbase.EncodeAddress(), parentHeaderExtra)
	}

	// calculate votes write into header.extra
	currentHeaderExtra, err = d.processCustomTx(currentHeaderExtra, c, header, txs)
	if err != nil {
		return err
	}
	// Assemble the voting snapshot to check which votes make sense
	snap, err := d.snapshot(c, height-1, header.PreviousBlockHash, nil, genesisVotes, defaultLoopCntRecalculateSigners)
	if err != nil {
		return err
	}

	currentHeaderExtra.ConfirmedBlockNumber = snap.getLastConfirmedBlockNumber(currentHeaderExtra.CurrentBlockConfirmations).Uint64()
	// write signerQueue in first header, from self vote signers in genesis block
	if height == 1 {
		currentHeaderExtra.LoopStartTime = d.config.GenesisTimestamp
		for i := 0; i < int(d.config.MaxSignerCount); i++ {
			currentHeaderExtra.SignerQueue = append(currentHeaderExtra.SignerQueue, d.config.SelfVoteSigners[i%len(d.config.SelfVoteSigners)])
		}
	}
	if height%d.config.MaxSignerCount == 0 {
		//currentHeaderExtra.LoopStartTime = header.Time.Uint64()
		currentHeaderExtra.LoopStartTime = currentHeaderExtra.LoopStartTime + d.config.Period*d.config.MaxSignerCount
		// create random signersQueue in currentHeaderExtra by snapshot.Tally
		currentHeaderExtra.SignerQueue = []string{}
		newSignerQueue, err := snap.createSignerQueue()
		if err != nil {
			return err
		}

		currentHeaderExtra.SignerQueue = newSignerQueue

	}
	// encode header.extra
	currentHeaderExtraEnc, err := json.Marshal(currentHeaderExtra)
	if err != nil {
		return err
	}
	header.Extra = append(header.Extra, currentHeaderExtraEnc...)
	header.Extra = append(header.Extra, make([]byte, extraSeal)...)
	return nil
}

func (d *Dpos) Seal(c chain.Chain, block *types.Block) (*types.Block, error) {
	header := block.BlockHeader
	height := header.Height
	if height == 0 {
		return nil, errUnknownBlock
	}

	if d.config.Period == 0 && len(block.Transactions) == 0 {
		return nil, errWaitTransactions
	}
	// Bail out if we're unauthorized to sign a block
	snap, err := d.snapshot(c, height-1, header.PreviousBlockHash, nil, nil, defaultLoopCntRecalculateSigners)
	if err != nil {
		return nil, err
	}
	if !snap.inturn(d.signer, header.Timestamp) {
		return nil, errUnauthorized
	}

	var xPrv chainkd.XPrv
	if config.CommonConfig.Consensus.Dpos.XPrv == "" {
		return nil, errors.New("Signer is empty")
	}
	xPrv.UnmarshalText([]byte(config.CommonConfig.Consensus.Dpos.XPrv))
	sign := xPrv.Sign(block.BlockCommitment.TransactionsMerkleRoot.Bytes())
	pubHash := crypto.Ripemd160(xPrv.XPub().PublicKey())

	control, err := vmutil.P2WPKHProgram([]byte(pubHash))
	if err != nil {
		return nil, err
	}

	block.Proof = types.Proof{Sign: sign, ControlProgram: control}
	return block, nil
}

func (d *Dpos) IsSealer(c chain.Chain, hash bc.Hash, header *types.BlockHeader, headerTime uint64) (bool, error) {
	var (
		snap    *Snapshot
		headers []*types.BlockHeader
	)
	h := hash
	height := header.Height
	for snap == nil {
		// If an in-memory snapshot was found, use that
		if s, ok := d.recents.Get(h); ok {
			snap = s.(*Snapshot)
			break
		}
		// If an on-disk checkpoint snapshot can be found, use that
		if height%checkpointInterval == 0 {
			if s, err := loadSnapshot(d.config, d.signatures, d.store, h); err == nil {
				log.WithFields(log.Fields{"func": "IsSealer", "number": height, "hash": h}).Warn("Loaded voting snapshot from disk")
				snap = s
				break
			} else {
				log.Warn("loadSnapshot:", err)
			}
		}

		if height == 0 {
			genesis, err := c.GetHeaderByHeight(0)
			if err != nil {
				return false, err
			}
			var genesisVotes []*Vote
			alreadyVote := make(map[string]struct{})
			for _, voter := range d.config.SelfVoteSigners {
				if _, ok := alreadyVote[voter]; !ok {
					genesisVotes = append(genesisVotes, &Vote{
						Voter:     voter,
						Candidate: voter,
						Stake:     0,
						//Stake:     state.GetBalance(voter),
					})
					alreadyVote[voter] = struct{}{}
				}
			}
			snap = newSnapshot(d.config, d.signatures, genesis.Hash(), genesisVotes, defaultLoopCntRecalculateSigners)
			if err := snap.store(d.store); err != nil {
				return false, err
			}
			log.Info("Stored genesis voting snapshot to disk")
			break
		}

		header, err := c.GetHeaderByHeight(height)
		if header == nil || err != nil {
			return false, errors.New("unknown ancestor")
		}

		height, h = height-1, header.PreviousBlockHash
	}

	snap, err := snap.apply(headers)
	if err != nil {
		return false, err
	}

	d.recents.Add(snap.Hash, snap)

	if snap != nil {
		loopIndex := int((headerTime-snap.LoopStartTime)/snap.config.Period) % len(snap.Signers)
		if loopIndex >= len(snap.Signers) {
			return false, nil
		} else if *snap.Signers[loopIndex] != d.signer {
			return false, nil

		}
		return true, nil
	} else {
		return false, nil
	}
}

// snapshot retrieves the authorization snapshot at a given point in time.
func (d *Dpos) snapshot(c chain.Chain, number uint64, hash bc.Hash, parents []*types.BlockHeader, genesisVotes []*Vote, lcrs uint64) (*Snapshot, error) {

	var (
		headers []*types.BlockHeader
		snap    *Snapshot
	)
	h := hash

	for snap == nil {
		// If an in-memory snapshot was found, use that
		if s, ok := d.recents.Get(h); ok {
			snap = s.(*Snapshot)
			break
		}
		// If an on-disk checkpoint snapshot can be found, use that
		if number%checkpointInterval == 0 {
			if s, err := loadSnapshot(d.config, d.signatures, d.store, h); err == nil {
				log.WithFields(log.Fields{"number": number, "hash": h}).Warn("Loaded voting snapshot from disk")
				snap = s
				break
			}
		}
		if number == 0 {
			genesis, err := c.GetHeaderByHeight(0)
			if err != nil {
				return nil, err
			}
			if err := d.VerifyHeader(c, genesis, false); err != nil {
				return nil, err
			}

			snap = newSnapshot(d.config, d.signatures, genesis.Hash(), genesisVotes, lcrs)
			if err := snap.store(d.store); err != nil {
				return nil, err
			}
			log.Info("Stored genesis voting snapshot to disk")
			break
		}
		var header *types.BlockHeader
		if len(parents) > 0 {
			header = parents[len(parents)-1]
			if header.Hash() != h || header.Height != number {
				return nil, errors.New("unknown ancestor")
			}
			parents = parents[:len(parents)-1]
		} else {
			var err error
			header, err = c.GetHeaderByHeight(number)
			if header == nil || err != nil {
				return nil, errors.New("unknown ancestor")
			}
		}
		headers = append(headers, header)
		number, h = number-1, header.PreviousBlockHash
	}

	// Previous snapshot found, apply any pending headers on top of it
	for i := 0; i < len(headers)/2; i++ {
		headers[i], headers[len(headers)-1-i] = headers[len(headers)-1-i], headers[i]
	}
	snap, err := snap.apply(headers)
	if err != nil {
		return nil, err
	}
	d.recents.Add(snap.Hash, snap)

	// If we've generated a new checkpoint snapshot, save to disk
	if snap.Number%checkpointInterval == 0 && len(headers) > 0 {
		if err = snap.store(d.store); err != nil {
			return nil, err
		}
		log.Info("Stored voting snapshot to disk", "number", snap.Number, "hash", snap.Hash)
	}
	return snap, err
}

// Get the signer missing from last signer till header.Coinbase
func getSignerMissing(lastSigner string, currentSigner string, extra HeaderExtra) []string {

	var signerMissing []string
	recordMissing := false
	for _, signer := range extra.SignerQueue {
		if signer == lastSigner {
			recordMissing = true
			continue
		}
		if signer == currentSigner {
			break
		}
		if recordMissing {
			signerMissing = append(signerMissing, signer)
		}
	}
	return signerMissing
}
