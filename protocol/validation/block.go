package validation

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/consensus"
	"github.com/vapor/errors"
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

func checkCoinbaseAmount(b *bc.Block, amount uint64) error {
	if len(b.Transactions) == 0 {
		return errors.Wrap(ErrWrongCoinbaseTransaction, "block is empty")
	}

	tx := b.Transactions[0]
	if len(tx.TxHeader.ResultIds) != 1 {
		return errors.Wrap(ErrWrongCoinbaseTransaction, "have more than 1 output")
	}

	var SourceAmount uint64
	switch output := tx.Entries[*tx.TxHeader.ResultIds[0]].(type) {
	case *bc.IntraChainOutput:
		SourceAmount = output.Source.Value.Amount
	case *bc.VoteOutput:
		SourceAmount = output.Source.Value.Amount
	default:
		return errors.Wrapf(bc.ErrEntryType, "entry %x has unexpected type %T", tx.TxHeader.ResultIds[0].Bytes(), output)
	}

	if SourceAmount != amount {
		return errors.Wrap(ErrWrongCoinbaseTransaction, "dismatch output amount")
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
func ValidateBlock(b *bc.Block, parent *types.BlockHeader) error {
	startTime := time.Now()
	if err := ValidateBlockHeader(b, parent); err != nil {
		return err
	}

	blockGasSum := uint64(0)
	coinbaseAmount := consensus.BlockSubsidy(b.BlockHeader.Height)
	b.TransactionStatus = bc.NewTransactionStatus()

	validateResults := ValidateTxs(b.Transactions, b)
	for i, validateResult := range validateResults {
		if !validateResult.gasStatus.GasValid {
			return errors.Wrapf(validateResult.err, "validate of transaction %d of %d", i, len(b.Transactions))
		}

		if err := b.TransactionStatus.SetStatus(i, validateResult.err != nil); err != nil {
			return err
		}
		coinbaseAmount += validateResult.gasStatus.BTMValue
		if blockGasSum += uint64(validateResult.gasStatus.GasUsed); blockGasSum > consensus.MaxBlockGas {
			return errOverBlockLimit
		}
	}

	if err := checkCoinbaseAmount(b, coinbaseAmount); err != nil {
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
		return errors.WithDetailf(errMismatchedMerkleRoot, "transaction status merkle root")
	}

	log.WithFields(log.Fields{
		"module":   logModule,
		"height":   b.Height,
		"hash":     b.ID.String(),
		"duration": time.Since(startTime),
	}).Debug("finish validate block")
	return nil
}
