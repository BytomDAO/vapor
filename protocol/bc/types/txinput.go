package types

import (
	"fmt"
	"io"

	"github.com/vapor/encoding/blockchain"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
)

// serflag variables for input types.
const (
	IssuanceInputType uint8 = iota
	SpendInputType
	CoinbaseInputType
	UnvoteInputType
)

type (
	// TxInput is the top level struct of tx input.
	TxInput struct {
		AssetVersion uint64
		TypedInput
		CommitmentSuffix []byte
		WitnessSuffix    []byte
	}

	// TypedInput return the txinput type.
	TypedInput interface {
		InputType() uint8
	}
)

var errBadAssetID = errors.New("asset ID does not match other issuance parameters")

// AssetAmount return the asset id and amount of the txinput.
func (t *TxInput) AssetAmount() bc.AssetAmount {
	switch inp := t.TypedInput.(type) {
	case *SpendInput:
		return inp.AssetAmount
	}
	return bc.AssetAmount{}
}

// AssetID return the assetID of the txinput
func (t *TxInput) AssetID() bc.AssetID {
	switch inp := t.TypedInput.(type) {
	case *SpendInput:
		return *inp.AssetId

	}
	return bc.AssetID{}
}

// Amount return the asset amount of the txinput
func (t *TxInput) Amount() uint64 {
	switch inp := t.TypedInput.(type) {
	case *SpendInput:
		return inp.Amount
	}
	return 0
}

// ControlProgram return the control program of the spend input
func (t *TxInput) ControlProgram() []byte {
	if si, ok := t.TypedInput.(*SpendInput); ok {
		return si.ControlProgram
	}
	return nil
}

// Arguments get the args for the input
func (t *TxInput) Arguments() [][]byte {
	switch inp := t.TypedInput.(type) {
	case *SpendInput:
		return inp.Arguments
	}
	return nil
}

// SetArguments set the args for the input
func (t *TxInput) SetArguments(args [][]byte) {
	switch inp := t.TypedInput.(type) {
	case *SpendInput:
		inp.Arguments = args
	}
}

// SpentOutputID calculate the hash of spended output
func (t *TxInput) SpentOutputID() (o bc.Hash, err error) {
	if si, ok := t.TypedInput.(*SpendInput); ok {
		o, err = ComputeOutputID(&si.SpendCommitment)
	}
	return o, err
}

func (t *TxInput) readFrom(r *blockchain.Reader) (err error) {
	if t.AssetVersion, err = blockchain.ReadVarint63(r); err != nil {
		return err
	}

	t.CommitmentSuffix, err = blockchain.ReadExtensibleString(r, func(r *blockchain.Reader) error {
		if t.AssetVersion != 1 {
			return nil
		}
		var icType [1]byte
		if _, err = io.ReadFull(r, icType[:]); err != nil {
			return errors.Wrap(err, "reading input commitment type")
		}
		switch icType[0] {
		case SpendInputType:
			si := new(SpendInput)
			t.TypedInput = si
			if si.SpendCommitmentSuffix, err = si.SpendCommitment.readFrom(r, 1); err != nil {
				return err
			}

		case CoinbaseInputType:
			ci := new(CoinbaseInput)
			t.TypedInput = ci
			if ci.Arbitrary, err = blockchain.ReadVarstr31(r); err != nil {
				return err
			}

		default:
			return fmt.Errorf("unsupported input type %d", icType[0])
		}
		return nil
	})
	if err != nil {
		return err
	}

	t.WitnessSuffix, err = blockchain.ReadExtensibleString(r, func(r *blockchain.Reader) error {
		if t.AssetVersion != 1 {
			return nil
		}

		switch inp := t.TypedInput.(type) {
		case *SpendInput:
			if inp.Arguments, err = blockchain.ReadVarstrList(r); err != nil {
				return err
			}
		}
		return nil
	})

	return err
}

func (t *TxInput) writeTo(w io.Writer) error {
	if _, err := blockchain.WriteVarint63(w, t.AssetVersion); err != nil {
		return errors.Wrap(err, "writing asset version")
	}

	if _, err := blockchain.WriteExtensibleString(w, t.CommitmentSuffix, t.writeInputCommitment); err != nil {
		return errors.Wrap(err, "writing input commitment")
	}

	_, err := blockchain.WriteExtensibleString(w, t.WitnessSuffix, t.writeInputWitness)
	return errors.Wrap(err, "writing input witness")
}

func (t *TxInput) writeInputCommitment(w io.Writer) (err error) {
	if t.AssetVersion != 1 {
		return nil
	}

	switch inp := t.TypedInput.(type) {
	case *SpendInput:
		if _, err = w.Write([]byte{SpendInputType}); err != nil {
			return err
		}
		return inp.SpendCommitment.writeExtensibleString(w, inp.SpendCommitmentSuffix, t.AssetVersion)

	case *CoinbaseInput:
		if _, err = w.Write([]byte{CoinbaseInputType}); err != nil {
			return err
		}
		if _, err = blockchain.WriteVarstr31(w, inp.Arbitrary); err != nil {
			return errors.Wrap(err, "writing coinbase arbitrary")
		}
	}
	return nil
}

func (t *TxInput) writeInputWitness(w io.Writer) error {
	if t.AssetVersion != 1 {
		return nil
	}

	switch inp := t.TypedInput.(type) {
	case *SpendInput:
		_, err := blockchain.WriteVarstrList(w, inp.Arguments)
		return err
	}
	return nil
}
