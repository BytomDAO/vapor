package state

import (
	"github.com/vapor/consensus"
	"github.com/vapor/database/storage"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
)

// UtxoViewpoint represents a view into the set of unspent transaction outputs
type UtxoViewpoint struct {
	Entries map[bc.Hash]*storage.UtxoEntry
}

// NewUtxoViewpoint returns a new empty unspent transaction output view.
func NewUtxoViewpoint() *UtxoViewpoint {
	return &UtxoViewpoint{
		Entries: make(map[bc.Hash]*storage.UtxoEntry),
	}
}

func (view *UtxoViewpoint) ApplyTransaction(block *bc.Block, tx *bc.Tx, statusFail bool) error {
	for _, prevout := range tx.MainchainOutputIDs {
		entry, ok := view.Entries[prevout]
		if !ok {
			return errors.New("fail to find mainchain output entry")
		}

		if entry.Type != storage.CrosschainUTXOType {
			return errors.New("look up mainchainOutputID but find utxo not from mainchain")
		}

		if entry.Spent {
			return errors.New("mainchain output has been spent")
		}

		entry.BlockHeight = block.Height
		entry.SpendOutput()
	}

	for _, prevout := range tx.SpentOutputIDs {
		assetID := bc.AssetID{}
		entryOutput, err := tx.Entry(prevout)
		if err != nil {
			return err
		}

		switch output := entryOutput.(type) {
		case *bc.IntraChainOutput:
			assetID = *output.Source.Value.AssetId
		case *bc.VoteOutput:
			assetID = *output.Source.Value.AssetId
		default:
			return errors.Wrapf(bc.ErrEntryType, "entry %x has unexpected type %T", prevout.Bytes(), entryOutput)
		}

		if statusFail && assetID != *consensus.BTMAssetID {
			continue
		}

		entry, ok := view.Entries[prevout]
		if !ok {
			return errors.New("fail to find utxo entry")
		}

		if entry.Spent {
			return errors.New("utxo has been spent")
		}

		switch entry.Type {
		case storage.CrosschainUTXOType:
			return errors.New("look up spentOutputID but find utxo from mainchain")

		case storage.CoinbaseUTXOType:
			if (entry.BlockHeight + consensus.CoinbasePendingBlockNumber) > block.Height {
				return errors.New("coinbase utxo is not ready for use")
			}

		case storage.VoteUTXOType:
			if (entry.BlockHeight + consensus.VotePendingBlockNumber) > block.Height {
				return errors.New("Coin is  within the voting lock time")
			}
		}

		entry.SpendOutput()
	}

	for _, id := range tx.TxHeader.ResultIds {
		assetID := bc.AssetID{}
		entryOutput, err := tx.Entry(*id)
		if err != nil {
			continue
		}

		utxoType := storage.NormalUTXOType

		switch output := entryOutput.(type) {
		case *bc.IntraChainOutput:
			if output.Source.Value.Amount == uint64(0) {
				continue
			}
			assetID = *output.Source.Value.AssetId
		case *bc.VoteOutput:
			assetID = *output.Source.Value.AssetId
			utxoType = storage.VoteUTXOType
		default:
			// due to it's a retirement, utxo doesn't care this output type so skip it
			continue
		}

		if statusFail && assetID != *consensus.BTMAssetID {
			continue
		}

		if block != nil && len(block.Transactions) > 0 && block.Transactions[0].ID == tx.ID {
			utxoType = storage.CoinbaseUTXOType
		}
		view.Entries[*id] = storage.NewUtxoEntry(utxoType, block.Height, false)
	}
	return nil
}

func (view *UtxoViewpoint) ApplyBlock(block *bc.Block, txStatus *bc.TransactionStatus) error {
	for i, tx := range block.Transactions {
		statusFail, err := txStatus.GetStatus(i)
		if err != nil {
			return err
		}
		if err := view.ApplyTransaction(block, tx, statusFail); err != nil {
			return err
		}
	}
	return nil
}

func (view *UtxoViewpoint) CanSpend(hash *bc.Hash) bool {
	entry := view.Entries[*hash]
	return entry != nil && !entry.Spent
}

func (view *UtxoViewpoint) DetachTransaction(tx *bc.Tx, statusFail bool) error {
	for _, prevout := range tx.MainchainOutputIDs {
		// don't simply delete(view.Entries, prevout), because we need to delete from db in saveUtxoView()
		entry, ok := view.Entries[prevout]
		if ok && (entry.Type != storage.CrosschainUTXOType) {
			return errors.New("look up mainchainOutputID but find utxo not from mainchain")
		}

		if ok && !entry.Spent {
			return errors.New("try to revert an unspent utxo")
		}

		if !ok {
			view.Entries[prevout] = storage.NewUtxoEntry(storage.CrosschainUTXOType, 0, false)
			continue
		}
		entry.UnspendOutput()
	}

	for _, prevout := range tx.SpentOutputIDs {
		assetID := bc.AssetID{}
		entryOutput, err := tx.Entry(prevout)
		if err != nil {
			return err
		}

		utxoType := storage.NormalUTXOType
		switch output := entryOutput.(type) {
		case *bc.IntraChainOutput:
			assetID = *output.Source.Value.AssetId
		case *bc.VoteOutput:
			assetID = *output.Source.Value.AssetId
			utxoType = storage.VoteUTXOType
		default:
			return errors.Wrapf(bc.ErrEntryType, "entry %x has unexpected type %T", prevout.Bytes(), entryOutput)
		}

		if statusFail && assetID != *consensus.BTMAssetID {
			continue
		}

		entry, ok := view.Entries[prevout]
		if ok && (entry.Type == storage.CrosschainUTXOType) {
			return errors.New("look up SpentOutputIDs but find mainchain utxo")
		}

		if ok && !entry.Spent {
			return errors.New("try to revert an unspent utxo")
		}

		if !ok {
			view.Entries[prevout] = storage.NewUtxoEntry(utxoType, 0, false)
			continue
		}
		entry.UnspendOutput()
	}

	for _, id := range tx.TxHeader.ResultIds {
		assetID := bc.AssetID{}
		entryOutput, err := tx.Entry(*id)
		if err != nil {
			continue
		}

		utxoType := storage.NormalUTXOType
		switch output := entryOutput.(type) {
		case *bc.IntraChainOutput:
			if output.Source.Value.Amount == uint64(0) {
				continue
			}
			assetID = *output.Source.Value.AssetId
		case *bc.VoteOutput:
			assetID = *output.Source.Value.AssetId
			utxoType = storage.VoteUTXOType
		default:
			// due to it's a retirement, utxo doesn't care this output type so skip it
			continue
		}

		if statusFail && assetID != *consensus.BTMAssetID {
			continue
		}

		view.Entries[*id] = storage.NewUtxoEntry(utxoType, 0, true)
	}
	return nil
}

func (view *UtxoViewpoint) DetachBlock(block *bc.Block, txStatus *bc.TransactionStatus) error {
	for i := len(block.Transactions) - 1; i >= 0; i-- {
		statusFail, err := txStatus.GetStatus(i)
		if err != nil {
			return err
		}
		if err := view.DetachTransaction(block.Transactions[i], statusFail); err != nil {
			return err
		}
	}
	return nil
}

func (view *UtxoViewpoint) HasUtxo(hash *bc.Hash) bool {
	_, ok := view.Entries[*hash]
	return ok
}
