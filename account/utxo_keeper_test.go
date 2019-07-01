package account

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/golang/groupcache/lru"
	log "github.com/sirupsen/logrus"
	"github.com/vapor/blockchain/txbuilder"
	"github.com/vapor/crypto/ed25519/chainkd"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/database/storage"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
	"github.com/vapor/testutil"
)

func TestAddUnconfirmedUtxo(t *testing.T) {
	cases := []struct {
		before   utxoKeeper
		after    utxoKeeper
		addUtxos []*UTXO
	}{
		{
			before: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{},
			},
			after: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{},
			},
			addUtxos: []*UTXO{},
		},
		{
			before: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{},
			},
			after: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
				},
			},
			addUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			},
		},
		{
			before: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
				},
			},
			after: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
				},
			},
			addUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			},
		},
		{
			before: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
				},
			},
			after: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
					bc.NewHash([32]byte{0x02}): &UTXO{OutputID: bc.NewHash([32]byte{0x02})},
					bc.NewHash([32]byte{0x03}): &UTXO{OutputID: bc.NewHash([32]byte{0x03})},
				},
			},
			addUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x02})},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03})},
			},
		},
	}

	for i, c := range cases {
		c.before.AddUnconfirmedUtxo(c.addUtxos)
		if !testutil.DeepEqual(c.before, c.after) {
			t.Errorf("case %d: got %v want %v", i, c.before, c.after)
		}
	}
}

