package dpos

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"time"

	lru "github.com/hashicorp/golang-lru"
	"github.com/vapor/config"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	defaultFullCredit               = 1000 // no punished
	missingPublishCredit            = 100  // punished for missing one block seal
	signRewardCredit                = 10   // seal one block
	autoRewardCredit                = 1    // credit auto recover for each block
	minCalSignerQueueCredit         = 300  // when calculate the signerQueue
	defaultOfficialMaxSignerCount   = 21   // official max signer count
	defaultOfficialFirstLevelCount  = 10   // official first level , 100% in signer queue
	defaultOfficialSecondLevelCount = 20   // official second level, 60% in signer queue
	defaultOfficialThirdLevelCount  = 30   // official third level, 40% in signer queue
	// the credit of one signer is at least minCalSignerQueueCredit
	candidateStateNormal = 1
	candidateMaxLen      = 500 // if candidateNeedPD is false and candidate is more than candidateMaxLen, then minimum tickets candidates will be remove in each LCRS*loop
)

var errIncorrectTallyCount = errors.New("incorrect tally count")

type Snapshot struct {
	config          *config.DposConfig    // Consensus engine configuration parameters
	sigcache        *lru.ARCCache         // Cache of recent block signatures to speed up ecrecover
	LCRS            uint64                // Loop count to recreate signers from top tally
	Period          uint64                `json:"period"`           // Period of seal each block
	Number          uint64                `json:"number"`           // Block Number where the snapshot was created
	ConfirmedNumber uint64                `json:"confirmed_number"` // Block Number confirmed when the snapshot was created
	Hash            bc.Hash               `json:"hash"`             // Block hash where the snapshot was created
	HistoryHash     []bc.Hash             `json:"historyHash"`      // Block hash list for two recent loop
	Signers         []*string             `json:"signers"`          // Signers queue in current header
	Votes           map[string]*Vote      `json:"votes"`            // All validate votes from genesis block
	Tally           map[string]uint64     `json:"tally"`            // Stake for each candidate address
	Voters          map[string]uint64     `json:"voters"`           // Block height for each voter address
	Candidates      map[string]uint64     `json:"candidates"`       // Candidates for Signers (0- adding procedure 1- normal 2- removing procedure)
	Punished        map[string]uint64     `json:"punished"`         // The signer be punished count cause of missing seal
	Confirmations   map[uint64][]string   `json:"confirms"`         // The signer confirm given block height
	Proposals       map[bc.Hash]*Proposal `json:"proposals"`        // The Proposals going or success (failed proposal will be removed)
	HeaderTime      uint64                `json:"headerTime"`       // Time of the current header
	LoopStartTime   uint64                `json:"loopStartTime"`    // Start Time of the current loop
}

// newSnapshot creates a new snapshot with the specified startup parameters. only ever use if for
// the genesis block.
func newSnapshot(config *config.DposConfig, sigcache *lru.ARCCache, hash bc.Hash, votes []*Vote, lcrs uint64) *Snapshot {

	snap := &Snapshot{
		config:          config,
		sigcache:        sigcache,
		LCRS:            lcrs,
		Period:          config.Period,
		Number:          0,
		ConfirmedNumber: 0,
		Hash:            hash,
		HistoryHash:     []bc.Hash{},
		Signers:         []*string{},
		Votes:           make(map[string]*Vote),
		Tally:           make(map[string]uint64),
		Voters:          make(map[string]uint64),
		Punished:        make(map[string]uint64),
		Candidates:      make(map[string]uint64),
		Confirmations:   make(map[uint64][]string),
		Proposals:       make(map[bc.Hash]*Proposal),
		HeaderTime:      uint64(time.Now().Unix()) - 1,
		LoopStartTime:   config.GenesisTimestamp,
	}
	snap.HistoryHash = append(snap.HistoryHash, hash)
	for _, vote := range votes {
		// init Votes from each vote
		snap.Votes[vote.Voter] = vote
		// init Tally
		_, ok := snap.Tally[vote.Candidate]
		if !ok {
			snap.Tally[vote.Candidate] = 0
		}
		snap.Tally[vote.Candidate] += vote.Stake
		// init Voters
		snap.Voters[vote.Voter] = 0 // block height is 0 , vote in genesis block
		// init Candidates
		snap.Candidates[vote.Voter] = candidateStateNormal
	}

	for i := 0; i < int(config.MaxSignerCount); i++ {
		snap.Signers = append(snap.Signers, &config.SelfVoteSigners[i%len(config.SelfVoteSigners)])
	}

	return snap
}

