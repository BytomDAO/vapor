package test

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	acc "github.com/vapor/account"
	"github.com/vapor/database"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/protocol/bc"
	mock "github.com/vapor/test/mock"
	"github.com/vapor/testutil"
)

func TestReserveParticular(t *testing.T) {
	currentHeight := func() uint64 { return 9527 }
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	cases := []struct {
		before      mock.UTXOKeeper
		after       mock.UTXOKeeper
		err         error
		reserveHash bc.Hash
		exp         time.Time
	}{
		{
			before: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed: map[bc.Hash]*acc.UTXO{
					bc.NewHash([32]byte{0x01}): &acc.UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				Reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 0,
				},
				Reservations: map[uint64]*mock.Reservation{},
			},
			after: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed: map[bc.Hash]*acc.UTXO{
					bc.NewHash([32]byte{0x01}): &acc.UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				Reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 0,
				},
				Reservations: map[uint64]*mock.Reservation{},
			},
			reserveHash: bc.NewHash([32]byte{0x01}),
			err:         acc.ErrReserved,
		},
		{
			before: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed: map[bc.Hash]*acc.UTXO{
					bc.NewHash([32]byte{0x01}): &acc.UTXO{
						OutputID:    bc.NewHash([32]byte{0x01}),
						AccountID:   "testAccount",
						Amount:      3,
						ValidHeight: 9528,
					},
				},
				Reserved:     map[bc.Hash]uint64{},
				Reservations: map[uint64]*mock.Reservation{},
			},
			after: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed: map[bc.Hash]*acc.UTXO{
					bc.NewHash([32]byte{0x01}): &acc.UTXO{
						OutputID:    bc.NewHash([32]byte{0x01}),
						AccountID:   "testAccount",
						Amount:      3,
						ValidHeight: 9528,
					},
				},
				Reserved:     map[bc.Hash]uint64{},
				Reservations: map[uint64]*mock.Reservation{},
			},
			reserveHash: bc.NewHash([32]byte{0x01}),
			err:         acc.ErrImmature,
		},
		{
			before: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed: map[bc.Hash]*acc.UTXO{
					bc.NewHash([32]byte{0x01}): &acc.UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				Reserved:     map[bc.Hash]uint64{},
				Reservations: map[uint64]*mock.Reservation{},
			},
			after: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed: map[bc.Hash]*acc.UTXO{
					bc.NewHash([32]byte{0x01}): &acc.UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				Reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
				Reservations: map[uint64]*mock.Reservation{
					1: &mock.Reservation{
						ID: 1,
						UTXOs: []*acc.UTXO{
							&acc.UTXO{
								OutputID:  bc.NewHash([32]byte{0x01}),
								AccountID: "testAccount",
								Amount:    3,
							},
						},
						Change: 0,
						Expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			reserveHash: bc.NewHash([32]byte{0x01}),
			err:         nil,
			exp:         time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
		},
	}

	for i, c := range cases {
		if _, err := c.before.ReserveParticular(c.reserveHash, true, c.exp); err != c.err {
			t.Errorf("case %d: got error %v want error %v", i, err, c.err)
		}
		checkUtxoKeeperEqual(t, i, &c.before, &c.after)
	}
}

func TestFindUtxos(t *testing.T) {
	currentHeight := func() uint64 { return 9527 }
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	cases := []struct {
		uk             mock.UTXOKeeper
		dbUtxos        []*acc.UTXO
		useUnconfirmed bool
		wantUtxos      []*acc.UTXO
		immatureAmount uint64
		vote           []byte
	}{
		{
			uk: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed:   map[bc.Hash]*acc.UTXO{},
			},
			dbUtxos:        []*acc.UTXO{},
			useUnconfirmed: true,
			wantUtxos:      []*acc.UTXO{},
			immatureAmount: 0,
		},
		{
			uk: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed:   map[bc.Hash]*acc.UTXO{},
			},
			dbUtxos: []*acc.UTXO{
				&acc.UTXO{
					OutputID:  bc.NewHash([32]byte{0x01}),
					AccountID: "testAccount",
					Amount:    3,
				},
				&acc.UTXO{
					OutputID:  bc.NewHash([32]byte{0x02}),
					AccountID: "testAccount",
					AssetID:   bc.AssetID{V0: 6},
					Amount:    3,
				},
			},
			useUnconfirmed: false,
			wantUtxos: []*acc.UTXO{
				&acc.UTXO{
					OutputID:  bc.NewHash([32]byte{0x01}),
					AccountID: "testAccount",
					Amount:    3,
				},
			},
			immatureAmount: 0,
		},
		{
			uk: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed:   map[bc.Hash]*acc.UTXO{},
			},
			dbUtxos: []*acc.UTXO{
				&acc.UTXO{
					OutputID:    bc.NewHash([32]byte{0x02}),
					AccountID:   "testAccount",
					Amount:      3,
					ValidHeight: 9528,
				},
			},
			useUnconfirmed: false,
			wantUtxos:      []*acc.UTXO{},
			immatureAmount: 3,
		},
		{
			uk: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed: map[bc.Hash]*acc.UTXO{
					bc.NewHash([32]byte{0x01}): &acc.UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
			},
			dbUtxos: []*acc.UTXO{
				&acc.UTXO{
					OutputID:  bc.NewHash([32]byte{0x02}),
					AccountID: "testAccount",
					Amount:    3,
				},
			},
			useUnconfirmed: false,
			wantUtxos: []*acc.UTXO{
				&acc.UTXO{
					OutputID:  bc.NewHash([32]byte{0x02}),
					AccountID: "testAccount",
					Amount:    3,
				},
			},
			immatureAmount: 0,
		},
		{
			uk: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed: map[bc.Hash]*acc.UTXO{
					bc.NewHash([32]byte{0x11}): &acc.UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
			},
			dbUtxos: []*acc.UTXO{
				&acc.UTXO{
					OutputID:  bc.NewHash([32]byte{0x02}),
					AccountID: "testAccount",
					Amount:    3,
				},
			},
			useUnconfirmed: true,
			wantUtxos: []*acc.UTXO{
				&acc.UTXO{
					OutputID:  bc.NewHash([32]byte{0x02}),
					AccountID: "testAccount",
					Amount:    3,
				},
				&acc.UTXO{
					OutputID:  bc.NewHash([32]byte{0x01}),
					AccountID: "testAccount",
					Amount:    3,
				},
			},
			immatureAmount: 0,
		},
		{
			uk: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed: map[bc.Hash]*acc.UTXO{
					bc.NewHash([32]byte{0x01}): &acc.UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    1,
					},
					bc.NewHash([32]byte{0x02}): &acc.UTXO{
						OutputID:  bc.NewHash([32]byte{0x02}),
						AccountID: "notMe",
						Amount:    2,
					},
				},
			},
			dbUtxos: []*acc.UTXO{
				&acc.UTXO{
					OutputID:  bc.NewHash([32]byte{0x03}),
					AccountID: "testAccount",
					Amount:    3,
				},
				&acc.UTXO{
					OutputID:  bc.NewHash([32]byte{0x04}),
					AccountID: "notMe",
					Amount:    4,
				},
			},
			useUnconfirmed: true,
			wantUtxos: []*acc.UTXO{
				&acc.UTXO{
					OutputID:  bc.NewHash([32]byte{0x03}),
					AccountID: "testAccount",
					Amount:    3,
				},
				&acc.UTXO{
					OutputID:  bc.NewHash([32]byte{0x01}),
					AccountID: "testAccount",
					Amount:    1,
				},
			},
			immatureAmount: 0,
		},
		{
			uk: mock.UTXOKeeper{
				Store:         database.NewAccountStore(testDB),
				CurrentHeight: currentHeight,
				Unconfirmed:   map[bc.Hash]*acc.UTXO{},
			},
			dbUtxos: []*acc.UTXO{
				&acc.UTXO{
					OutputID:  bc.NewHash([32]byte{0x01}),
					AccountID: "testAccount",
					Amount:    6,
					Vote:      []byte("af594006a40837d9f028daabb6d589df0b9138daefad5683e5233c2646279217294a8d532e60863bcf196625a35fb8ceeffa3c09610eb92dcfb655a947f13269"),
				},
			},
			useUnconfirmed: false,
			wantUtxos: []*acc.UTXO{
				&acc.UTXO{
					OutputID:  bc.NewHash([32]byte{0x01}),
					AccountID: "testAccount",
					Amount:    6,
					Vote:      []byte("af594006a40837d9f028daabb6d589df0b9138daefad5683e5233c2646279217294a8d532e60863bcf196625a35fb8ceeffa3c09610eb92dcfb655a947f13269"),
				},
			},
			immatureAmount: 0,
			vote:           []byte("af594006a40837d9f028daabb6d589df0b9138daefad5683e5233c2646279217294a8d532e60863bcf196625a35fb8ceeffa3c09610eb92dcfb655a947f13269"),
		},
	}

	for i, c := range cases {
		for _, u := range c.dbUtxos {
			if err := c.uk.Store.SetStandardUTXO(u.OutputID, u); err != nil {
				t.Error(err)
			}
		}

		gotUtxos, immatureAmount := c.uk.FindUtxos("testAccount", &bc.AssetID{}, c.useUnconfirmed, c.vote)
		if !testutil.DeepEqual(gotUtxos, c.wantUtxos) {
			t.Errorf("case %d: got %v want %v", i, gotUtxos, c.wantUtxos)
		}
		if immatureAmount != c.immatureAmount {
			t.Errorf("case %d: got %v want %v", i, immatureAmount, c.immatureAmount)
		}

		for _, u := range c.dbUtxos {
			c.uk.Store.DeleteStandardUTXO(u.OutputID)
		}
	}
}

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