func TestCancel(t *testing.T) {
	cases := []struct {
		before    utxoKeeper
		after     utxoKeeper
		cancelRid uint64
	}{
		{
			before: utxoKeeper{
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			cancelRid: 0,
		},
		{
			before: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
				reservations: map[uint64]*reservation{
					1: &reservation{
						id: 1,
						utxos: []*UTXO{
							&UTXO{OutputID: bc.NewHash([32]byte{0x01})},
						},
						change: 9527,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			after: utxoKeeper{
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			cancelRid: 1,
		},
		{
			before: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
				reservations: map[uint64]*reservation{
					1: &reservation{
						id: 1,
						utxos: []*UTXO{
							&UTXO{OutputID: bc.NewHash([32]byte{0x01})},
						},
						change: 9527,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			after: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
				reservations: map[uint64]*reservation{
					1: &reservation{
						id: 1,
						utxos: []*UTXO{
							&UTXO{OutputID: bc.NewHash([32]byte{0x01})},
						},
						change: 9527,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			cancelRid: 2,
		},
		{
			before: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
					bc.NewHash([32]byte{0x02}): 3,
					bc.NewHash([32]byte{0x03}): 3,
					bc.NewHash([32]byte{0x04}): 3,
				},
				reservations: map[uint64]*reservation{
					1: &reservation{
						id: 1,
						utxos: []*UTXO{
							&UTXO{OutputID: bc.NewHash([32]byte{0x01})},
						},
						change: 9527,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
					3: &reservation{
						id: 3,
						utxos: []*UTXO{
							&UTXO{OutputID: bc.NewHash([32]byte{0x02})},
							&UTXO{OutputID: bc.NewHash([32]byte{0x03})},
							&UTXO{OutputID: bc.NewHash([32]byte{0x04})},
						},
						change: 9528,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			after: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
				reservations: map[uint64]*reservation{
					1: &reservation{
						id: 1,
						utxos: []*UTXO{
							&UTXO{OutputID: bc.NewHash([32]byte{0x01})},
						},
						change: 9527,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			cancelRid: 3,
		},
	}

	for i, c := range cases {
		c.before.cancel(c.cancelRid)
		if !testutil.DeepEqual(c.before, c.after) {
			t.Errorf("case %d: got %v want %v", i, c.before, c.after)
		}
	}
}

func TestRemoveUnconfirmedUtxo(t *testing.T) {
	cases := []struct {
		before      utxoKeeper
		after       utxoKeeper
		removeUtxos []*bc.Hash
	}{
		{
			before: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{},
			},
			after: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{},
			},
			removeUtxos: []*bc.Hash{},
		},
		{
			before: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{
					bc.Hash{V0: 1}: &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
				},
			},
			after: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{},
			},
			removeUtxos: []*bc.Hash{
				&bc.Hash{V0: 1},
			},
		},
		{
			before: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{
					bc.Hash{V0: 1}: &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
					bc.Hash{V0: 2}: &UTXO{OutputID: bc.NewHash([32]byte{0x02})},
					bc.Hash{V0: 3}: &UTXO{OutputID: bc.NewHash([32]byte{0x03})},
					bc.Hash{V0: 4}: &UTXO{OutputID: bc.NewHash([32]byte{0x04})},
					bc.Hash{V0: 5}: &UTXO{OutputID: bc.NewHash([32]byte{0x05})},
				},
			},
			after: utxoKeeper{
				unconfirmed: map[bc.Hash]*UTXO{
					bc.Hash{V0: 2}: &UTXO{OutputID: bc.NewHash([32]byte{0x02})},
					bc.Hash{V0: 4}: &UTXO{OutputID: bc.NewHash([32]byte{0x04})},
				},
			},
			removeUtxos: []*bc.Hash{
				&bc.Hash{V0: 1},
				&bc.Hash{V0: 3},
				&bc.Hash{V0: 5},
			},
		},
	}

	for i, c := range cases {
		c.before.RemoveUnconfirmedUtxo(c.removeUtxos)
		if !testutil.DeepEqual(c.before, c.after) {
			t.Errorf("case %d: got %v want %v", i, c.before, c.after)
		}
	}
}

func TestReserve(t *testing.T) {
	currentHeight := func() uint64 { return 9527 }
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	accountStore := newMockAccountStore(testDB)

	cases := []struct {
		before        utxoKeeper
		after         utxoKeeper
		err           error
		reserveAmount uint64
		exp           time.Time
		vote          []byte
	}{
		{
			before: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				reserved:      map[bc.Hash]uint64{},
				reservations:  map[uint64]*reservation{},
			},
			after: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				reserved:      map[bc.Hash]uint64{},
				reservations:  map[uint64]*reservation{},
			},
			reserveAmount: 1,
			err:           ErrInsufficient,
		},
		{
			before: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			reserveAmount: 4,
			err:           ErrInsufficient,
		},
		{
			before: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:    bc.NewHash([32]byte{0x01}),
						AccountID:   "testAccount",
						Amount:      3,
						ValidHeight: 9528,
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:    bc.NewHash([32]byte{0x01}),
						AccountID:   "testAccount",
						Amount:      3,
						ValidHeight: 9528,
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			reserveAmount: 3,
			err:           ErrImmature,
		},
		{
			before: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 0,
				},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 0,
				},
				reservations: map[uint64]*reservation{},
			},
			reserveAmount: 3,
			err:           ErrReserved,
		},
		{
			before: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
				reservations: map[uint64]*reservation{
					1: &reservation{
						id: 1,
						utxos: []*UTXO{
							&UTXO{
								OutputID:  bc.NewHash([32]byte{0x01}),
								AccountID: "testAccount",
								Amount:    3,
							},
						},
						change: 1,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			reserveAmount: 2,
			err:           nil,
			exp:           time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			before: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				nextIndex:     1,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
					bc.NewHash([32]byte{0x02}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x02}),
						AccountID: "testAccount",
						Amount:    5,
					},
					bc.NewHash([32]byte{0x03}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x03}),
						AccountID: "testAccount",
						Amount:    7,
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
					bc.NewHash([32]byte{0x02}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x02}),
						AccountID: "testAccount",
						Amount:    5,
					},
					bc.NewHash([32]byte{0x03}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x03}),
						AccountID: "testAccount",
						Amount:    7,
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
					bc.NewHash([32]byte{0x02}): 2,
					bc.NewHash([32]byte{0x03}): 2,
				},
				reservations: map[uint64]*reservation{
					2: &reservation{
						id: 2,
						utxos: []*UTXO{
							&UTXO{
								OutputID:  bc.NewHash([32]byte{0x03}),
								AccountID: "testAccount",
								Amount:    7,
							},
							&UTXO{
								OutputID:  bc.NewHash([32]byte{0x02}),
								AccountID: "testAccount",
								Amount:    5,
							},
						},
						change: 4,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			reserveAmount: 8,
			err:           nil,
			exp:           time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
		},
		{
			before: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
						Vote:      []byte("af594006a40837d9f028daabb6d589df0b9138daefad5683e5233c2646279217294a8d532e60863bcf196625a35fb8ceeffa3c09610eb92dcfb655a947f13269"),
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
						Vote:      []byte("af594006a40837d9f028daabb6d589df0b9138daefad5683e5233c2646279217294a8d532e60863bcf196625a35fb8ceeffa3c09610eb92dcfb655a947f13269"),
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
				reservations: map[uint64]*reservation{
					1: &reservation{
						id: 1,
						utxos: []*UTXO{
							&UTXO{
								OutputID:  bc.NewHash([32]byte{0x01}),
								AccountID: "testAccount",
								Amount:    3,
								Vote:      []byte("af594006a40837d9f028daabb6d589df0b9138daefad5683e5233c2646279217294a8d532e60863bcf196625a35fb8ceeffa3c09610eb92dcfb655a947f13269"),
							},
						},
						change: 1,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
					},
				},
			},
			reserveAmount: 2,
			err:           nil,
			exp:           time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
			vote:          []byte("af594006a40837d9f028daabb6d589df0b9138daefad5683e5233c2646279217294a8d532e60863bcf196625a35fb8ceeffa3c09610eb92dcfb655a947f13269"),
		},
	}

	for i, c := range cases {
		if _, err := c.before.Reserve("testAccount", &bc.AssetID{}, c.reserveAmount, true, c.vote, c.exp); err != c.err {
			t.Errorf("case %d: got error %v want error %v", i, err, c.err)
		}
		checkUtxoKeeperEqual(t, i, &c.before, &c.after)
	}
}