// loadSnapshot loads an existing snapshot from the database.
func loadSnapshot(config *config.DposConfig, sigcache *lru.ARCCache, store protocol.Store, hash bc.Hash) (*Snapshot, error) {
	data, err := store.Get(&hash)
	if err != nil {
		return nil, err
	}
	snap := new(Snapshot)
	if err := json.Unmarshal(data, snap); err != nil {
		return nil, err
	}
	snap.config = config
	snap.sigcache = sigcache
	return snap, nil
}

// store inserts the snapshot into the database.
func (s *Snapshot) store(store protocol.Store) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return store.Set(&s.Hash, data)
}

// copy creates a deep copy of the snapshot, though not the individual votes.
func (s *Snapshot) copy() *Snapshot {
	cpy := &Snapshot{
		config:          s.config,
		sigcache:        s.sigcache,
		LCRS:            s.LCRS,
		Period:          s.Period,
		Number:          s.Number,
		ConfirmedNumber: s.ConfirmedNumber,
		Hash:            s.Hash,
		HistoryHash:     make([]bc.Hash, len(s.HistoryHash)),

		Signers:       make([]*string, len(s.Signers)),
		Votes:         make(map[string]*Vote),
		Tally:         make(map[string]uint64),
		Voters:        make(map[string]uint64),
		Candidates:    make(map[string]uint64),
		Punished:      make(map[string]uint64),
		Proposals:     make(map[bc.Hash]*Proposal),
		Confirmations: make(map[uint64][]string),

		HeaderTime:    s.HeaderTime,
		LoopStartTime: s.LoopStartTime,
	}
	copy(cpy.HistoryHash, s.HistoryHash)
	copy(cpy.Signers, s.Signers)
	for voter, vote := range s.Votes {
		cpy.Votes[voter] = &Vote{
			Voter:     vote.Voter,
			Candidate: vote.Candidate,
			Stake:     vote.Stake,
		}
	}
	for candidate, tally := range s.Tally {
		cpy.Tally[candidate] = tally
	}
	for voter, number := range s.Voters {
		cpy.Voters[voter] = number
	}
	for candidate, state := range s.Candidates {
		cpy.Candidates[candidate] = state
	}
	for signer, cnt := range s.Punished {
		cpy.Punished[signer] = cnt
	}
	for blockNumber, confirmers := range s.Confirmations {
		cpy.Confirmations[blockNumber] = make([]string, len(confirmers))
		copy(cpy.Confirmations[blockNumber], confirmers)
	}
	for txHash, proposal := range s.Proposals {
		cpy.Proposals[txHash] = proposal.copy()
	}

	return cpy
}

