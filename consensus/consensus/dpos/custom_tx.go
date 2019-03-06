package dpos

import (
	"encoding/json"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/vapor/chain"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const (
	/*
	 *  vapor:version:category:action/data
	 */
	vaporPrefix        = "vapor"
	vaporVersion       = "1"
	vaporCategoryEvent = "event"
	vaporCategoryLog   = "oplog"
	vaporCategorySC    = "sc"
	vaporEventVote     = "vote"
	vaporEventConfirm  = "confirm"
	vaporEventPorposal = "proposal"
	vaporEventDeclare  = "declare"

	vaporMinSplitLen      = 3
	posPrefix             = 0
	posVersion            = 1
	posCategory           = 2
	posEventVote          = 3
	posEventConfirm       = 3
	posEventProposal      = 3
	posEventDeclare       = 3
	posEventConfirmNumber = 4

	/*
	 *  proposal type
	 */
	proposalTypeCandidateAdd                  = 1
	proposalTypeCandidateRemove               = 2
	proposalTypeMinerRewardDistributionModify = 3 // count in one thousand

	/*
	 * proposal related
	 */
	maxValidationLoopCnt     = 123500 // About one month if seal each block per second & 21 super nodes
	minValidationLoopCnt     = 12350  // About three days if seal each block per second & 21 super nodes
	defaultValidationLoopCnt = 30875  // About one week if seal each block per second & 21 super nodes
)

// Vote :
// vote come from custom tx which data like "vapor:1:event:vote"
// Sender of tx is Voter, the tx.to is Candidate
// Stake is the balance of Voter when create this vote
type Vote struct {
	Voter     string `json:"Voter"`
	Candidate string `json:"Candidate"`
	Stake     uint64 `json:"Stake"`
}

// Confirmation :
// confirmation come  from custom tx which data like "vapor:1:event:confirm:123"
// 123 is the block number be confirmed
// Sender of tx is Signer only if the signer in the SignerQueue for block number 123
type Confirmation struct {
	Signer      string `json:"signer"`
	BlockNumber uint64 `json:"block_number"`
}

// Proposal :
// proposal come from  custom tx which data like "vapor:1:event:proposal:candidate:add:address" or "vapor:1:event:proposal:percentage:60"
// proposal only come from the current candidates
// not only candidate add/remove , current signer can proposal for params modify like percentage of reward distribution ...
type Proposal struct {
	Hash                   bc.Hash    `json:"hash"`              // tx hash
	ValidationLoopCnt      uint64     `json:"ValidationLoopCnt"` // validation block number length of this proposal from the received block number
	ImplementNumber        uint64     `json:"ImplementNumber"`   // block number to implement modification in this proposal
	ProposalType           uint64     `json:"ProposalType"`      // type of proposal 1 - add candidate 2 - remove candidate ...
	Proposer               string     `json:"Proposer"`          //
	Candidate              string     `json:"Candidate"`
	MinerRewardPerThousand uint64     `json:"MinerRewardPerThousand"`
	Declares               []*Declare `json:"Declares"`       // Declare this proposal received
	ReceivedNumber         uint64     `json:"ReceivedNumber"` // block number of proposal received
}

func (p *Proposal) copy() *Proposal {
	cpy := &Proposal{
		Hash:                   p.Hash,
		ValidationLoopCnt:      p.ValidationLoopCnt,
		ImplementNumber:        p.ImplementNumber,
		ProposalType:           p.ProposalType,
		Proposer:               p.Proposer,
		Candidate:              p.Candidate,
		MinerRewardPerThousand: p.MinerRewardPerThousand,
		Declares:               make([]*Declare, len(p.Declares)),
		ReceivedNumber:         p.ReceivedNumber,
	}

	copy(cpy.Declares, p.Declares)
	return cpy
}

// Declare :
// declare come from custom tx which data like "vapor:1:event:declare:hash:yes"
// proposal only come from the current candidates
// hash is the hash of proposal tx
type Declare struct {
	ProposalHash bc.Hash `json:"ProposalHash"`
	Declarer     string  `json:"Declarer"`
	Decision     bool    `json:"Decision"`
}

