package account

import (
	"testing"
	"time"

	"github.com/vapor/protocol/bc"
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