func TestReserveParticular(t *testing.T) {
	currentHeight := func() uint64 { return 9527 }
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	accountStore := newMockAccountStore(testDB)

	cases := []struct {
		before      utxoKeeper
		after       utxoKeeper
		err         error
		reserveHash bc.Hash
		exp         time.Time
	}{
		{
			before: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 0,
				},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 0,
				},
				reservations: map[uint64]*reservation{},
			},
			reserveHash: bc.NewHash([32]byte{0x01}),
			err:         ErrReserved,
		},
		{
			before: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:    bc.NewHash([32]byte{0x01}),
						AccountID:   "testAccount",
						Amount:      3,
						ValidHeight: 9528,
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:    bc.NewHash([32]byte{0x01}),
						AccountID:   "testAccount",
						Amount:      3,
						ValidHeight: 9528,
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			reserveHash: bc.NewHash([32]byte{0x01}),
			err:         ErrImmature,
		},
		{
			before: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved:     map[bc.Hash]uint64{},
				reservations: map[uint64]*reservation{},
			},
			after: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
				reservations: map[uint64]*reservation{
					1: &reservation{
						id: 1,
						utxos: []*UTXO{
							&UTXO{
								OutputID:  bc.NewHash([32]byte{0x01}),
								AccountID: "testAccount",
								Amount:    3,
							},
						},
						change: 0,
						expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC),
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

func TestExpireReservation(t *testing.T) {
	before := &utxoKeeper{
		reservations: map[uint64]*reservation{
			1: &reservation{expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC)},
			2: &reservation{expiry: time.Date(3016, 8, 10, 0, 0, 0, 0, time.UTC)},
			3: &reservation{expiry: time.Date(2016, 8, 10, 0, 0, 0, 0, time.UTC)},
			4: &reservation{expiry: time.Date(3016, 8, 10, 0, 0, 0, 0, time.UTC)},
			5: &reservation{expiry: time.Date(3016, 8, 10, 0, 0, 0, 0, time.UTC)},
		},
	}
	after := &utxoKeeper{
		reservations: map[uint64]*reservation{
			2: &reservation{expiry: time.Date(3016, 8, 10, 0, 0, 0, 0, time.UTC)},
			4: &reservation{expiry: time.Date(3016, 8, 10, 0, 0, 0, 0, time.UTC)},
			5: &reservation{expiry: time.Date(3016, 8, 10, 0, 0, 0, 0, time.UTC)},
		},
	}
	before.expireReservation(time.Date(2017, 8, 10, 0, 0, 0, 0, time.UTC))
	checkUtxoKeeperEqual(t, 0, before, after)
}

