package test

import (
	"encoding/json"
	"os"
	"testing"

	acc "github.com/vapor/account"
	"github.com/vapor/database"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/protocol/bc"
	mock "github.com/vapor/test/mock"
	"github.com/vapor/testutil"
)

func TestFindUtxo(t *testing.T) {
	currentHeight := func() uint64 { return 9527 }
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	cases := []struct {
		uk             mock.UTXOKeeper
		dbUtxos        map[string]*acc.UTXO
		outHash        bc.Hash
		useUnconfirmed bool
		wantUtxo       *acc.UTXO
		err            error
	}{
		{
			uk: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed:   map[bc.Hash]*acc.UTXO{},
			},
			dbUtxos:  map[string]*acc.UTXO{},
			outHash:  bc.NewHash([32]byte{0x01}),
			wantUtxo: nil,
			err:      acc.ErrMatchUTXO,
		},
		{
			uk: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed: map[bc.Hash]*acc.UTXO{
					bc.NewHash([32]byte{0x01}): &acc.UTXO{OutputID: bc.NewHash([32]byte{0x01})},
				},
			},
			dbUtxos:        map[string]*acc.UTXO{},
			outHash:        bc.NewHash([32]byte{0x01}),
			wantUtxo:       nil,
			useUnconfirmed: false,
			err:            acc.ErrMatchUTXO,
		},
		{
			uk: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed: map[bc.Hash]*acc.UTXO{
					bc.NewHash([32]byte{0x01}): &acc.UTXO{OutputID: bc.NewHash([32]byte{0x01})},
				},
			},
			dbUtxos:        map[string]*acc.UTXO{},
			outHash:        bc.NewHash([32]byte{0x01}),
			wantUtxo:       &acc.UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			useUnconfirmed: true,
			err:            nil,
		},
		{
			uk: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed:   map[bc.Hash]*acc.UTXO{},
			},
			dbUtxos: map[string]*acc.UTXO{
				string(database.StandardUTXOKey(bc.NewHash([32]byte{0x01}))): &acc.UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			},
			outHash:        bc.NewHash([32]byte{0x01}),
			wantUtxo:       &acc.UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			useUnconfirmed: false,
			err:            nil,
		},
		{
			uk: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed:   map[bc.Hash]*acc.UTXO{},
			},
			dbUtxos: map[string]*acc.UTXO{
				string(database.ContractUTXOKey(bc.NewHash([32]byte{0x01}))): &acc.UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			},
			outHash:        bc.NewHash([32]byte{0x01}),
			wantUtxo:       &acc.UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			useUnconfirmed: false,
			err:            nil,
		},
	}

	for i, c := range cases {
		for k, u := range c.dbUtxos {
			data, err := json.Marshal(u)
			if err != nil {
				t.Error(err)
			}
			testDB.Set([]byte(k), data)
		}

		gotUtxo, err := c.uk.FindUtxo(c.outHash, c.useUnconfirmed)
		if !testutil.DeepEqual(gotUtxo, c.wantUtxo) {
			t.Errorf("case %d: got %v want %v", i, gotUtxo, c.wantUtxo)
		}
		if err != c.err {
			t.Errorf("case %d: got %v want %v", i, err, c.err)
		}

		for _, u := range c.dbUtxos {
			c.uk.Store.DeleteStandardUTXO(u.OutputID)
		}
	}
}

func checkUtxoKeeperEqual(t *testing.T, i int, a, b *mock.UTXOKeeper) {
	if !testutil.DeepEqual(a.Unconfirmed, b.Unconfirmed) {
		t.Errorf("case %d: unconfirmed got %v want %v", i, a.Unconfirmed, b.Unconfirmed)
	}
	if !testutil.DeepEqual(a.Reserved, b.Reserved) {
		t.Errorf("case %d: reserved got %v want %v", i, a.Reserved, b.Reserved)
	}
	if !testutil.DeepEqual(a.Reservations, b.Reservations) {
		t.Errorf("case %d: reservations got %v want %v", i, a.Reservations, b.Reservations)
	}
}
