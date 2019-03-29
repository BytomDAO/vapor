package protocolbc

import (
	"github.com/vapor/crypto/sha3pool"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
)

// Tx is a wrapper for the entries-based representation of a transaction.
type Tx struct {
	*bc.TxHeader
	ID       bc.Hash
	Entries  map[bc.Hash]bc.Entry
	InputIDs []bc.Hash // 1:1 correspondence with TxData.Inputs

	SpentOutputIDs []bc.Hash
	GasInputIDs    []bc.Hash
}

// SigHash ...
func (tx *Tx) SigHash(n uint32) (hash bc.Hash) {
	hasher := sha3pool.Get256()
	defer sha3pool.Put256(hasher)

	tx.InputIDs[n].WriteTo(hasher)
	tx.ID.WriteTo(hasher)
	hash.ReadFrom(hasher)
	return hash
}

// Convenience routines for accessing entries of specific types by ID.
var (
	ErrEntryType    = errors.New("invalid entry type")
	ErrMissingEntry = errors.New("missing entry")
)

// Output try to get the output entry by given hash
func (tx *Tx) Output(id bc.Hash) (*bc.Output, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	o, ok := e.(*bc.Output)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return o, nil
}

// Spend try to get the spend entry by given hash
func (tx *Tx) Spend(id bc.Hash) (*bc.Spend, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	sp, ok := e.(*bc.Spend)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return sp, nil
}

// Issuance try to get the issuance entry by given hash
func (tx *Tx) Issuance(id bc.Hash) (*bc.Issuance, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	iss, ok := e.(*bc.Issuance)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return iss, nil
}