func TestFindUtxos(t *testing.T) {
	currentHeight := func() uint64 { return 9527 }
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer func() {
		testDB.Close()
		os.RemoveAll("temp")
	}()

	accountStore := newMockAccountStore(testDB)

	cases := []struct {
		uk             utxoKeeper
		dbUtxos        []*UTXO
		useUnconfirmed bool
		wantUtxos      []*UTXO
		immatureAmount uint64
		vote           []byte
	}{
		{
			uk: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed:   map[bc.Hash]*UTXO{},
			},
			dbUtxos:        []*UTXO{},
			useUnconfirmed: true,
			wantUtxos:      []*UTXO{},
			immatureAmount: 0,
		},
		{
			uk: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed:   map[bc.Hash]*UTXO{},
			},
			dbUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x01}),
					AccountID: "testAccount",
					Amount:    3,
				},
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x02}),
					AccountID: "testAccount",
					AssetID:   bc.AssetID{V0: 6},
					Amount:    3,
				},
			},
			useUnconfirmed: false,
			wantUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x01}),
					AccountID: "testAccount",
					Amount:    3,
				},
			},
			immatureAmount: 0,
		},
		{
			uk: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed:   map[bc.Hash]*UTXO{},
			},
			dbUtxos: []*UTXO{
				&UTXO{
					OutputID:    bc.NewHash([32]byte{0x02}),
					AccountID:   "testAccount",
					Amount:      3,
					ValidHeight: 9528,
				},
			},
			useUnconfirmed: false,
			wantUtxos:      []*UTXO{},
			immatureAmount: 3,
		},
		{
			uk: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
			},
			dbUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x02}),
					AccountID: "testAccount",
					Amount:    3,
				},
			},
			useUnconfirmed: false,
			wantUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x02}),
					AccountID: "testAccount",
					Amount:    3,
				},
			},
			immatureAmount: 0,
		},
		{
			uk: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x11}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    3,
					},
				},
			},
			dbUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x02}),
					AccountID: "testAccount",
					Amount:    3,
				},
			},
			useUnconfirmed: true,
			wantUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x02}),
					AccountID: "testAccount",
					Amount:    3,
				},
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x01}),
					AccountID: "testAccount",
					Amount:    3,
				},
			},
			immatureAmount: 0,
		},
		{
			uk: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x01}),
						AccountID: "testAccount",
						Amount:    1,
					},
					bc.NewHash([32]byte{0x02}): &UTXO{
						OutputID:  bc.NewHash([32]byte{0x02}),
						AccountID: "notMe",
						Amount:    2,
					},
				},
			},
			dbUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x03}),
					AccountID: "testAccount",
					Amount:    3,
				},
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x04}),
					AccountID: "notMe",
					Amount:    4,
				},
			},
			useUnconfirmed: true,
			wantUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x03}),
					AccountID: "testAccount",
					Amount:    3,
				},
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x01}),
					AccountID: "testAccount",
					Amount:    1,
				},
			},
			immatureAmount: 0,
		},
		{
			uk: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed:   map[bc.Hash]*UTXO{},
			},
			dbUtxos: []*UTXO{
				&UTXO{
					OutputID:  bc.NewHash([32]byte{0x01}),
					AccountID: "testAccount",
					Amount:    6,
					Vote:      []byte("af594006a40837d9f028daabb6d589df0b9138daefad5683e5233c2646279217294a8d532e60863bcf196625a35fb8ceeffa3c09610eb92dcfb655a947f13269"),
				},
			},
			useUnconfirmed: false,
			wantUtxos: []*UTXO{
				&UTXO{
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
			if err := c.uk.store.SetStandardUTXO(u.OutputID, u); err != nil {
				t.Error(err)
			}
		}

		gotUtxos, immatureAmount := c.uk.findUtxos("testAccount", &bc.AssetID{}, c.useUnconfirmed, c.vote)
		if !testutil.DeepEqual(gotUtxos, c.wantUtxos) {
			t.Errorf("case %d: got %v want %v", i, gotUtxos, c.wantUtxos)
		}
		if immatureAmount != c.immatureAmount {
			t.Errorf("case %d: got %v want %v", i, immatureAmount, c.immatureAmount)
		}

		for _, u := range c.dbUtxos {
			c.uk.store.DeleteStandardUTXO(u.OutputID)
		}
	}
}

