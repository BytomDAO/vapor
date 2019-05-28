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
