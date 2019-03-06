package test

import (
	"github.com/vapor/consensus"
	"github.com/vapor/crypto"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/validation"
	"github.com/vapor/protocol/vm"
	"github.com/vapor/protocol/vm/vmutil"
)

// NewBlock create block according to the current status of chain
func NewBlock(chain *protocol.Chain, txs []*types.Tx, controlProgram []byte) (*types.Block, error) {
	gasUsed := uint64(0)
	txsFee := uint64(0)
	txEntries := []*bc.Tx{nil}
	txStatus := bc.NewTransactionStatus()
	if err := txStatus.SetStatus(0, false); err != nil {
		return nil, err
	}

	preBlockHeader := chain.BestBlockHeader()

	b := &types.Block{
		BlockHeader: types.BlockHeader{
			Version:           1,
			Height:            preBlockHeader.Height + 1,
			PreviousBlockHash: preBlockHeader.Hash(),
			Timestamp:         preBlockHeader.Timestamp + 1,
			BlockCommitment:   types.BlockCommitment{},
		},
		Transactions: []*types.Tx{nil},
	}

	bcBlock := &bc.Block{BlockHeader: &bc.BlockHeader{Height: preBlockHeader.Height + 1}}
	for _, tx := range txs {
		gasOnlyTx := false
		gasStatus, err := validation.ValidateTx(tx.Tx, bcBlock)
		if err != nil {
			if !gasStatus.GasValid {
				continue
			}
			gasOnlyTx = true
		}

		txStatus.SetStatus(len(b.Transactions), gasOnlyTx)
		b.Transactions = append(b.Transactions, tx)
		txEntries = append(txEntries, tx.Tx)
		gasUsed += uint64(gasStatus.GasUsed)
		txsFee += txFee(tx)
	}

	coinbaseTx, err := CreateCoinbaseTx(controlProgram, preBlockHeader.Height+1, txsFee)
	if err != nil {
		return nil, err
	}

	b.Transactions[0] = coinbaseTx
	txEntries[0] = coinbaseTx.Tx
	b.TransactionsMerkleRoot, err = types.TxMerkleRoot(txEntries)
	if err != nil {
		return nil, err
	}

	b.TransactionStatusHash, err = types.TxStatusMerkleRoot(txStatus.VerifyStatus)
	proof, err := generateProof(*b)
	if err != nil {
		return nil, err
	}
	b.Proof = proof
	return b, err
}

func generateProof(block types.Block) (types.Proof, error) {
	var xPrv chainkd.XPrv
	if consensus.ActiveNetParams.Signer == "" {
		return types.Proof{}, errors.New("Signer is empty")
	}
	xPrv.UnmarshalText([]byte(consensus.ActiveNetParams.Signer))
	sign := xPrv.Sign(block.BlockCommitment.TransactionsMerkleRoot.Bytes())
	pubHash := crypto.Ripemd160(xPrv.XPub().PublicKey())
	control, err := vmutil.P2WPKHProgram([]byte(pubHash))
	if err != nil {
		return types.Proof{}, err
	}
	return types.Proof{Sign: sign, ControlProgram: control}, nil
}

// ReplaceCoinbase replace the coinbase tx of block with coinbaseTx
func ReplaceCoinbase(block *types.Block, coinbaseTx *types.Tx) (err error) {
	block.Transactions[0] = coinbaseTx
	txEntires := []*bc.Tx{coinbaseTx.Tx}
	for i := 1; i < len(block.Transactions); i++ {
		txEntires = append(txEntires, block.Transactions[i].Tx)
	}

	block.TransactionsMerkleRoot, err = types.TxMerkleRoot(txEntires)
	return
}

// AppendBlocks append empty blocks to chain, mainly used to mature the coinbase tx
func AppendBlocks(chain *protocol.Chain, num uint64) error {
	for i := uint64(0); i < num; i++ {
		block, err := NewBlock(chain, nil, []byte{byte(vm.OP_TRUE)})
		if err != nil {
			return err
		}
		if err := SolveAndUpdate(chain, block); err != nil {
			return err
		}
	}
	return nil
}

// SolveAndUpdate solve difficulty and update chain status
func SolveAndUpdate(chain *protocol.Chain, block *types.Block) error {
	_, err := chain.ProcessBlock(block)
	return err
}