func TestFindUTXO(t *testing.T) {
	currentHeight := func() uint64 { return 9527 }
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	accountStore := newMockAccountStore(testDB)

	cases := []struct {
		uk             utxoKeeper
		dbUtxos        map[string]*UTXO
		outHash        bc.Hash
		useUnconfirmed bool
		wantUtxo       *UTXO
		err            error
	}{
		{
			uk: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed:   map[bc.Hash]*UTXO{},
			},
			dbUtxos:  map[string]*UTXO{},
			outHash:  bc.NewHash([32]byte{0x01}),
			wantUtxo: nil,
			err:      ErrMatchUTXO,
		},
		{
			uk: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
				},
			},
			dbUtxos:        map[string]*UTXO{},
			outHash:        bc.NewHash([32]byte{0x01}),
			wantUtxo:       nil,
			useUnconfirmed: false,
			err:            ErrMatchUTXO,
		},
		{
			uk: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed: map[bc.Hash]*UTXO{
					bc.NewHash([32]byte{0x01}): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
				},
			},
			dbUtxos:        map[string]*UTXO{},
			outHash:        bc.NewHash([32]byte{0x01}),
			wantUtxo:       &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			useUnconfirmed: true,
			err:            nil,
		},
		{
			uk: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed:   map[bc.Hash]*UTXO{},
			},
			dbUtxos: map[string]*UTXO{
				string(StandardUTXOKey(bc.NewHash([32]byte{0x01}))): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			},
			outHash:        bc.NewHash([32]byte{0x01}),
			wantUtxo:       &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			useUnconfirmed: false,
			err:            nil,
		},
		{
			uk: utxoKeeper{
				store:         accountStore,
				currentHeight: currentHeight,
				unconfirmed:   map[bc.Hash]*UTXO{},
			},
			dbUtxos: map[string]*UTXO{
				string(ContractUTXOKey(bc.NewHash([32]byte{0x01}))): &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
			},
			outHash:        bc.NewHash([32]byte{0x01}),
			wantUtxo:       &UTXO{OutputID: bc.NewHash([32]byte{0x01})},
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

		gotUtxo, err := c.uk.findUtxo(c.outHash, c.useUnconfirmed)
		if !testutil.DeepEqual(gotUtxo, c.wantUtxo) {
			t.Errorf("case %d: got %v want %v", i, gotUtxo, c.wantUtxo)
		}
		if err != c.err {
			t.Errorf("case %d: got %v want %v", i, err, c.err)
		}

		for _, u := range c.dbUtxos {
			c.uk.store.DeleteStandardUTXO(u.OutputID)
		}
	}
}

