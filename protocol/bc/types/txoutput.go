package types

import (
	"fmt"
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
		// Unconsumed suffixes of the commitment and witness extensible strings.
		CommitmentSuffix []byte
	}

	// TypedOutput return the txoutput type.
	TypedOutput interface {
		OutputType() uint8
	}
)

// AssetAmount return the asset id and amount of a txoutput.
func (to *TxOutput) AssetAmount() bc.AssetAmount {
	switch outp := to.TypedOutput.(type) {
	case *IntraChainOutput:
		return outp.AssetAmount

	case *CrossChainOutput:
		return outp.AssetAmount

	default:
		return bc.AssetAmount{}
	}
}

// ControlProgram return the control program of the txoutput
func (to *TxOutput) ControlProgram() []byte {
	switch outp := to.TypedOutput.(type) {
	case *IntraChainOutput:
		return outp.ControlProgram

	case *CrossChainOutput:
		return outp.ControlProgram

	default:
		return nil
	}
}

// VMVersion return the VM version of the txoutput
func (to *TxOutput) VMVersion() uint64 {
	switch outp := to.TypedOutput.(type) {
	case *IntraChainOutput:
		return outp.VMVersion

	case *CrossChainOutput:
		return outp.VMVersion

	default:
		return 0
	}
}

// TODO: OutputType
func (to *TxOutput) readFrom(r *blockchain.Reader) (err error) {
	if to.AssetVersion, err = blockchain.ReadVarint63(r); err != nil {
		return errors.Wrap(err, "reading asset version")
	}

	to.CommitmentSuffix, err = blockchain.ReadExtensibleString(r, func(r *blockchain.Reader) error {
		if to.AssetVersion != currentAssetVersion {
			return nil
		}

		var outType [1]byte
		if _, err = io.ReadFull(r, outType[:]); err != nil {
			return errors.Wrap(err, "reading output type")
		}

		switch outType[0] {
		case IntraChainOutputType:
			out := new(IntraChainOutput)
			to.TypedOutput = out
			if out.CommitmentSuffix, err = out.OutputCommitment.readFrom(r, to.AssetVersion); err != nil {
				return errors.Wrap(err, "reading intra-chain output commitment")
			}

		case CrossChainOutputType:
			out := new(CrossChainOutput)
			to.TypedOutput = out
			if out.CommitmentSuffix, err = out.OutputCommitment.readFrom(r, to.AssetVersion); err != nil {
				return errors.Wrap(err, "reading cross-chain output commitment")
			}

		default:
			return fmt.Errorf("unsupported output type %d", outType[0])
		}

		return nil
	})

	if err != nil {
		return err
	}

	// read and ignore the (empty) output witness
	_, err = blockchain.ReadVarstr31(r)
	return errors.Wrap(err, "reading output witness")
}

// TODO: OutputType
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
	if to.AssetVersion != currentAssetVersion {
		return nil
	}

	switch outp := to.TypedOutput.(type) {
	case *IntraChainOutput:
		return outp.OutputCommitment.writeExtensibleString(w, outp.CommitmentSuffix, to.AssetVersion)

	case *CrossChainOutput:
		return outp.OutputCommitment.writeExtensibleString(w, outp.CommitmentSuffix, to.AssetVersion)

	default:
		return nil
	}
}

// TODO:
// ComputeOutputID assembles an output entry given a spend commitment and
// computes and returns its corresponding entry ID.
func ComputeOutputID(sc *SpendCommitment) (h bc.Hash, err error) {
	defer func() {
		if r, ok := recover().(error); ok {
			err = r
		}
	}()
	src := &bc.ValueSource{
		Ref:      &sc.SourceID,
		Value:    &sc.AssetAmount,
		Position: sc.SourcePosition,
	}
	o := bc.NewIntraChainOutput(src, &bc.Program{VmVersion: sc.VMVersion, Code: sc.ControlProgram}, 0)

	h = bc.EntryID(o)
	return h, nil
}
