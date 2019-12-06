package types

import (
	"fmt"
	"io"

	"github.com/bytom/vapor/encoding/blockchain"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
)

// serflag variables for output types.
const (
	IntraChainOutputType uint8 = iota
	CrossChainOutputType
	VoteOutputType
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

// OutputCommitment return the OutputCommitment of a txoutput.
func (to *TxOutput) OutputCommitment() OutputCommitment {
	switch outp := to.TypedOutput.(type) {
	case *IntraChainOutput:
		return outp.OutputCommitment

	case *CrossChainOutput:
		return outp.OutputCommitment

	case *VoteOutput:
		return outp.OutputCommitment

	default:
		return OutputCommitment{}
	}
}

// AssetAmount return the asset id and amount of a txoutput.
func (to *TxOutput) AssetAmount() bc.AssetAmount {
	switch outp := to.TypedOutput.(type) {
	case *IntraChainOutput:
		return outp.AssetAmount

	case *CrossChainOutput:
		return outp.AssetAmount

	case *VoteOutput:
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

	case *VoteOutput:
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

	case *VoteOutput:
		return outp.VMVersion

	default:
		return 0
	}
}

func (to *TxOutput) readFrom(r *blockchain.Reader) (err error) {
	if to.AssetVersion, err = blockchain.ReadVarint63(r); err != nil {
		return errors.Wrap(err, "reading asset version")
	}

	to.CommitmentSuffix, err = blockchain.ReadExtensibleString(r, func(r *blockchain.Reader) error {
		if to.AssetVersion != 1 {
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

		case VoteOutputType:
			out := new(VoteOutput)
			to.TypedOutput = out
			if out.Vote, err = blockchain.ReadVarstr31(r); err != nil {
				return errors.Wrap(err, "reading vote output vote")
			}

			if out.CommitmentSuffix, err = out.OutputCommitment.readFrom(r, to.AssetVersion); err != nil {
				return errors.Wrap(err, "reading vote output commitment")
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

func (to *TxOutput) writeTo(w io.Writer) error {
	if _, err := blockchain.WriteVarint63(w, to.AssetVersion); err != nil {
		return errors.Wrap(err, "writing asset version")
	}

	if _, err := blockchain.WriteExtensibleString(w, to.CommitmentSuffix, to.writeOutputCommitment); err != nil {
		return errors.Wrap(err, "writing output commitment")
	}

	if _, err := blockchain.WriteVarstr31(w, nil); err != nil {
		return errors.Wrap(err, "writing witness")
	}

	return nil
}

func (to *TxOutput) writeOutputCommitment(w io.Writer) error {
	if to.AssetVersion != 1 {
		return nil
	}

	switch outp := to.TypedOutput.(type) {
	case *IntraChainOutput:
		if _, err := w.Write([]byte{IntraChainOutputType}); err != nil {
			return err
		}
		return outp.OutputCommitment.writeExtensibleString(w, outp.CommitmentSuffix, to.AssetVersion)

	case *CrossChainOutput:
		if _, err := w.Write([]byte{CrossChainOutputType}); err != nil {
			return err
		}
		return outp.OutputCommitment.writeExtensibleString(w, outp.CommitmentSuffix, to.AssetVersion)

	case *VoteOutput:
		if _, err := w.Write([]byte{VoteOutputType}); err != nil {
			return err
		}
		if _, err := blockchain.WriteVarstr31(w, outp.Vote); err != nil {
			return err
		}
		return outp.OutputCommitment.writeExtensibleString(w, outp.CommitmentSuffix, to.AssetVersion)

	default:
		return nil
	}
}

// ComputeOutputID assembles an intra-chain(!) output entry given a spend
// commitment and computes and returns its corresponding entry ID.
func ComputeOutputID(sc *SpendCommitment, inputType uint8, vote []byte) (h bc.Hash, err error) {
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
	var o bc.Entry
	switch inputType {
	case SpendInputType:
		o = bc.NewIntraChainOutput(src, &bc.Program{VmVersion: sc.VMVersion, Code: sc.ControlProgram}, 0)
	case VetoInputType:
		o = bc.NewVoteOutput(src, &bc.Program{VmVersion: sc.VMVersion, Code: sc.ControlProgram}, 0, vote)
	default:
		return h, fmt.Errorf("Input type error:[%v]", inputType)
	}

	h = bc.EntryID(o)
	return h, nil
}