// apply creates a new authorization snapshot by applying the given headers to
// the original one.
func (s *Snapshot) apply(headers []*types.BlockHeader) (*Snapshot, error) {
	// Allow passing in no headers for cleaner code
	if len(headers) == 0 {
		return s, nil
	}
	// Sanity check that the headers can be applied
	for i := 0; i < len(headers)-1; i++ {
		if headers[i+1].Height != headers[i].Height+1 {
			return nil, errInvalidVotingChain
		}
	}
	if headers[0].Height != s.Number+1 {
		return nil, errInvalidVotingChain
	}
	// Iterate through the headers and create a new snapshot
	snap := s.copy()
	for _, header := range headers {

		// Resolve the authorization key and check against signers
		coinbase, err := ecrecover(header, s.sigcache, nil)
		if err != nil {
			return nil, err
		}

		headerExtra := HeaderExtra{}
		if err := json.Unmarshal(header.Extra[extraVanity:len(header.Extra)-extraSeal], &headerExtra); err != nil {
			return nil, err
		}

		snap.HeaderTime = header.Timestamp
		snap.LoopStartTime = headerExtra.LoopStartTime
		snap.Signers = nil
		for i := range headerExtra.SignerQueue {
			snap.Signers = append(snap.Signers, &headerExtra.SignerQueue[i])
		}

		snap.ConfirmedNumber = headerExtra.ConfirmedBlockNumber

		if len(snap.HistoryHash) >= int(s.config.MaxSignerCount)*2 {
			snap.HistoryHash = snap.HistoryHash[1 : int(s.config.MaxSignerCount)*2]
		}

		snap.HistoryHash = append(snap.HistoryHash, header.Hash())

		// deal the new confirmation in this block
		snap.updateSnapshotByConfirmations(headerExtra.CurrentBlockConfirmations)

		// deal the new vote from voter
		snap.updateSnapshotByVotes(headerExtra.CurrentBlockVotes, header.Height)

		// deal the snap related with punished
		snap.updateSnapshotForPunish(headerExtra.SignerMissing, header.Height, coinbase)

		// deal proposals
		snap.updateSnapshotByProposals(headerExtra.CurrentBlockProposals, header.Height)

		// deal declares
		snap.updateSnapshotByDeclares(headerExtra.CurrentBlockDeclares, header.Height)

		// calculate proposal result
		snap.calculateProposalResult(header.Height)

		// check the len of candidate if not candidateNeedPD
		if !candidateNeedPD && (snap.Number+1)%(snap.config.MaxSignerCount*snap.LCRS) == 0 && len(snap.Candidates) > candidateMaxLen {
			snap.removeExtraCandidate()
		}

	}
	snap.Number += uint64(len(headers))
	snap.Hash = headers[len(headers)-1].Hash()
	snap.updateSnapshotForExpired()
	err := snap.verifyTallyCnt()
	if err != nil {
		return nil, err
	}
	return snap, nil
}

func (s *Snapshot) removeExtraCandidate() {
	// remove minimum tickets tally beyond candidateMaxLen
	tallySlice := s.buildTallySlice()
	sort.Sort(TallySlice(tallySlice))
	if len(tallySlice) > candidateMaxLen {
		removeNeedTally := tallySlice[candidateMaxLen:]
		for _, tallySlice := range removeNeedTally {
			delete(s.Candidates, tallySlice.addr)
		}
	}
}

func (s *Snapshot) verifyTallyCnt() error {

	tallyTarget := make(map[string]uint64)
	for _, v := range s.Votes {
		if _, ok := tallyTarget[v.Candidate]; ok {
			tallyTarget[v.Candidate] = tallyTarget[v.Candidate] + v.Stake
		} else {
			tallyTarget[v.Candidate] = v.Stake
		}
	}
	for address, tally := range s.Tally {
		if targetTally, ok := tallyTarget[address]; ok && targetTally == tally {
			continue
		} else {
			return errIncorrectTallyCount
		}
	}

	return nil
}

func (s *Snapshot) updateSnapshotByDeclares(declares []Declare, headerHeight uint64) {
	for _, declare := range declares {
		if proposal, ok := s.Proposals[declare.ProposalHash]; ok {
			// check the proposal enable status and valid block number
			if proposal.ReceivedNumber+proposal.ValidationLoopCnt*s.config.MaxSignerCount < headerHeight || !s.isCandidate(declare.Declarer) {
				continue
			}
			// check if this signer already declare on this proposal
			alreadyDeclare := false
			for _, v := range proposal.Declares {
				if v.Declarer == declare.Declarer {
					// this declarer already declare for this proposal
					alreadyDeclare = true
					break
				}
			}
			if alreadyDeclare {
				continue
			}
			// add declare to proposal
			s.Proposals[declare.ProposalHash].Declares = append(s.Proposals[declare.ProposalHash].Declares,
				&Declare{declare.ProposalHash, declare.Declarer, declare.Decision})

		}
	}
}