func TestOptUTXOs(t *testing.T) {
	cases := []struct {
		uk             utxoKeeper
		input          []*UTXO
		inputAmount    uint64
		wantUtxos      []*UTXO
		optAmount      uint64
		reservedAmount uint64
	}{
		{
			uk: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
			},
			input:          []*UTXO{},
			inputAmount:    13,
			wantUtxos:      []*UTXO{},
			optAmount:      0,
			reservedAmount: 0,
		},
		{
			uk: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
			},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
			},
			inputAmount:    13,
			wantUtxos:      []*UTXO{},
			optAmount:      0,
			reservedAmount: 1,
		},
		{
			uk: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
				},
			},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 3},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 5},
			},
			inputAmount: 13,
			wantUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 5},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 3},
			},
			optAmount:      8,
			reservedAmount: 1,
		},
		{
			uk: utxoKeeper{
				reserved: map[bc.Hash]uint64{
					bc.NewHash([32]byte{0x01}): 1,
					bc.NewHash([32]byte{0x02}): 2,
					bc.NewHash([32]byte{0x03}): 3,
				},
			},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 3},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 5},
			},
			inputAmount:    1,
			wantUtxos:      []*UTXO{},
			optAmount:      0,
			reservedAmount: 9,
		},
		{
			uk: utxoKeeper{},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 3},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 5},
			},
			inputAmount: 1,
			wantUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
			},
			optAmount:      1,
			reservedAmount: 0,
		},
		{
			uk: utxoKeeper{},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 2},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 3},
				&UTXO{OutputID: bc.NewHash([32]byte{0x05}), Amount: 5},
			},
			inputAmount: 5,
			wantUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 3},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 2},
			},
			optAmount:      5,
			reservedAmount: 0,
		},
		{
			uk: utxoKeeper{},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x04}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x05}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x06}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x08}), Amount: 6},
			},
			inputAmount: 6,
			wantUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x08}), Amount: 6},
			},
			optAmount:      6,
			reservedAmount: 0,
		},
		{
			uk: utxoKeeper{},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x04}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x05}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x06}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x08}), Amount: 6},
			},
			inputAmount: 5,
			wantUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x04}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x05}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x06}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
			},
			optAmount:      5,
			reservedAmount: 0,
		},
		{
			uk: utxoKeeper{},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 3},
				&UTXO{OutputID: bc.NewHash([32]byte{0x05}), Amount: 5},
				&UTXO{OutputID: bc.NewHash([32]byte{0x07}), Amount: 7},
				&UTXO{OutputID: bc.NewHash([32]byte{0x11}), Amount: 11},
				&UTXO{OutputID: bc.NewHash([32]byte{0x13}), Amount: 13},
				&UTXO{OutputID: bc.NewHash([32]byte{0x23}), Amount: 23},
				&UTXO{OutputID: bc.NewHash([32]byte{0x31}), Amount: 31},
			},
			inputAmount: 13,
			wantUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x07}), Amount: 7},
				&UTXO{OutputID: bc.NewHash([32]byte{0x05}), Amount: 5},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 3},
			},
			optAmount:      15,
			reservedAmount: 0,
		},
		{
			uk: utxoKeeper{},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
			},
			inputAmount: 1,
			wantUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
			},
			optAmount:      1,
			reservedAmount: 0,
		},
		{
			uk: utxoKeeper{},
			input: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 2},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 3},
				&UTXO{OutputID: bc.NewHash([32]byte{0x04}), Amount: 4},
				&UTXO{OutputID: bc.NewHash([32]byte{0x05}), Amount: 5},
				&UTXO{OutputID: bc.NewHash([32]byte{0x06}), Amount: 6},
				&UTXO{OutputID: bc.NewHash([32]byte{0x07}), Amount: 7},
				&UTXO{OutputID: bc.NewHash([32]byte{0x08}), Amount: 8},
				&UTXO{OutputID: bc.NewHash([32]byte{0x09}), Amount: 9},
				&UTXO{OutputID: bc.NewHash([32]byte{0x10}), Amount: 10},
				&UTXO{OutputID: bc.NewHash([32]byte{0x11}), Amount: 11},
				&UTXO{OutputID: bc.NewHash([32]byte{0x12}), Amount: 12},
			},
			inputAmount: 15,
			wantUtxos: []*UTXO{
				&UTXO{OutputID: bc.NewHash([32]byte{0x05}), Amount: 5},
				&UTXO{OutputID: bc.NewHash([32]byte{0x04}), Amount: 4},
				&UTXO{OutputID: bc.NewHash([32]byte{0x03}), Amount: 3},
				&UTXO{OutputID: bc.NewHash([32]byte{0x02}), Amount: 2},
				&UTXO{OutputID: bc.NewHash([32]byte{0x01}), Amount: 1},
			},
			optAmount:      15,
			reservedAmount: 0,
		},
	}

	for i, c := range cases {
		got, optAmount, reservedAmount := c.uk.optUTXOs(c.input, c.inputAmount)
		if !testutil.DeepEqual(got, c.wantUtxos) {
			t.Errorf("case %d: utxos got %v want %v", i, got, c.wantUtxos)
		}
		if optAmount != c.optAmount {
			t.Errorf("case %d: utxos got %v want %v", i, optAmount, c.optAmount)
		}
		if reservedAmount != c.reservedAmount {
			t.Errorf("case %d: reservedAmount got %v want %v", i, reservedAmount, c.reservedAmount)
		}
	}
}

func checkUtxoKeeperEqual(t *testing.T, i int, a, b *utxoKeeper) {
	if !testutil.DeepEqual(a.unconfirmed, b.unconfirmed) {
		t.Errorf("case %d: unconfirmed got %v want %v", i, a.unconfirmed, b.unconfirmed)
	}
	if !testutil.DeepEqual(a.reserved, b.reserved) {
		t.Errorf("case %d: reserved got %v want %v", i, a.reserved, b.reserved)
	}
	if !testutil.DeepEqual(a.reservations, b.reservations) {
		t.Errorf("case %d: reservations got %v want %v", i, a.reservations, b.reservations)
	}
}

