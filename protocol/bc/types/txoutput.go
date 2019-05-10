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
		// TODO:
		// OutputCommitment
		// Unconsumed suffixes of the commitment and witness extensible strings.
		// TODO:
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
	case *IntraChainTxOutput:
		return outp.AssetAmount
	case *CrossChainTxOutput:
		return outp.AssetAmount
	default:
		return bc.AssetAmount{}
	}
}

// ControlProgram return the control program of the txoutput
func (to *TxOutput) ControlProgram() []byte {
	switch outp := to.TypedOutput.(type) {
	case *IntraChainTxOutput:
		return outp.ControlProgram
	case *CrossChainTxOutput:
		return outp.ControlProgram
	default:
		return nil
	}
}

// VMVersion return the VM version of the txoutput
func (to *TxOutput) VMVersion() uint64 {
	switch outp := to.TypedOutput.(type) {
	case *IntraChainTxOutput:
		return outp.VMVersion
	case *CrossChainTxOutput:
		return outp.VMVersion
	default:
		return 0
	}
}

// TODO:
func (to *TxOutput) readFrom(r *blockchain.Reader) (err error) {
	if to.AssetVersion, err = blockchain.ReadVarint63(r); err != nil {
		return errors.Wrap(err, "reading asset version")
	}
	return

	// var assetID bc.AssetID
	// t.CommitmentSuffix, err = blockchain.ReadExtensibleString(r, func(r *blockchain.Reader) error {
	// 	if t.AssetVersion != 1 {
	// 		return nil
	// 	}
	// 	var icType [1]byte
	// 	if _, err = io.ReadFull(r, icType[:]); err != nil {
	// 		return errors.Wrap(err, "reading input commitment type")
	// 	}
	// 	switch icType[0] {
	// 	case IssuanceInputType:
	// 		ii := new(IssuanceInput)
	// 		t.TypedInput = ii

	// 		if ii.Nonce, err = blockchain.ReadVarstr31(r); err != nil {
	// 			return err
	// 		}
	// 		if _, err = assetID.ReadFrom(r); err != nil {
	// 			return err
	// 		}
	// 		if ii.Amount, err = blockchain.ReadVarint63(r); err != nil {
	// 			return err
	// 		}

	// 	case SpendInputType:
	// 		si := new(SpendInput)
	// 		t.TypedInput = si
	// 		if si.SpendCommitmentSuffix, err = si.SpendCommitment.readFrom(r, 1); err != nil {
	// 			return err
	// 		}

	// 	case CoinbaseInputType:
	// 		ci := new(CoinbaseInput)
	// 		t.TypedInput = ci
	// 		if ci.Arbitrary, err = blockchain.ReadVarstr31(r); err != nil {
	// 			return err
	// 		}

	// 	default:
	// 		return fmt.Errorf("unsupported input type %d", icType[0])
	// 	}
	// 	return nil
	// })
	// if err != nil {
	// 	return err
	// }

	// t.WitnessSuffix, err = blockchain.ReadExtensibleString(r, func(r *blockchain.Reader) error {
	// 	if t.AssetVersion != 1 {
	// 		return nil
	// 	}

	// 	switch inp := t.TypedInput.(type) {
	// 	case *IssuanceInput:
	// 		if inp.AssetDefinition, err = blockchain.ReadVarstr31(r); err != nil {
	// 			return err
	// 		}
	// 		if inp.VMVersion, err = blockchain.ReadVarint63(r); err != nil {
	// 			return err
	// 		}
	// 		if inp.IssuanceProgram, err = blockchain.ReadVarstr31(r); err != nil {
	// 			return err
	// 		}
	// 		if inp.AssetID() != assetID {
	// 			return errBadAssetID
	// 		}
	// 		if inp.Arguments, err = blockchain.ReadVarstrList(r); err != nil {
	// 			return err
	// 		}

	// 	case *SpendInput:
	// 		if inp.Arguments, err = blockchain.ReadVarstrList(r); err != nil {
	// 			return err
	// 		}
	// 	}
	// 	return nil
	// })

	// return err

	// if to.CommitmentSuffix, err = to.OutputCommitment.readFrom(r, to.AssetVersion); err != nil {
	// 	return errors.Wrap(err, "reading output commitment")
	// }

	// // read and ignore the (empty) output witness
	// _, err = blockchain.ReadVarstr31(r)
	// return errors.Wrap(err, "reading output witness")
}

// TODO:
func (to *TxOutput) writeTo(w io.Writer) error {
	if _, err := blockchain.WriteVarint63(w, to.AssetVersion); err != nil {
		return errors.Wrap(err, "writing asset version")
	}

	// if err := to.writeCommitment(w); err != nil {
	// 	return errors.Wrap(err, "writing output commitment")
	// }

	// if _, err := blockchain.WriteVarstr31(w, nil); err != nil {
	// 	return errors.Wrap(err, "writing witness")
	// }
	return nil
}

// TODO:
func (to *TxOutput) writeCommitment(w io.Writer) error {
	return nil
	// return to.OutputCommitment.writeExtensibleString(w, to.CommitmentSuffix, to.AssetVersion)
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
