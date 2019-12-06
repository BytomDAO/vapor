package database

import (
	"github.com/golang/protobuf/proto"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/database/storage"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/state"
)

const utxoPreFix = "UT:"

func calcUtxoKey(hash *bc.Hash) []byte {
	return []byte(utxoPreFix + hash.String())
}

func getTransactionsUtxo(db dbm.DB, view *state.UtxoViewpoint, txs []*bc.Tx) error {
	for _, tx := range txs {
		for _, prevout := range tx.SpentOutputIDs {
			if view.HasUtxo(&prevout) {
				continue
			}

			data := db.Get(calcUtxoKey(&prevout))
			if data == nil {
				continue
			}

			var utxo storage.UtxoEntry
			if err := proto.Unmarshal(data, &utxo); err != nil {
				return errors.Wrap(err, "unmarshaling utxo entry")
			}

			view.Entries[prevout] = &utxo
		}

		for _, prevout := range tx.MainchainOutputIDs {
			if view.HasUtxo(&prevout) {
				continue
			}

			data := db.Get(calcUtxoKey(&prevout))
			if data == nil {
				view.Entries[prevout] = storage.NewUtxoEntry(storage.CrosschainUTXOType, 0, false)
				continue
			}

			var utxo storage.UtxoEntry
			if err := proto.Unmarshal(data, &utxo); err != nil {
				return errors.Wrap(err, "unmarshaling mainchain ouput entry")
			}

			view.Entries[prevout] = &utxo
		}
	}

	return nil
}

func getUtxo(db dbm.DB, hash *bc.Hash) (*storage.UtxoEntry, error) {
	var utxo storage.UtxoEntry
	data := db.Get(calcUtxoKey(hash))
	if data == nil {
		return nil, errors.New("can't find utxo in db")
	}
	if err := proto.Unmarshal(data, &utxo); err != nil {
		return nil, errors.Wrap(err, "unmarshaling utxo entry")
	}
	return &utxo, nil
}

func saveUtxoView(batch dbm.Batch, view *state.UtxoViewpoint) error {
	for key, entry := range view.Entries {
		if entry.Type == storage.CrosschainUTXOType && !entry.Spent {
			batch.Delete(calcUtxoKey(&key))
			continue
		}

		if entry.Type == storage.NormalUTXOType && entry.Spent {
			batch.Delete(calcUtxoKey(&key))
			continue
		}

		b, err := proto.Marshal(entry)
		if err != nil {
			return errors.Wrap(err, "marshaling utxo entry")
		}
		batch.Set(calcUtxoKey(&key), b)
	}
	return nil
}

func SaveUtxoView(batch dbm.Batch, view *state.UtxoViewpoint) error {
	return saveUtxoView(batch, view)
}