// HeaderExtra is the struct of info in header.Extra[extraVanity:len(header.extra)-extraSeal]
type HeaderExtra struct {
	CurrentBlockConfirmations []Confirmation `json:"current_block_confirmations"`
	CurrentBlockVotes         []Vote         `json:"CurrentBlockVotes"`
	CurrentBlockProposals     []Proposal     `json:"CurrentBlockProposals"`
	CurrentBlockDeclares      []Declare      `json:"CurrentBlockDeclares"`
	ModifyPredecessorVotes    []Vote         `json:"ModifyPredecessorVotes"`
	LoopStartTime             uint64         `json:"LoopStartTime"`
	SignerQueue               []string       `json:"SignerQueue"`
	SignerMissing             []string       `json:"SignerMissing"`
	ConfirmedBlockNumber      uint64         `json:"ConfirmedBlockNumber"`
}

// Calculate Votes from transaction in this block, write into header.Extra
func (d *Dpos) processCustomTx(headerExtra HeaderExtra, c chain.Chain, header *types.BlockHeader, txs []*bc.Tx) (HeaderExtra, error) {

	var (
		snap   *Snapshot
		err    error
		height uint64
	)
	height = header.Height
	if height > 1 {
		snap, err = d.snapshot(c, height-1, header.PreviousBlockHash, nil, nil, defaultLoopCntRecalculateSigners)
		if err != nil {
			return headerExtra, err
		}
	}

	for _, tx := range txs {
		var (
			from string
			to   string
		)
		dpos := new(bc.Dpos)
		stake := uint64(0)
		for _, value := range tx.Entries {
			switch d := value.(type) {
			case *bc.Dpos:
				from = d.From
				to = d.To
				dpos = d
				stake = d.Stake
			default:
				continue
			}

			if len(dpos.Data) >= len(vaporPrefix) {
				txData := dpos.Data
				txDataInfo := strings.Split(txData, ":")
				if len(txDataInfo) >= vaporMinSplitLen && txDataInfo[posPrefix] == vaporPrefix && txDataInfo[posVersion] == vaporVersion {
					switch txDataInfo[posCategory] {
					case vaporCategoryEvent:
						if len(txDataInfo) > vaporMinSplitLen {
							if txDataInfo[posEventVote] == vaporEventVote && (!candidateNeedPD || snap.isCandidate(to)) {
								headerExtra.CurrentBlockVotes = d.processEventVote(headerExtra.CurrentBlockVotes, stake, from, to)
							} else if txDataInfo[posEventConfirm] == vaporEventConfirm {
								headerExtra.CurrentBlockConfirmations = d.processEventConfirm(headerExtra.CurrentBlockConfirmations, c, txDataInfo, height, tx, from)
							} else if txDataInfo[posEventProposal] == vaporEventPorposal && snap.isCandidate(from) {
								headerExtra.CurrentBlockProposals = d.processEventProposal(headerExtra.CurrentBlockProposals, txDataInfo, tx, from)
							} else if txDataInfo[posEventDeclare] == vaporEventDeclare && snap.isCandidate(from) {
								headerExtra.CurrentBlockDeclares = d.processEventDeclare(headerExtra.CurrentBlockDeclares, txDataInfo, tx, from)

							}
						} else {
							// todo : something wrong, leave this transaction to process as normal transaction
						}

					case vaporCategoryLog:
						// todo :
					case vaporCategorySC:
						// todo :
					}
				}
			}
		}
	}

	return headerExtra, nil
}

