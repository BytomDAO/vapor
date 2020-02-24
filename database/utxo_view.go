package database

import (
	"fmt"

	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/database/storage"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/state"
	"github.com/golang/protobuf/proto"
)

const utxoPreFix = "UT:"

func calcUtxoKey(hash *bc.Hash) []byte {
	return []byte(utxoPreFix + hash.String())
}

func getTransactionsUtxo(db dbm.DB, view *state.UtxoViewpoint, txs []*bc.Tx) error {
	fmt.Println("[very important]")
	for _, tx := range txs {
		fmt.Println("[very important] getTransactionsUtxo tx.string()", tx.String())
		for _, prevout := range tx.SpentOutputIDs {
			fmt.Println("[very important] getTransactionsUtxo prevout", prevout.String())
			if view.HasUtxo(&prevout) {
				continue
			}

			utxoEntry, err := GetUtxo(db, &prevout)
			fmt.Println("[why really important] prevout", prevout.String())
			fmt.Println("[why really important] utxoEntry:", utxoEntry.String(), "err", err)

			fmt.Println("calcKey:", calcUtxoKey(&prevout))
			data := db.Get(calcUtxoKey(&prevout))
			if data == nil {
				fmt.Println("why data is not here")
				continue
			}

			var utxo storage.UtxoEntry
			fmt.Println("Unmarshal data", data)
			if err := proto.Unmarshal(data, &utxo); err != nil {
				return errors.Wrap(err, "unmarshaling utxo entry")
			}

			view.Entries[prevout] = &utxo
		}

		for _, prevout := range tx.MainchainOutputIDs {
			fmt.Println("MainchainOutputIDs preout", prevout.String())
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

func GetUtxo(db dbm.DB, hash *bc.Hash) (*storage.UtxoEntry, error) {
	return getUtxo(db, hash)
}

func saveUtxoView(batch dbm.Batch, view *state.UtxoViewpoint) error {
	fmt.Println("[important] now go to saveUtxoView len entries:", len(view.Entries))
	for key, entry := range view.Entries {
		fmt.Println("[important] saveUtxoView key:", key.String(), " entry:", entry.String())
		if entry.Type == storage.CrosschainUTXOType && !entry.Spent {
			fmt.Println("[important] delete key:", calcUtxoKey(&key))
			batch.Delete(calcUtxoKey(&key))
			continue
		}

		if entry.Type == storage.NormalUTXOType && entry.Spent {
			fmt.Println("[important] delete key:", calcUtxoKey(&key))
			batch.Delete(calcUtxoKey(&key))
			continue
		}

		b, err := proto.Marshal(entry)
		if err != nil {
			return errors.Wrap(err, "marshaling utxo entry")
		}
		fmt.Println("[important set] calcUtxoKey(&key)", calcUtxoKey(&key))
		batch.Set(calcUtxoKey(&key), b)
	}
	return nil
}

func SaveUtxoView(batch dbm.Batch, view *state.UtxoViewpoint) error {
	return saveUtxoView(batch, view)
}
