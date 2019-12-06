package bc

import (
	"github.com/bytom/vapor/crypto/sha3pool"
	"github.com/bytom/vapor/errors"
)

// Tx is a wrapper for the entries-based representation of a transaction.
type Tx struct {
	*TxHeader
	ID       Hash
	Entries  map[Hash]Entry
	InputIDs []Hash // 1:1 correspondence with TxData.Inputs

	SpentOutputIDs     []Hash
	MainchainOutputIDs []Hash
	GasInputIDs        []Hash
}

// SigHash ...
func (tx *Tx) SigHash(n uint32) (hash Hash) {
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

// IntraChainOutput try to get the intra-chain output entry by given hash
func (tx *Tx) IntraChainOutput(id Hash) (*IntraChainOutput, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	o, ok := e.(*IntraChainOutput)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return o, nil
}

// CrossChainOutput try to get the cross-chain output entry by given hash
func (tx *Tx) CrossChainOutput(id Hash) (*CrossChainOutput, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	o, ok := e.(*CrossChainOutput)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return o, nil
}

// Entry try to get the  entry by given hash
func (tx *Tx) Entry(id Hash) (Entry, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	return e, nil
}

// Spend try to get the spend entry by given hash
func (tx *Tx) Spend(id Hash) (*Spend, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	sp, ok := e.(*Spend)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return sp, nil
}

// VetoInput try to get the veto entry by given hash
func (tx *Tx) VetoInput(id Hash) (*VetoInput, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	sp, ok := e.(*VetoInput)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return sp, nil
}

// VoteOutput try to get the vote output entry by given hash
func (tx *Tx) VoteOutput(id Hash) (*VoteOutput, error) {
	e, ok := tx.Entries[id]
	if !ok || e == nil {
		return nil, errors.Wrapf(ErrMissingEntry, "id %x", id.Bytes())
	}
	o, ok := e.(*VoteOutput)
	if !ok {
		return nil, errors.Wrapf(ErrEntryType, "entry %x has unexpected type %T", id.Bytes(), e)
	}
	return o, nil
}
