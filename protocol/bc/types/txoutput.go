package types

import (
	"io"

	"github.com/vapor/encoding/blockchain"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
)

// serflag variables for output types.
const (
	IntraChainOutputType uint8 = iota
	CrossChainOutputType
)

type (
	// TxOutput is the top level struct of tx output.
	TxOutput struct {
		AssetVersion uint64
		TypedOutput
		OutputCommitment
		// Unconsumed suffixes of the commitment and witness extensible strings.
		CommitmentSuffix []byte
	}

	// TypedOutput return the txoutput type.
	TypedOutput interface {
		OutputType() uint8
	}
)

// NewTxOutput create a new output struct
func NewTxOutput(assetID bc.AssetID, amount uint64, controlProgram []byte) *TxOutput {
	return &TxOutput{
		AssetVersion: 1,
		OutputCommitment: OutputCommitment{
			AssetAmount: bc.AssetAmount{
				AssetId: &assetID,
				Amount:  amount,
			},
			VMVersion:      1,
			ControlProgram: controlProgram,
		},
	}
}

func (to *TxOutput) readFrom(r *blockchain.Reader) (err error) {
	if to.AssetVersion, err = blockchain.ReadVarint63(r); err != nil {
		return errors.Wrap(err, "reading asset version")
	}

	if to.CommitmentSuffix, err = to.OutputCommitment.readFrom(r, to.AssetVersion); err != nil {
		return errors.Wrap(err, "reading output commitment")
	}

	// read and ignore the (empty) output witness
	_, err = blockchain.ReadVarstr31(r)
	return errors.Wrap(err, "reading output witness")
}

func (to *TxOutput) writeTo(w io.Writer) error {
	if _, err := blockchain.WriteVarint63(w, to.AssetVersion); err != nil {
		return errors.Wrap(err, "writing asset version")
	}

	if err := to.writeCommitment(w); err != nil {
		return errors.Wrap(err, "writing output commitment")
	}

	if _, err := blockchain.WriteVarstr31(w, nil); err != nil {
		return errors.Wrap(err, "writing witness")
	}
	return nil
}

func (to *TxOutput) writeCommitment(w io.Writer) error {
	return to.OutputCommitment.writeExtensibleString(w, to.CommitmentSuffix, to.AssetVersion)
}