func (d *Dpos) processEventProposal(currentBlockProposals []Proposal, txDataInfo []string, tx *bc.Tx, proposer string) []Proposal {
	proposal := Proposal{
		Hash:                   tx.ID,
		ValidationLoopCnt:      defaultValidationLoopCnt,
		ImplementNumber:        uint64(1),
		ProposalType:           proposalTypeCandidateAdd,
		Proposer:               proposer,
		MinerRewardPerThousand: minerRewardPerThousand,
		Declares:               []*Declare{},
		ReceivedNumber:         uint64(0),
	}

	for i := 0; i < len(txDataInfo[posEventProposal+1:])/2; i++ {
		k, v := txDataInfo[posEventProposal+1+i*2], txDataInfo[posEventProposal+2+i*2]
		switch k {
		case "vlcnt":
			// If vlcnt is missing then user default value, but if the vlcnt is beyond the min/max value then ignore this proposal
			if validationLoopCnt, err := strconv.Atoi(v); err != nil || validationLoopCnt < minValidationLoopCnt || validationLoopCnt > maxValidationLoopCnt {
				return currentBlockProposals
			} else {
				proposal.ValidationLoopCnt = uint64(validationLoopCnt)
			}
		case "implement_number":
			if implementNumber, err := strconv.Atoi(v); err != nil || implementNumber <= 0 {
				return currentBlockProposals
			} else {
				proposal.ImplementNumber = uint64(implementNumber)
			}
		case "proposal_type":
			if proposalType, err := strconv.Atoi(v); err != nil || (proposalType != proposalTypeCandidateAdd && proposalType != proposalTypeCandidateRemove && proposalType != proposalTypeMinerRewardDistributionModify) {
				return currentBlockProposals
			} else {
				proposal.ProposalType = uint64(proposalType)
			}
		case "candidate":
			// not check here
			//proposal.Candidate.UnmarshalText([]byte(v))
			/*
				address, err := common.DecodeAddress(v, &consensus.ActiveNetParams)
				if err != nil {
					return currentBlockProposals
				}
			*/
			proposal.Candidate = v
		case "mrpt":
			// miner reward per thousand
			if mrpt, err := strconv.Atoi(v); err != nil || mrpt < 0 || mrpt > 1000 {
				return currentBlockProposals
			} else {
				proposal.MinerRewardPerThousand = uint64(mrpt)
			}

		}
	}

	return append(currentBlockProposals, proposal)
}

func (d *Dpos) processEventDeclare(currentBlockDeclares []Declare, txDataInfo []string, tx *bc.Tx, declarer string) []Declare {
	declare := Declare{
		ProposalHash: bc.Hash{},
		Declarer:     declarer,
		Decision:     true,
	}

	for i := 0; i < len(txDataInfo[posEventDeclare+1:])/2; i++ {
		k, v := txDataInfo[posEventDeclare+1+i*2], txDataInfo[posEventDeclare+2+i*2]
		switch k {
		case "hash":
			declare.ProposalHash.UnmarshalText([]byte(v))
		case "decision":
			if v == "yes" {
				declare.Decision = true
			} else if v == "no" {
				declare.Decision = false
			} else {
				return currentBlockDeclares
			}
		}
	}

	return append(currentBlockDeclares, declare)
}

func (d *Dpos) processEventVote(currentBlockVotes []Vote, stake uint64, voter, to string) []Vote {
	if stake >= d.config.MinVoterBalance {
		currentBlockVotes = append(currentBlockVotes, Vote{
			Voter:     voter,
			Candidate: to,
			Stake:     stake,
		})
	}
	return currentBlockVotes
}

func (d *Dpos) processEventConfirm(currentBlockConfirmations []Confirmation, c chain.Chain, txDataInfo []string, number uint64, tx *bc.Tx, confirmer string) []Confirmation {
	if len(txDataInfo) > posEventConfirmNumber {
		confirmedBlockNumber, err := strconv.Atoi(txDataInfo[posEventConfirmNumber])
		if err != nil || number-uint64(confirmedBlockNumber) > d.config.MaxSignerCount || number-uint64(confirmedBlockNumber) < 0 {
			return currentBlockConfirmations
		}
		confirmedHeader, err := c.GetBlockByHeight(uint64(confirmedBlockNumber))
		if confirmedHeader == nil {
			log.Info("Fail to get confirmedHeader")
			return currentBlockConfirmations
		}
		confirmedHeaderExtra := HeaderExtra{}
		if extraVanity+extraSeal > len(confirmedHeader.Extra) {
			return currentBlockConfirmations
		}
		if err := json.Unmarshal(confirmedHeader.Extra[extraVanity:len(confirmedHeader.Extra)-extraSeal], &confirmedHeaderExtra); err != nil {
			log.Info("Fail to decode parent header", "err", err)
			return currentBlockConfirmations
		}
		for _, s := range confirmedHeaderExtra.SignerQueue {
			if s == confirmer {
				currentBlockConfirmations = append(currentBlockConfirmations, Confirmation{
					Signer:      confirmer,
					BlockNumber: uint64(confirmedBlockNumber),
				})
				break
			}
		}
	}

	return currentBlockConfirmations
}
