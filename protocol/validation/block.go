package validation

import (
	"bytes"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/protocol/validation"
	"github.com/vapor/consensus"
	"github.com/vapor/errors"
	"github.com/vapor/math/checked"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

const logModule = "leveldb"

var (
	errBadTimestamp          = errors.New("block timestamp is not in the valid range")
	errBadBits               = errors.New("block bits is invalid")
	errMismatchedBlock       = errors.New("mismatched block")
	errMismatchedMerkleRoot  = errors.New("mismatched merkle root")
	errMisorderedBlockHeight = errors.New("misordered block height")
	errOverBlockLimit        = errors.New("block's gas is over the limit")
	errWorkProof             = errors.New("invalid difficulty proof of work")
	errVersionRegression     = errors.New("version regression")
)

func checkBlockTime(b *bc.Block, parent *types.BlockHeader) error {
	now := uint64(time.Now().UnixNano() / 1e6)
	if b.Timestamp < (parent.Timestamp + consensus.BlockTimeInterval) {
		return errBadTimestamp
	}
	if b.Timestamp > (now + consensus.MaxTimeOffsetMs) {
		return errBadTimestamp
	}

	return nil
}

func checkCoinbaseTx(b *bc.Block, rewards []CoinbaseReward) error {
	if len(b.Transactions) == 0 {
		return errors.Wrap(ErrWrongCoinbaseTransaction, "block is empty")
	}

	tx := b.Transactions[0]
	if len(tx.TxHeader.ResultIds) != len(rewards) {
		return errors.Wrapf(ErrWrongCoinbaseTransaction, "dismatch number of outputs, got:%d, want:%d", len(tx.TxHeader.ResultIds), len(rewards))
	}

	var coinbaseAmount uint64
	var coinbaseReceiver []byte
	for i, output := range tx.TxHeader.ResultIds {
		switch e := tx.Entries[*output].(type) {
		case *bc.IntraChainOutput:
			coinbaseAmount = e.Source.Value.Amount
			coinbaseReceiver = e.ControlProgram.Code
		default:
			return errors.Wrapf(bc.ErrEntryType, "entry %x has unexpected type %T", tx.TxHeader.ResultIds[0].Bytes(), output)
		}

		if i == 0 {
			if coinbaseAmount != 0 {
				return errors.Wrapf(ErrWrongCoinbaseTransaction, "dismatch output amount, got:%d, want:0", coinbaseAmount)
			}
		} else {
			if rewards[i].Amount != coinbaseAmount {
				return errors.Wrapf(ErrWrongCoinbaseTransaction, "dismatch output amount, got:%d, want:%d", coinbaseAmount, rewards[i].Amount)
			}
		}

		if res := bytes.Compare(rewards[i].ControlProgram, coinbaseReceiver); res != 0 {
			return errors.Wrapf(ErrWrongCoinbaseTransaction, "dismatch output control program, got:%v, want:%v", coinbaseReceiver, rewards[i].ControlProgram)
		}
	}
	return nil
}

// ValidateBlockHeader check the block's header
func ValidateBlockHeader(b *bc.Block, parent *types.BlockHeader) error {
	if b.Version != 1 {
		return errors.WithDetailf(errVersionRegression, "previous block verson %d, current block version %d", parent.Version, b.Version)
	}
	if b.Height != parent.Height+1 {
		return errors.WithDetailf(errMisorderedBlockHeight, "previous block height %d, current block height %d", parent.Height, b.Height)
	}
	if parent.Hash() != *b.PreviousBlockId {
		return errors.WithDetailf(errMismatchedBlock, "previous block ID %x, current block wants %x", parent.Hash().Bytes(), b.PreviousBlockId.Bytes())
	}

	return checkBlockTime(b, parent)
}

// ValidateBlock validates a block and the transactions within.
func ValidateBlock(b *bc.Block, parent *types.BlockHeader, rewards []CoinbaseReward) error {
	startTime := time.Now()
	if err := ValidateBlockHeader(b, parent); err != nil {
		return err
	}

	reward, err := CalCoinbaseReward(b)
	if err != nil {
		return err
	}

	if b.Height%consensus.RoundVoteBlockNums == 0 {
		aggregateFlag := false
		for i, r := range rewards {
			if res := bytes.Compare(r.ControlProgram, reward.ControlProgram); res == 0 {
				var ok bool
				if rewards[i].Amount, ok = checked.AddUint64(rewards[i].Amount, reward.Amount); !ok {
					return validation.ErrOverflow
				}
				aggregateFlag = true
				break
			}
		}

		if !aggregateFlag {
			rewards = append(rewards, *reward)
		}
		sort.Sort(SortByAmount(rewards))
	}
	rewards = append([]CoinbaseReward{CoinbaseReward{ControlProgram: reward.ControlProgram}}, rewards...)
	if err := checkCoinbaseTx(b, rewards); err != nil {
		return err
	}

	txMerkleRoot, err := types.TxMerkleRoot(b.Transactions)
	if err != nil {
		return errors.Wrap(err, "computing transaction id merkle root")
	}
	if txMerkleRoot != *b.TransactionsRoot {
		return errors.WithDetailf(errMismatchedMerkleRoot, "transaction id merkle root. compute: %v, given: %v", txMerkleRoot, *b.TransactionsRoot)
	}

	txStatusHash, err := types.TxStatusMerkleRoot(b.TransactionStatus.VerifyStatus)
	if err != nil {
		return errors.Wrap(err, "computing transaction status merkle root")
	}
	if txStatusHash != *b.TransactionStatusHash {
		return errors.WithDetailf(errMismatchedMerkleRoot, "transaction status merkle root. compute: %v, given: %v", txStatusHash, *b.TransactionStatusHash)
	}

	log.WithFields(log.Fields{
		"module":   logModule,
		"height":   b.Height,
		"hash":     b.ID.String(),
		"duration": time.Since(startTime),
	}).Debug("finish validate block")
	return nil
}

// CoinbaseReward contains receiver and reward
type CoinbaseReward struct {
	Amount         uint64
	ControlProgram []byte
}

// SortByAmount implements sort.Interface for CoinbaseReward slices
type SortByAmount []CoinbaseReward

func (a SortByAmount) Len() int           { return len(a) }
func (a SortByAmount) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SortByAmount) Less(i, j int) bool { return a[i].Amount < a[j].Amount }

