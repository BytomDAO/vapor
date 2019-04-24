package database

import (
	"github.com/golang/protobuf/proto"

	dbm "github.com/vapor/database/db"
	"github.com/vapor/database/orm"
	"github.com/vapor/database/storage"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/state"
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
		if entry.Spent && !entry.IsCoinBase && !entry.IsCliam {
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

func getTransactionsUtxoFromSQLDB(db dbm.SQLDB, view *state.UtxoViewpoint, txs []*bc.Tx) error {
	for _, tx := range txs {
		for _, prevout := range tx.SpentOutputIDs {
			if view.HasUtxo(&prevout) {
				continue
			}
			data := &orm.UtxoViewpoint{
				OutputID: prevout.String(),
			}

			if err := db.Db().Where(data).Find(data).Error; err != nil {
				continue
			}

			view.Entries[prevout] = &storage.UtxoEntry{
				IsCoinBase:  data.IsCoinBase,
				BlockHeight: data.BlockHeight,
				Spent:       data.Spent,
				IsCliam:     data.IsCliam,
			}
		}
	}

	return nil
}

func getUtxoFromSQLDB(db dbm.SQLDB, hash *bc.Hash) (*storage.UtxoEntry, error) {
	utxoViewpoint := &orm.UtxoViewpoint{
		OutputID: hash.String(),
	}

	if err := db.Db().Where(utxoViewpoint).Find(utxoViewpoint).Error; err != nil {
		return nil, err
	}

	return &storage.UtxoEntry{
		IsCoinBase:  utxoViewpoint.IsCoinBase,
		BlockHeight: utxoViewpoint.BlockHeight,
		Spent:       utxoViewpoint.Spent,
		IsCliam:     utxoViewpoint.IsCliam,
	}, nil
}

func saveUtxoViewToSQLDB(db dbm.SQLDB, view *state.UtxoViewpoint) error {
	for key, entry := range view.Entries {
		if entry.Spent && !entry.IsCoinBase && !entry.IsCliam {
			if err := db.Db().Where("out_put_id = ?", key.String()).Delete(&orm.UtxoViewpoint{}).Error; err != nil {
				return err
			}
			continue
		}
		utxoViewpoint := &orm.UtxoViewpoint{
			OutputID:    key.String(),
			IsCoinBase:  entry.IsCoinBase,
			BlockHeight: entry.BlockHeight,
			Spent:       entry.Spent,
			IsCliam:     entry.IsCliam,
		}
		if err := db.Db().Save(utxoViewpoint).Error; err != nil {
			return err
		}
	}
	return nil
}

func SaveUtxoView(batch dbm.Batch, view *state.UtxoViewpoint) error {
	return saveUtxoView(batch, view)
}