func (s *Snapshot) calculateProposalResult(headerHeight uint64) {

	for hashKey, proposal := range s.Proposals {
		// the result will be calculate at receiverdNumber + vlcnt + 1
		if proposal.ReceivedNumber+proposal.ValidationLoopCnt*s.config.MaxSignerCount+1 == headerHeight {
			// calculate the current stake of this proposal
			judegmentStake := big.NewInt(0)
			for _, tally := range s.Tally {
				judegmentStake.Add(judegmentStake, new(big.Int).SetUint64(tally))
			}
			judegmentStake.Mul(judegmentStake, big.NewInt(2))
			judegmentStake.Div(judegmentStake, big.NewInt(3))
			// calculate declare stake
			yesDeclareStake := big.NewInt(0)
			for _, declare := range proposal.Declares {
				if declare.Decision {
					if _, ok := s.Tally[declare.Declarer]; ok {
						yesDeclareStake.Add(yesDeclareStake, new(big.Int).SetUint64(s.Tally[declare.Declarer]))
					}
				}
			}
			if yesDeclareStake.Cmp(judegmentStake) > 0 {
				// process add candidate
				switch proposal.ProposalType {
				case proposalTypeCandidateAdd:
					if candidateNeedPD {
						s.Candidates[s.Proposals[hashKey].Candidate] = candidateStateNormal
					}
				case proposalTypeCandidateRemove:
					if _, ok := s.Candidates[proposal.Candidate]; ok && candidateNeedPD {
						delete(s.Candidates, proposal.Candidate)
					}
				case proposalTypeMinerRewardDistributionModify:
					minerRewardPerThousand = s.Proposals[hashKey].MinerRewardPerThousand
				}
			}
		}
	}
}

func (s *Snapshot) updateSnapshotByProposals(proposals []Proposal, headerHeight uint64) {
	for _, proposal := range proposals {
		proposal.ReceivedNumber = headerHeight
		s.Proposals[proposal.Hash] = &proposal
	}
}

func (s *Snapshot) updateSnapshotForExpired() {
	// deal the expired vote
	var expiredVotes []*Vote
	for voterAddress, voteNumber := range s.Voters {
		if s.Number-voteNumber > s.config.Epoch {
			// clear the vote
			if expiredVote, ok := s.Votes[voterAddress]; ok {
				expiredVotes = append(expiredVotes, expiredVote)
			}
		}
	}
	// remove expiredVotes only enough voters left
	if uint64(len(s.Voters)-len(expiredVotes)) >= s.config.MaxSignerCount {
		for _, expiredVote := range expiredVotes {
			s.Tally[expiredVote.Candidate] -= expiredVote.Stake
			// TODO
			if s.Tally[expiredVote.Candidate] == 0 {
				delete(s.Tally, expiredVote.Candidate)
			}
			delete(s.Votes, expiredVote.Voter)
			delete(s.Voters, expiredVote.Voter)
		}
	}

	// deal the expired confirmation
	for blockNumber := range s.Confirmations {
		if s.Number-blockNumber > s.config.MaxSignerCount {
			delete(s.Confirmations, blockNumber)
		}
	}

	// TODO
	// remove 0 stake tally

	for address, tally := range s.Tally {
		if tally <= 0 && uint64(len(s.Tally)) > s.config.MaxSignerCount {
			delete(s.Tally, address)
		}
	}
}

func (s *Snapshot) updateSnapshotByConfirmations(confirmations []Confirmation) {
	for _, confirmation := range confirmations {
		_, ok := s.Confirmations[confirmation.BlockNumber]
		if !ok {
			s.Confirmations[confirmation.BlockNumber] = []string{}
		}
		addConfirmation := true
		for _, address := range s.Confirmations[confirmation.BlockNumber] {
			if confirmation.Signer == address {
				addConfirmation = false
				break
			}
		}
		if addConfirmation == true {
			s.Confirmations[confirmation.BlockNumber] = append(s.Confirmations[confirmation.BlockNumber], confirmation.Signer)
		}
	}
}

func (s *Snapshot) updateSnapshotByVotes(votes []Vote, headerHeight uint64) {
	for _, vote := range votes {
		// update Votes, Tally, Voters data
		if lastVote, ok := s.Votes[vote.Voter]; ok {
			s.Tally[lastVote.Candidate] = s.Tally[lastVote.Candidate] - lastVote.Stake
		}
		if _, ok := s.Tally[vote.Candidate]; ok {
			s.Tally[vote.Candidate] = s.Tally[vote.Candidate] + vote.Stake
		} else {
			s.Tally[vote.Candidate] = vote.Stake
			if !candidateNeedPD {
				s.Candidates[vote.Candidate] = candidateStateNormal
			}
		}
		s.Votes[vote.Voter] = &Vote{vote.Voter, vote.Candidate, vote.Stake}
		s.Voters[vote.Voter] = headerHeight
	}
}