type mockAccountStore struct {
	accountDB dbm.DB
	batch     dbm.Batch
}

// NewAccountStore create new AccountStore.
func newMockAccountStore(db dbm.DB) *mockAccountStore {
	return &mockAccountStore{
		accountDB: db,
		batch:     nil,
	}
}

var (
	UTXOPrefix  = []byte{0x00, 0x3a}
	SUTXOPrefix = []byte{0x01, 0x3a}
)

// StandardUTXOKey makes an account unspent outputs key to store
func StandardUTXOKey(id bc.Hash) []byte {
	name := id.String()
	return append(UTXOPrefix, []byte(name)...)
}

// ContractUTXOKey makes a smart contract unspent outputs key to store
func ContractUTXOKey(id bc.Hash) []byte {
	name := id.String()
	return append(SUTXOPrefix, []byte(name)...)
}

func (store *mockAccountStore) InitBatch() error                                { return nil }
func (store *mockAccountStore) CommitBatch() error                              { return nil }
func (store *mockAccountStore) DeleteAccount(*Account) error                    { return nil }
func (store *mockAccountStore) GetAccountByAlias(string) (*Account, error)      { return nil, nil }
func (store *mockAccountStore) GetAccountByID(string) (*Account, error)         { return nil, nil }
func (store *mockAccountStore) GetAccountIndex([]chainkd.XPub) uint64           { return 0 }
func (store *mockAccountStore) GetBip44ContractIndex(string, bool) uint64       { return 0 }
func (store *mockAccountStore) GetCoinbaseArbitrary() []byte                    { return nil }
func (store *mockAccountStore) GetContractIndex(string) uint64                  { return 0 }
func (store *mockAccountStore) GetControlProgram(bc.Hash) (*CtrlProgram, error) { return nil, nil }
func (store *mockAccountStore) GetMiningAddress() (*CtrlProgram, error)         { return nil, nil }
func (store *mockAccountStore) ListAccounts(string) ([]*Account, error)         { return nil, nil }
func (store *mockAccountStore) ListControlPrograms() ([]*CtrlProgram, error)    { return nil, nil }
func (store *mockAccountStore) SetAccount(*Account) error                       { return nil }
func (store *mockAccountStore) SetAccountIndex(*Account) error                  { return nil }
func (store *mockAccountStore) SetBip44ContractIndex(string, bool, uint64)      { return }
func (store *mockAccountStore) SetCoinbaseArbitrary([]byte)                     { return }
func (store *mockAccountStore) SetContractIndex(string, uint64)                 { return }
func (store *mockAccountStore) SetControlProgram(bc.Hash, *CtrlProgram) error   { return nil }
func (store *mockAccountStore) SetMiningAddress(*CtrlProgram) error             { return nil }

// DeleteStandardUTXO delete utxo by outpu id
func (store *mockAccountStore) DeleteStandardUTXO(outputID bc.Hash) {
	if store.batch == nil {
		store.accountDB.Delete(StandardUTXOKey(outputID))
	} else {
		store.batch.Delete(StandardUTXOKey(outputID))
	}
}

// GetUTXO get standard utxo by id
func (store *mockAccountStore) GetUTXO(outid bc.Hash) (*UTXO, error) {
	u := new(UTXO)
	if data := store.accountDB.Get(StandardUTXOKey(outid)); data != nil {
		return u, json.Unmarshal(data, u)
	}
	if data := store.accountDB.Get(ContractUTXOKey(outid)); data != nil {
		return u, json.Unmarshal(data, u)
	}
	return nil, ErrMatchUTXO
}

// ListUTXOs get utxos by accountID
func (store *mockAccountStore) ListUTXOs() []*UTXO {
	utxoIter := store.accountDB.IteratorPrefix([]byte(UTXOPrefix))
	defer utxoIter.Release()

	utxos := []*UTXO{}
	for utxoIter.Next() {
		utxo := new(UTXO)
		if err := json.Unmarshal(utxoIter.Value(), utxo); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("utxoKeeper findUtxos fail on unmarshal utxo")
			continue
		}
		utxos = append(utxos, utxo)
	}
	return utxos
}

