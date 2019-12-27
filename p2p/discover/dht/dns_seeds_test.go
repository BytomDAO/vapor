package dht

import (
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/bytom/vapor/consensus"
)

var testnetAddr = []string{"1.2.3.4", "5.6.7.8"}
var mainnetAddr = []string{"11.22.33.44", "55.66.77.88"}
var errAddr = []string{"a.b.ab.abc", "55.66.77.88"}

func lookupHostNormal(host string) ([]string, error) {
	switch host {
	case consensus.MainNetParams.DNSSeeds[0]:
		return mainnetAddr, nil
	case consensus.TestNetParams.DNSSeeds[0]:
		return testnetAddr, nil
	}
	return nil, nil
}

func lookupHostErrIP(host string) ([]string, error) {
	return errAddr, nil
}

func TestQueryDNSSeeds(t *testing.T) {
	testCases := []struct {
		chainID    string
		lookupHost func(host string) (addrs []string, err error)
		wantErr    error
		wantAddr   []string
	}{
		{
			chainID:    "testnet",
			lookupHost: lookupHostNormal,
			wantErr:    nil,
			wantAddr:   []string{"1.2.3.4:56657", "5.6.7.8:56657"},
		},
		{
			chainID:    "mainnet",
			lookupHost: lookupHostNormal,
			wantErr:    nil,
			wantAddr:   []string{"11.22.33.44:56656", "55.66.77.88:56656"},
		},
		{
			chainID:    "solonet",
			lookupHost: lookupHostNormal,
			wantErr:    errDNSSeedsEmpty,
			wantAddr:   nil,
		},
		{
			chainID:    "testnet",
			lookupHost: lookupHostErrIP,
			wantErr:    errDNSTimeout,
			wantAddr:   nil,
		},
	}

	for i, tc := range testCases {
		consensus.ActiveNetParams = consensus.NetParams[tc.chainID]
		addresses, err := QueryDNSSeeds(tc.lookupHost)
		if err != tc.wantErr {
			t.Fatalf("test %d: error mismatch for query dns seed got %q want %q", i, err, tc.wantErr)
		}

		if !reflect.DeepEqual(addresses, tc.wantAddr) {
			t.Fatalf("test %d: result mismatch for query dns seed got %s want %s", i, spew.Sdump(addresses), spew.Sdump(tc.wantAddr))
		}
	}
}