// CalCoinbaseReward calculate the coinbase reward for block
func CalCoinbaseReward(b *bc.Block) (*CoinbaseReward, error) {
	if len(b.Transactions) == 0 {
		return nil, errors.Wrap(ErrWrongCoinbaseTransaction, "block is empty")
	}

	tx := b.Transactions[0]
	if len(tx.TxHeader.ResultIds) == 0 {
		return nil, errors.Wrap(ErrWrongCoinbaseTransaction, "without output")
	}

	var coinbaseReceiver []byte
	switch e := tx.Entries[*tx.TxHeader.ResultIds[0]].(type) {
	case *bc.IntraChainOutput:
		coinbaseReceiver = e.ControlProgram.Code
	default:
		return nil, errors.Wrapf(bc.ErrEntryType, "entry %x has unexpected type %T", tx.TxHeader.ResultIds[0].Bytes(), e)
	}

	if coinbaseReceiver == nil {
		return nil, errors.New("not found the zero coinbase output")
	}

	blockGasSum := uint64(0)
	coinbaseAmount := consensus.BlockSubsidy(b.BlockHeader.Height)
	b.TransactionStatus = bc.NewTransactionStatus()

	validateResults := ValidateTxs(b.Transactions, b)
	for i, validateResult := range validateResults {
		if !validateResult.gasStatus.GasValid {
			return nil, errors.Wrapf(validateResult.err, "validate of transaction %d of %d", i, len(b.Transactions))
		}

		if err := b.TransactionStatus.SetStatus(i, validateResult.err != nil); err != nil {
			return nil, err
		}
		coinbaseAmount += validateResult.gasStatus.BTMValue
		if blockGasSum += uint64(validateResult.gasStatus.GasUsed); blockGasSum > consensus.MaxBlockGas {
			return nil, errOverBlockLimit
		}
	}
	return &CoinbaseReward{
		Amount:         coinbaseAmount,
		ControlProgram: coinbaseReceiver,
	}, nil
}