// SetStandardUTXO set standard utxo
func (store *mockAccountStore) SetStandardUTXO(outputID bc.Hash, utxo *UTXO) error {
	data, err := json.Marshal(utxo)
	if err != nil {
		return err
	}
	if store.batch == nil {
		store.accountDB.Set(StandardUTXOKey(outputID), data)
	} else {
		store.batch.Set(StandardUTXOKey(outputID), data)
	}
	return nil
}

func mockAccountManager(t *testing.T) *Manager {
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "memdb", dirPath)
	accountStore := newMockAccountStore(testDB)

	// dispatcher := event.NewDispatcher()
	// // store := database.NewStore(testDB)
	// store := newMockStore(testDB)
	// // accountStore := database.NewAccountStore(testDB)
	// accountStore := newMockAccountStore(testDB)
	// txPool := protocol.NewTxPool(store, dispatcher)
	// config.CommonConfig = config.DefaultConfig()
	// chain, err := protocol.NewChain(store, txPool, dispatcher)
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// return NewManager(accountStore, chain)

	bestBlockHeight := func() uint64 { return 9527 }

	return &Manager{
		store:       accountStore,
		chain:       nil,
		utxoKeeper:  newUtxoKeeper(bestBlockHeight, accountStore),
		cache:       lru.New(maxAccountCache),
		aliasCache:  lru.New(maxAccountCache),
		delayedACPs: make(map[*txbuilder.TemplateBuilder][]*CtrlProgram),
	}
}

type mockStore struct {
	db dbm.DB
	// cache cache
}

// newStore creates and returns a new Store object.
func newMockStore(db dbm.DB) *mockStore {
	// fillBlockHeaderFn := func(hash *bc.Hash) (*types.BlockHeader, error) {
	// 	return GetBlockHeader(db, hash)
	// }
	// fillBlockTxsFn := func(hash *bc.Hash) ([]*types.Tx, error) {
	// 	return GetBlockTransactions(db, hash)
	// }

	// fillBlockHashesFn := func(height uint64) ([]*bc.Hash, error) {
	// 	return GetBlockHashesByHeight(db, height)
	// }

	// fillMainChainHashFn := func(height uint64) (*bc.Hash, error) {
	// 	return GetMainChainHash(db, height)
	// }

	// fillVoteResultFn := func(seq uint64) (*state.VoteResult, error) {
	// 	return GetVoteResult(db, seq)
	// }

	// cache := newCache(fillBlockHeaderFn, fillBlockTxsFn, fillBlockHashesFn, fillMainChainHashFn, fillVoteResultFn)
	return &mockStore{
		db: db,
		// cache: nil,
	}
}

func (s *mockStore) BlockExist(*bc.Hash) bool                                     { return false }
func (s *mockStore) GetBlock(*bc.Hash) (*types.Block, error)                      { return nil, nil }
func (s *mockStore) GetBlockHeader(*bc.Hash) (*types.BlockHeader, error)          { return nil, nil }
func (s *mockStore) GetStoreStatus() *protocol.BlockStoreState                    { return nil }
func (s *mockStore) GetTransactionStatus(*bc.Hash) (*bc.TransactionStatus, error) { return nil, nil }
func (s *mockStore) GetTransactionsUtxo(*state.UtxoViewpoint, []*bc.Tx) error     { return nil }
func (s *mockStore) GetUtxo(*bc.Hash) (*storage.UtxoEntry, error)                 { return nil, nil }
func (s *mockStore) GetVoteResult(uint64) (*state.VoteResult, error)              { return nil, nil }
func (s *mockStore) GetMainChainHash(uint64) (*bc.Hash, error)                    { return nil, nil }
func (s *mockStore) GetBlockHashesByHeight(uint64) ([]*bc.Hash, error)            { return nil, nil }
func (s *mockStore) SaveBlock(*types.Block, *bc.TransactionStatus) error          { return nil }
func (s *mockStore) SaveBlockHeader(*types.BlockHeader) error                     { return nil }
func (s *mockStore) SaveChainStatus(*types.BlockHeader, *types.BlockHeader, []*types.BlockHeader, *state.UtxoViewpoint, []*state.VoteResult) error {
	return nil
}
