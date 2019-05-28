package database

import (
	"encoding/json"

	dbm "github.com/vapor/database/leveldb"
	// "github.com/vapor/database/storage"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/state"
)

const mainchainOutputPreFix = "MCO:"

func calcMainchainOutputKey(hash *bc.Hash) []byte {
	return []byte(mainchainOutputPreFix + hash.String())
}

func getMainchainOutputToClaim(db dbm.DB, view *state.MainchainOutputViewpoint, txs []*bc.Tx) error {
	for _, tx := range txs {
		for _, prevout := range tx.MainchainOutputIDs {
			if view.HasEntry(&prevout) {
				continue
			}

			// 	data := db.Get(calcUtxoKey(&prevout))
			// 	if data == nil {
			// 		continue
			// 	}

			// 	var utxo storage.UtxoEntry
			// 	if err := proto.Unmarshal(data, &utxo); err != nil {
			// 		return errors.Wrap(err, "unmarshaling utxo entry")
			// 	}

			view.Entries[prevout] = nil
		}
	}

	return nil
}

func saveMainchainOutputView(batch dbm.Batch, view *state.MainchainOutputViewpoint) error {
	if view == nil {
		return nil
	}

	for key, entry := range view.Entries {
		// TODO:???
		if !entry.Claimed {
			batch.Delete(calcMainchainOutputKey(&key))
			continue
		}

		b, err := json.Marshal(entry)
		if err != nil {
			return errors.Wrap(err, "marshaling mainchain output entry")
		}
		batch.Set(calcMainchainOutputKey(&key), b)
	}
	return nil
}
