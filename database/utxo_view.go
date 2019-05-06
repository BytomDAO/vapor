package database

import (
	"github.com/golang/protobuf/proto"
	"github.com/jinzhu/gorm"

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
			data := &orm.Utxo{
				OutputID: prevout.String(),
			}

			if err := db.Db().Where(data).Find(data).Error; err != nil {
				if err != gorm.ErrRecordNotFound {
					return err
				}
				continue
			}

			view.Entries[prevout] = &storage.UtxoEntry{
				IsCoinBase:  data.IsCoinBase,
				BlockHeight: data.BlockHeight,
				Spent:       data.Spent,
			}
		}
	}

	return nil
}

func getUtxoFromSQLDB(db dbm.SQLDB, hash *bc.Hash) (*storage.UtxoEntry, error) {
	utxoViewpoint := &orm.Utxo{
		OutputID: hash.String(),
	}

	if err := db.Db().Where(utxoViewpoint).Find(utxoViewpoint).Error; err != nil {
		return nil, err
	}

	return &storage.UtxoEntry{
		IsCoinBase:  utxoViewpoint.IsCoinBase,
		BlockHeight: utxoViewpoint.BlockHeight,
		Spent:       utxoViewpoint.Spent,
	}, nil
}

func saveUtxoViewToSQLDB(tx *gorm.DB, view *state.UtxoViewpoint) error {
	for key, entry := range view.Entries {
		if entry.Spent && !entry.IsCoinBase && !entry.IsCliam {
			if err := tx.Where("output_id = ?", key.String()).Delete(&orm.Utxo{}).Error; err != nil {
				return err
			}
			continue
		}
		utxoViewpoint := &orm.Utxo{
			OutputID:    key.String(),
			IsCoinBase:  entry.IsCoinBase,
			BlockHeight: entry.BlockHeight,
			Spent:       entry.Spent,
		}

		if entry.IsCoinBase {
			db := tx.Model(&orm.Utxo{}).Where(utxoViewpoint).Update("spent", entry.Spent)
			if err := db.Error; err != nil {
				return err
			}
			if db.RowsAffected != 0 {
				continue
			}
		}

		if err := tx.Create(utxoViewpoint).Error; err != nil {
			return err
		}
	}
	return nil
}

func SaveUtxoView(batch dbm.Batch, view *state.UtxoViewpoint) error {
	return saveUtxoView(batch, view)
}