func (s *Snapshot) updateSnapshotByMPVotes(votes []Vote) {
	for _, txVote := range votes {
		if lastVote, ok := s.Votes[txVote.Voter]; ok {
			s.Tally[lastVote.Candidate] = s.Tally[lastVote.Candidate] - lastVote.Stake
			s.Tally[lastVote.Candidate] = s.Tally[lastVote.Candidate] + txVote.Stake
			s.Votes[txVote.Voter] = &Vote{Voter: txVote.Voter, Candidate: lastVote.Candidate, Stake: txVote.Stake}
		}
	}
}

func (s *Snapshot) updateSnapshotForPunish(signerMissing []string, headerNumber uint64, coinbase string) {
	// punish the missing signer
	for _, signerMissing := range signerMissing {
		if _, ok := s.Punished[signerMissing]; ok {
			s.Punished[signerMissing] += missingPublishCredit
		} else {
			s.Punished[signerMissing] = missingPublishCredit
		}
	}
	// reduce the punish of sign signer
	if _, ok := s.Punished[coinbase]; ok {

		if s.Punished[coinbase] > signRewardCredit {
			s.Punished[coinbase] -= signRewardCredit
		} else {
			delete(s.Punished, coinbase)
		}
	}
	// reduce the punish for all punished
	for signerEach := range s.Punished {
		if s.Punished[signerEach] > autoRewardCredit {
			s.Punished[signerEach] -= autoRewardCredit
		} else {
			delete(s.Punished, signerEach)
		}
	}
}

// inturn returns if a signer at a given block height is in-turn or not.
func (s *Snapshot) inturn(signer string, headerTime uint64) bool {
	// if all node stop more than period of one loop
	loopIndex := int((headerTime-s.LoopStartTime)/s.config.Period) % len(s.Signers)
	fmt.Println(loopIndex, signer)
	for _, v := range s.Signers {
		fmt.Println(*v)
	}
	if loopIndex >= len(s.Signers) {
		return false
	} else if *s.Signers[loopIndex] != signer {
		return false

	}
	return true
}

// check if address belong to voter
func (s *Snapshot) isVoter(address string) bool {
	if _, ok := s.Voters[address]; ok {
		return true
	}
	return false
}

// check if address belong to candidate
func (s *Snapshot) isCandidate(address string) bool {
	if _, ok := s.Candidates[address]; ok {
		return true
	}
	return false
}

// get last block number meet the confirm condition
func (s *Snapshot) getLastConfirmedBlockNumber(confirmations []Confirmation) *big.Int {
	cpyConfirmations := make(map[uint64][]string)
	for blockNumber, confirmers := range s.Confirmations {
		cpyConfirmations[blockNumber] = make([]string, len(confirmers))
		copy(cpyConfirmations[blockNumber], confirmers)
	}
	// update confirmation into snapshot
	for _, confirmation := range confirmations {
		_, ok := cpyConfirmations[confirmation.BlockNumber]
		if !ok {
			cpyConfirmations[confirmation.BlockNumber] = []string{}
		}
		addConfirmation := true
		for _, address := range cpyConfirmations[confirmation.BlockNumber] {
			if confirmation.Signer == address {
				addConfirmation = false
				break
			}
		}
		if addConfirmation == true {
			cpyConfirmations[confirmation.BlockNumber] = append(cpyConfirmations[confirmation.BlockNumber], confirmation.Signer)
		}
	}

	i := s.Number
	for ; i > s.Number-s.config.MaxSignerCount*2/3+1; i-- {
		if confirmers, ok := cpyConfirmations[i]; ok {
			if len(confirmers) > int(s.config.MaxSignerCount*2/3) {
				return big.NewInt(int64(i))
			}
		}
	}
	return big.NewInt(int64(i))
}
