package test

import (
	"testing"

	mock "github.com/vapor/test/mock"
	"github.com/vapor/testutil"
)

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
