package asset

import (
	"context"
	"io/ioutil"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	dbm "github.com/tendermint/tmlibs/db"

	"github.com/vapor/common"
	"github.com/vapor/config"
	"github.com/vapor/consensus"
	engine "github.com/vapor/consensus/consensus"
	"github.com/vapor/consensus/consensus/dpos"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/database/leveldb"
	"github.com/vapor/protocol"
	"github.com/vapor/testutil"
)

func TestDefineAssetWithLowercase(t *testing.T) {
	reg := mockNewRegistry(t)
	alias := "lower"
	asset, err := reg.Define([]chainkd.XPub{testutil.TestXPub}, 1, nil, alias, nil)
	if err != nil {
		t.Fatal(err)
	}
	if *asset.Alias != strings.ToUpper(alias) {
		t.Fatal("created asset alias should be uppercase")
	}
}

func TestDefineAssetWithSpaceTrimed(t *testing.T) {
	reg := mockNewRegistry(t)
	alias := " WITH SPACE "
	asset, err := reg.Define([]chainkd.XPub{testutil.TestXPub}, 1, nil, alias, nil)
	if err != nil {
		t.Fatal(err)
	}
	if *asset.Alias != strings.TrimSpace(alias) {
		t.Fatal("created asset alias should be uppercase")
	}
}

func TestDefineAsset(t *testing.T) {
	ctx := context.Background()
	reg := mockNewRegistry(t)
	asset, err := reg.Define([]chainkd.XPub{testutil.TestXPub}, 1, nil, "asset-alias", nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	found, err := reg.FindByID(ctx, &asset.AssetID)
	if err != nil {
		t.Errorf("unexpected error %v", err)
	}

	if !testutil.DeepEqual(asset, found) {
		t.Errorf("expected asset %v to be recorded as %v", asset, found)
	}
}

func TestDefineBtmAsset(t *testing.T) {
	reg := mockNewRegistry(t)
	_, err := reg.Define([]chainkd.XPub{testutil.TestXPub}, 1, nil, consensus.BTMAlias, nil)
	if err == nil {
		testutil.FatalErr(t, err)
	}
}

func TestFindAssetByID(t *testing.T) {
	ctx := context.Background()
	reg := mockNewRegistry(t)
	keys := []chainkd.XPub{testutil.TestXPub}
	asset, err := reg.Define(keys, 1, nil, "TESTASSET", nil)
	if err != nil {
		testutil.FatalErr(t, err)

	}
	found, err := reg.FindByID(ctx, &asset.AssetID)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if !testutil.DeepEqual(asset, found) {
		t.Errorf("expected %v and %v to match", asset, found)
	}
}

func TestUpdateAssetAlias(t *testing.T) {
	reg := mockNewRegistry(t)

	oldAlias := "OLD_ALIAS"
	newAlias := "NEW_ALIAS"

	asset, err := reg.Define([]chainkd.XPub{testutil.TestXPub}, 1, nil, oldAlias, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	if reg.UpdateAssetAlias(asset.AssetID.String(), newAlias) != nil {
		testutil.FatalErr(t, err)
	}

	asset1, err := reg.FindByAlias(newAlias)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	gotAlias := *asset1.Alias
	if !reflect.DeepEqual(gotAlias, newAlias) {
		t.Fatalf("alias:\ngot:  %v\nwant: %v", gotAlias, newAlias)
	}
}

type SortByAssetsAlias []*Asset

func (a SortByAssetsAlias) Len() int { return len(a) }
func (a SortByAssetsAlias) Less(i, j int) bool {
	return strings.Compare(*a[i].Alias, *a[j].Alias) <= 0
}
func (a SortByAssetsAlias) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func TestListAssets(t *testing.T) {
	reg := mockNewRegistry(t)

	firstAlias := "FIRST_ALIAS"
	secondAlias := "SECOND_ALIAS"

	firstAsset, err := reg.Define([]chainkd.XPub{testutil.TestXPub}, 1, nil, firstAlias, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	secondAsset, err := reg.Define([]chainkd.XPub{testutil.TestXPub}, 1, nil, secondAlias, nil)
	if err != nil {
		testutil.FatalErr(t, err)
	}

	wantAssets := []*Asset{DefaultNativeAsset, firstAsset, secondAsset}

	gotAssets, err := reg.ListAssets("")
	if err != nil {
		testutil.FatalErr(t, err)
	}
	sort.Sort(SortByAssetsAlias(wantAssets))
	sort.Sort(SortByAssetsAlias(gotAssets))
	if !testutil.DeepEqual(gotAssets, wantAssets) {
		t.Fatalf("got:\ngot:  %v\nwant: %v", gotAssets, wantAssets)
	}
}

func mockChain(testDB dbm.DB) (*protocol.Chain, error) {
	var engine engine.Engine
	switch config.CommonConfig.Consensus.Type {
	case "dpos":
		engine = dpos.GDpos
	}
	store := leveldb.NewStore(testDB)
	txPool := protocol.NewTxPool(store)
	chain, err := protocol.NewChain(store, txPool, engine)
	if err != nil {
		return nil, err
	}
	return chain, nil
}

func mockNewRegistry(t *testing.T) *Registry {
	config.CommonConfig = config.DefaultConfig()
	consensus.SoloNetParams.Signer = "78673764e0ba91a4c5ba9ec0c8c23c69e3d73bf27970e05e0a977e81e13bde475264d3b177a96646bc0ce517ae7fd63504c183ab6d330dea184331a4cf5912d5"
	config.CommonConfig.Consensus.SelfVoteSigners = append(config.CommonConfig.Consensus.SelfVoteSigners, "vsm1qkm743xmgnvh84pmjchq2s4tnfpgu9ae2f9slep")
	config.CommonConfig.Consensus.XPrv = "a8e281b615809046698fb0b0f2804a36d824d48fa443350f10f1b80649d39e5f1e85cf9855548915e36137345910606cbc8e7dd8497c831dce899ee6ac112445"
	for _, v := range config.CommonConfig.Consensus.SelfVoteSigners {
		address, err := common.DecodeAddress(v, &consensus.SoloNetParams)
		if err != nil {
			t.Fatal(err)
		}
		config.CommonConfig.Consensus.Signers = append(config.CommonConfig.Consensus.Signers, address)
	}
	dirPath, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dirPath)

	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	defer os.RemoveAll("temp")

	chain, err := mockChain(testDB)
	if err != nil {
		t.Fatal(err)
	}

	return NewRegistry(testDB, chain)
}
