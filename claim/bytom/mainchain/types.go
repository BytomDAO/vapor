package mainchain

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"

	bytomtypes "github.com/vapor/claim/bytom/protocolbc/types"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/encoding/blockchain"
	"github.com/vapor/encoding/bufpool"
	chainjson "github.com/vapor/encoding/json"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
)

// Template represents a partially- or fully-signed transaction.
type Template struct {
	Transaction         *bytomtypes.Tx        `json:"raw_transaction"`
	SigningInstructions []*SigningInstruction `json:"signing_instructions"`
	Fee                 uint64                `json:"fee"`
	// AllowAdditional affects whether Sign commits to the tx sighash or
	// to individual details of the tx so far. When true, signatures
	// commit to tx details, and new details may be added but existing
	// ones cannot be changed. When false, signatures commit to the tx
	// as a whole, and any change to the tx invalidates the signature.
	AllowAdditional bool `json:"allow_additional_actions"`
}

// Hash return sign hash
func (t *Template) Hash(idx uint32) bc.Hash {
	return t.Transaction.SigHash(idx)
}

func MarshalText(t *Template) ([]byte, error) {
	buf := bufpool.Get()
	defer bufpool.Put(buf)

	ew := errors.NewWriter(buf)
	transaction, err := t.Transaction.MarshalText()
	if err != nil {
		return nil, err
	}

	if _, err := blockchain.WriteVarstr31(ew, transaction); err != nil {
		return nil, err
	}

	if _, err := blockchain.WriteVarint31(ew, uint64(len(t.SigningInstructions))); err != nil {
		return nil, err
	}

	for _, signingInstruction := range t.SigningInstructions {
		b, err := json.MarshalIndent(signingInstruction, "", "  ")
		if err != nil {
			return nil, err
		}
		if _, err := blockchain.WriteVarstr31(ew, b); err != nil {
			return nil, err
		}
	}

	if _, err := blockchain.WriteVarint63(ew, t.Fee); err != nil {
		return nil, err
	}

	allowAdditional, err := json.Marshal(t.AllowAdditional)
	if err != nil {
		return nil, err
	}
	if _, err := blockchain.WriteVarstr31(ew, allowAdditional); err != nil {
		return nil, err
	}

	if ew.Err() != nil {
		return nil, ew.Err()
	}

	enc := make([]byte, hex.EncodedLen(buf.Len()))
	hex.Encode(enc, buf.Bytes())
	return enc, nil
}

func UnmarshalText(text []byte, t *Template) error {
	decoded := make([]byte, hex.DecodedLen(len(text)))
	if _, err := hex.Decode(decoded, text); err != nil {
		return err
	}

	r := blockchain.NewReader(decoded)

	b, err := blockchain.ReadVarstr31(r)
	if err != nil {
		return err
	}

	t.Transaction = &bytomtypes.Tx{}

	if err = t.Transaction.UnmarshalText(b); err != nil {
		return err
	}

	n, err := blockchain.ReadVarint31(r)
	if err != nil {
		return errors.Wrap(err, "reading number of transactions")
	}

	for ; n > 0; n-- {
		var signingInstruction SigningInstruction
		b, err := blockchain.ReadVarstr31(r)
		if err != nil {
			return err
		}
		err = json.Unmarshal(b, &signingInstruction)
		if err != nil {
			fmt.Errorf("error on input %s: %s", b, err)
		}

		t.SigningInstructions = append(t.SigningInstructions, &signingInstruction)
	}

	if t.Fee, err = blockchain.ReadVarint63(r); err != nil {
		return err
	}

	b, err = blockchain.ReadVarstr31(r)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, &t.AllowAdditional)
	if err != nil {
		fmt.Errorf("error on input %s: %s", b, err)
	}

	if trailing := r.Len(); trailing > 0 {
		return fmt.Errorf("trailing garbage (%d bytes)", trailing)
	}

	return nil
}

// Action is a interface
type Action interface {
	Build(context.Context, *TemplateBuilder) error
}

// Receiver encapsulates information about where to send assets.
type Receiver struct {
	ControlProgram chainjson.HexBytes `json:"control_program,omitempty"`
	Address        string             `json:"address,omitempty"`
}

// ContractArgument for smart contract
type ContractArgument struct {
	Type    string          `json:"type"`
	RawData json.RawMessage `json:"raw_data"`
}

// RawTxSigArgument is signature-related argument for run contract
type RawTxSigArgument struct {
	RootXPub chainkd.XPub         `json:"xpub"`
	Path     []chainjson.HexBytes `json:"derivation_path"`
}

// DataArgument is the other argument for run contract
type DataArgument struct {
	Value string `json:"value"`
}
