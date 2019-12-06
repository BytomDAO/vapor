package validation

import (
	"bytes"
	"encoding/hex"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
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
	if b.Timestamp < (parent.Timestamp + consensus.ActiveNetParams.BlockTimeInterval) {
		return errBadTimestamp
	}
	if b.Timestamp > (now + consensus.ActiveNetParams.MaxTimeOffsetMs) {
		return errBadTimestamp
	}

	return nil
}

func checkCoinbaseTx(b *bc.Block, rewards []state.CoinbaseReward) error {
	if len(b.Transactions) == 0 {
		return errors.Wrap(ErrWrongCoinbaseTransaction, "block is empty")
	}

	tx := b.Transactions[0]
	if len(tx.TxHeader.ResultIds) != len(rewards)+1 {
		return errors.Wrapf(ErrWrongCoinbaseTransaction, "dismatch number of outputs, got:%d, want:%d", len(tx.TxHeader.ResultIds), len(rewards))
	}

	rewards = append([]state.CoinbaseReward{state.CoinbaseReward{Amount: uint64(0)}}, rewards...)
	for i, output := range tx.TxHeader.ResultIds {
		out, err := tx.IntraChainOutput(*output)
		if err != nil {
			return err
		}

		if rewards[i].Amount != out.Source.Value.Amount {
			return errors.Wrapf(ErrWrongCoinbaseTransaction, "dismatch output amount, got:%d, want:%d", out.Source.Value.Amount, rewards[i].Amount)
		}

		if i == 0 {
			continue
		}

		if res := bytes.Compare(rewards[i].ControlProgram, out.ControlProgram.Code); res != 0 {
			return errors.Wrapf(ErrWrongCoinbaseTransaction, "dismatch output control_program, got:%s, want:%s", hex.EncodeToString(out.ControlProgram.Code), hex.EncodeToString(rewards[i].ControlProgram))
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
func ValidateBlock(b *bc.Block, parent *types.BlockHeader, rewards []state.CoinbaseReward) error {
	startTime := time.Now()
	if err := ValidateBlockHeader(b, parent); err != nil {
		return err
	}

	blockGasSum := uint64(0)
	b.TransactionStatus = bc.NewTransactionStatus()
	validateResults := ValidateTxs(b.Transactions, b)
	for i, validateResult := range validateResults {
		if !validateResult.gasStatus.GasValid {
			return errors.Wrapf(validateResult.err, "validate of transaction %d of %d", i, len(b.Transactions))
		}

		if err := b.TransactionStatus.SetStatus(i, validateResult.err != nil); err != nil {
			return err
		}

		if blockGasSum += uint64(validateResult.gasStatus.GasUsed); blockGasSum > consensus.ActiveNetParams.MaxBlockGas {
			return errOverBlockLimit
		}
	}

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
