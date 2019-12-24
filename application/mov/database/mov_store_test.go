package database

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bytom/vapor/application/mov/common"
	"github.com/bytom/vapor/database/leveldb"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/testutil"
)

var (
	assetID1 = &bc.AssetID{V0: 1}
	assetID2 = &bc.AssetID{V0: 2}
	assetID3 = &bc.AssetID{V0: 3}
	assetID4 = &bc.AssetID{V0: 4}
	assetID5 = &bc.AssetID{V0: 5}
	assetID6 = &bc.AssetID{V0: 6}
	assetID7 = &bc.AssetID{V0: 7}
	assetID8 = &bc.AssetID{V0: 8}

	mockOrders = []*common.Order{
		&common.Order{
			FromAssetID:      assetID1,
			ToAssetID:        assetID2,
			RatioNumerator:   100090,
			RatioDenominator: 100000,
			Utxo: &common.MovUtxo{
				SourceID:       &bc.Hash{V0: 21},
				Amount:         1,
				SourcePos:      0,
				ControlProgram: []byte("aa"),
			},
		},
		&common.Order{
			FromAssetID:      assetID1,
			ToAssetID:        assetID2,
			RatioNumerator:   90,
			RatioDenominator: 100000,
			Utxo: &common.MovUtxo{
				SourceID:       &bc.Hash{V0: 22},
				Amount:         1,
				SourcePos:      0,
				ControlProgram: []byte("aa"),
			},
		},
		&common.Order{
			FromAssetID:      assetID1,
			ToAssetID:        assetID2,
			RatioNumerator:   97,
			RatioDenominator: 100000,
			Utxo: &common.MovUtxo{
				SourceID:       &bc.Hash{V0: 23},
				Amount:         1,
				SourcePos:      0,
				ControlProgram: []byte("aa"),
			},
		},
		&common.Order{
			FromAssetID:      assetID1,
			ToAssetID:        assetID2,
			RatioNumerator:   98,
			RatioDenominator: 100000,
			Utxo: &common.MovUtxo{
				SourceID:       &bc.Hash{V0: 13},
				Amount:         1,
				SourcePos:      0,
				ControlProgram: []byte("aa"),
			},
		},
		&common.Order{
			FromAssetID:      assetID1,
			ToAssetID:        assetID2,
			RatioNumerator:   98,
			RatioDenominator: 100000,
			Utxo: &common.MovUtxo{
				SourceID:       &bc.Hash{V0: 24},
				Amount:         10,
				SourcePos:      1,
				ControlProgram: []byte("aa"),
			},
		},
		&common.Order{
			FromAssetID:      assetID1,
			ToAssetID:        assetID2,
			RatioNumerator:   99,
			RatioDenominator: 100000,
			Utxo: &common.MovUtxo{
				SourceID:       &bc.Hash{V0: 24},
				Amount:         1,
				SourcePos:      0,
				ControlProgram: []byte("aa"),
			},
		},
		&common.Order{
			FromAssetID:      assetID1,
			ToAssetID:        assetID2,
			RatioNumerator:   96,
			RatioDenominator: 100000,
			Utxo: &common.MovUtxo{
				SourceID:       &bc.Hash{V0: 25},
				Amount:         1,
				SourcePos:      0,
				ControlProgram: []byte("aa"),
			},
		},
		&common.Order{
			FromAssetID:      assetID1,
			ToAssetID:        assetID2,
			RatioNumerator:   95,
			RatioDenominator: 100000,
			Utxo: &common.MovUtxo{
				SourceID:       &bc.Hash{V0: 26},
				Amount:         1,
				SourcePos:      0,
				ControlProgram: []byte("aa"),
			},
		},
		&common.Order{
			FromAssetID:      assetID1,
			ToAssetID:        assetID2,
			RatioNumerator:   90,
			RatioDenominator: 100000,
			Utxo: &common.MovUtxo{
				SourceID:       &bc.Hash{V0: 1},
				Amount:         1,
				SourcePos:      0,
				ControlProgram: []byte("aa"),
			},
		},
		&common.Order{
			FromAssetID:      assetID1,
			ToAssetID:        assetID2,
			RatioNumerator:   90,
			RatioDenominator: 100000,
			Utxo: &common.MovUtxo{
				SourceID:       &bc.Hash{V0: 2},
				Amount:         1,
				SourcePos:      0,
				ControlProgram: []byte("aa"),
			},
		},
		&common.Order{
			FromAssetID:      assetID3,
			ToAssetID:        assetID2,
			RatioNumerator:   96,
			RatioDenominator: 100000,
			Utxo: &common.MovUtxo{
				SourceID:       &bc.Hash{V0: 33},
				Amount:         1,
				SourcePos:      0,
				ControlProgram: []byte("aa"),
			},
		},
		&common.Order{
			FromAssetID:      assetID4,
			ToAssetID:        assetID2,
			RatioNumerator:   95,
			RatioDenominator: 100000,
			Utxo: &common.MovUtxo{
				SourceID:       &bc.Hash{V0: 34},
				Amount:         1,
				SourcePos:      0,
				ControlProgram: []byte("aa"),
			},
		},
		&common.Order{
			FromAssetID:      assetID4,
			ToAssetID:        assetID2,
			RatioNumerator:   96,
			RatioDenominator: 100000,
			Utxo: &common.MovUtxo{
				SourceID:       &bc.Hash{V0: 36},
				Amount:         1,
				SourcePos:      0,
				ControlProgram: []byte("aa"),
			},
		},
		&common.Order{
			FromAssetID:      assetID5,
			ToAssetID:        assetID2,
			RatioNumerator:   96,
			RatioDenominator: 100000,
			Utxo: &common.MovUtxo{
				SourceID:       &bc.Hash{V0: 37},
				Amount:         1,
				SourcePos:      0,
				ControlProgram: []byte("aa"),
			},
		},
		&common.Order{
			FromAssetID:      assetID6,
			ToAssetID:        assetID2,
			RatioNumerator:   98,
			RatioDenominator: 100000,
			Utxo: &common.MovUtxo{
				SourceID:       &bc.Hash{V0: 38},
				Amount:         1,
				SourcePos:      0,
				ControlProgram: []byte("aa"),
			},
		},
	}
)

func TestGetAssetIDFromTradePairKey(t *testing.T) {
	b := calcTradePairKey(assetID1, assetID2)
	gotA := getAssetIDFromTradePairKey(b, fromAssetIDPos)
	gotB := getAssetIDFromTradePairKey(b, toAssetIDPos)

	if *gotA != *assetID1 {
		t.Fatalf("got wrong from asset id got %s, want %s", gotA.String(), assetID1.String())
	}

	if *gotB != *assetID2 {
		t.Fatalf("got wrong to asset id got %s, want %s", gotB.String(), assetID2.String())
	}
}

func TestSortOrderKey(t *testing.T) {
	dirname, err := ioutil.TempDir("", "db_common_test")
	require.Nil(t, err)

	db, err := leveldb.NewGoLevelDB("testdb", dirname)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		db.Close()
		os.RemoveAll(dirname)
	}()

	type expectedData struct {
		rate     float64
		utxoHash string
	}

	cases := []struct {
		orders []*common.Order
		want   []expectedData
	}{
		{
			orders: []*common.Order{
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   100090,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   90,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   97,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   98,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   98,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   98,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   98,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   98,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 27},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   99,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   96,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   95,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   91,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   92,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 27},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   93,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 28},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   94,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 29},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   77,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 30},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   88,
					RatioDenominator: 100000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 31},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   9999999521,
					RatioDenominator: 10000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 32},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID:      &bc.AssetID{V0: 1},
					ToAssetID:        &bc.AssetID{V0: 0},
					RatioNumerator:   8888887954,
					RatioDenominator: 10000,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 33},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			want: []expectedData{
				expectedData{
					rate:     0.00077,
					utxoHash: "f1b85307cf1f4eb6b193b6fc289413fdbb12bc362ced399762589b016e54dd02",
				},
				expectedData{
					rate:     0.00088,
					utxoHash: "49ef60af0f24962ed129a73142048ed0cb589041c629353932e3c3e0a4e822ba",
				},
				expectedData{
					rate:     0.00090,
					utxoHash: "67b2ac6ea71b271e72836e162811866f291ed2fab106c43519ca0c94ef8a5dce",
				},
				expectedData{
					rate:     0.00091,
					utxoHash: "47ff45b7b530512142981c2cee82faad63d6c9e7ffed0e72c3e42668f13b296f",
				},
				expectedData{
					rate:     0.00092,
					utxoHash: "b750d0b95f38043362c8335f242f97cfd3e1cada8fd171b914471a16cc0f14c6",
				},
				expectedData{
					rate:     0.00093,
					utxoHash: "04386ef57f0ca1be0a9be46c413900adbc0ab1e90e773959924aa73ca62edf64",
				},
				expectedData{
					rate:     0.00094,
					utxoHash: "c0fe6227c50da350a5e7b4ff85c18e9c901c323521067b9142acd128cf13ae82",
				},
				expectedData{
					rate:     0.00095,
					utxoHash: "47ff45b7b530512142981c2cee82faad63d6c9e7ffed0e72c3e42668f13b296f",
				},
				expectedData{
					rate:     0.00096,
					utxoHash: "bc92df1cbd20c98b0d18c9d93422a770849235867522a08e492196d16ed0a422",
				},
				expectedData{
					rate:     0.00097,
					utxoHash: "0cc0ded6fb337a3c5e6e4d008d6167dc58bdede43713898e914d65cda3b8499a",
				},
				expectedData{
					rate:     0.00098,
					utxoHash: "14b51a6103f75d9cacdf0f9551467588c687ed3b029e25c646d276720569e227",
				},
				expectedData{
					rate:     0.00098,
					utxoHash: "1fa9fae83d0a5401a4e92f80636966486e763eecca588aa11dff02b415320602",
				},
				expectedData{
					rate:     0.00098,
					utxoHash: "6687d18ddbe4e7381a844e393ca3032a412285c9da6988eff182106e28ba09ca",
				},
				expectedData{
					rate:     0.00098,
					utxoHash: "841b1de7c871dfe6e2d1886809d9ae12ec45e570233b03879305232b096fda43",
				},
				expectedData{
					rate:     0.00098,
					utxoHash: "a4bc534c267d35a9eafc25cd66e0cb270a2537a51186605b7f7591bc567ab4c6",
				},
				expectedData{
					rate:     0.00099,
					utxoHash: "fdedf4117def659e07cc8a8ca318d21ae577a05e1a0197844b54d493bdae5854",
				},
				expectedData{
					rate:     1.0009,
					utxoHash: "20be3bd2d406bb7fe6627b32768fb2073e997b962a4badfa4384210fed2ab9c6",
				},
				expectedData{
					rate:     888888.7954,
					utxoHash: "72192f56b9525c74c6a9f0419563bc0da76b0f3d6e89d9decdb6e67786ac3909",
				},
				expectedData{
					rate:     999999.9521,
					utxoHash: "7886844334659b4feffc41528cf81192925d3aa4a5ccb3652200b9073b7d47c3",
				},
			},
		},
	}

	for i, c := range cases {
		for _, order := range c.orders {
			key := calcOrderKey(order.FromAssetID, order.ToAssetID, order.UTXOHash(), order.Rate())
			data, err := json.Marshal(order.Utxo)
			if err != nil {
				t.Fatal(err)
			}

			db.SetSync(key, data)
		}

		got := []expectedData{}
		itr := db.IteratorPrefixWithStart(nil, nil, false)
		for itr.Next() {
			key := itr.Key()
			pos := len(ordersPrefix) + assetIDLen*2
			b := [32]byte{}
			copy(b[:], key[pos+8:])
			utxoHash := bc.NewHash(b)

			got = append(got, expectedData{
				rate:     getRateFromOrderKey(key),
				utxoHash: utxoHash.String(),
			})
		}
		itr.Release()

		if !testutil.DeepEqual(c.want, got) {
			t.Errorf("case %v: got recovery status, got: %v, want: %v.", i, got, c.want)
		}
	}
}

func TestMovStore(t *testing.T) {
	cases := []struct {
		desc             string
		beforeOrders     []*common.Order
		beforeTradePairs []*common.TradePair
		beforeDBStatus   *common.MovDatabaseState
		addOrders        []*common.Order
		delOrders        []*common.Order
		blockHeader      *types.BlockHeader
		wantOrders       []*common.Order
		wantTradePairs   []*common.TradePair
		wantDBState      *common.MovDatabaseState
	}{
		{
			desc: "add order",
			addOrders: []*common.Order{
				mockOrders[0],
				mockOrders[1],
				mockOrders[2],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
				mockOrders[6],
				mockOrders[7],
			},
			blockHeader: &types.BlockHeader{Height: 1, PreviousBlockHash: bc.Hash{V0: 524821139490765641, V1: 2484214155808702787, V2: 9108473449351508820, V3: 7972721253564512122}},
			wantOrders: []*common.Order{
				mockOrders[1],
				mockOrders[7],
				mockOrders[6],
				mockOrders[2],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
				mockOrders[0],
			},
			wantTradePairs: []*common.TradePair{
				&common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2, Count: 8},
			},
			wantDBState: &common.MovDatabaseState{Height: 1, Hash: &bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
		},
		{
			desc: "del some order",
			beforeOrders: []*common.Order{
				mockOrders[0],
				mockOrders[1],
				mockOrders[2],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
				mockOrders[6],
				mockOrders[7],
			},
			beforeTradePairs: []*common.TradePair{
				&common.TradePair{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Count:       8,
				},
			},
			beforeDBStatus: &common.MovDatabaseState{Height: 1, Hash: &bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
			delOrders: []*common.Order{
				mockOrders[0],
				mockOrders[1],
				mockOrders[2],
			},
			blockHeader: &types.BlockHeader{Height: 2, PreviousBlockHash: bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
			wantOrders: []*common.Order{
				mockOrders[7],
				mockOrders[6],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
			},
			wantTradePairs: []*common.TradePair{
				&common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2, Count: 5},
			},
			wantDBState: &common.MovDatabaseState{Height: 2, Hash: &bc.Hash{V0: 3724755213446347384, V1: 158878632373345042, V2: 18283800951484248781, V3: 7520797730449067221}},
		},
		{
			desc: "del all order",
			beforeOrders: []*common.Order{
				mockOrders[0],
				mockOrders[1],
				mockOrders[2],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
				mockOrders[6],
				mockOrders[7],
			},
			beforeTradePairs: []*common.TradePair{
				&common.TradePair{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Count:       8,
				},
			},
			beforeDBStatus: &common.MovDatabaseState{Height: 1, Hash: &bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
			delOrders: []*common.Order{
				mockOrders[0],
				mockOrders[1],
				mockOrders[2],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
				mockOrders[6],
				mockOrders[7],
			},
			blockHeader:    &types.BlockHeader{Height: 2, PreviousBlockHash: bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
			wantOrders:     []*common.Order{},
			wantTradePairs: []*common.TradePair{},
			wantDBState:    &common.MovDatabaseState{Height: 2, Hash: &bc.Hash{V0: 3724755213446347384, V1: 158878632373345042, V2: 18283800951484248781, V3: 7520797730449067221}},
		},
		{
			desc: "Add and delete the same trade pair", // Add and delete different transaction pairs
			beforeOrders: []*common.Order{
				mockOrders[0],
				mockOrders[1],
				mockOrders[2],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
				mockOrders[6],
				mockOrders[7],
			},
			beforeTradePairs: []*common.TradePair{
				&common.TradePair{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Count:       8,
				},
			},
			beforeDBStatus: &common.MovDatabaseState{Height: 1, Hash: &bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
			addOrders: []*common.Order{
				mockOrders[8],
				mockOrders[9],
			},
			delOrders: []*common.Order{
				mockOrders[0],
				mockOrders[1],
				mockOrders[2],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
				mockOrders[6],
				mockOrders[7],
			},
			blockHeader: &types.BlockHeader{Height: 2, PreviousBlockHash: bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
			wantOrders: []*common.Order{
				mockOrders[9],
				mockOrders[8],
			},
			wantTradePairs: []*common.TradePair{
				&common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2, Count: 2},
			},
			wantDBState: &common.MovDatabaseState{Height: 2, Hash: &bc.Hash{V0: 3724755213446347384, V1: 158878632373345042, V2: 18283800951484248781, V3: 7520797730449067221}},
		},
		{
			desc: "Add and delete different transaction pairs",
			beforeOrders: []*common.Order{
				mockOrders[0],
				mockOrders[1],
				mockOrders[2],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
				mockOrders[6],
				mockOrders[7],
				mockOrders[10],
				mockOrders[11],
			},
			beforeTradePairs: []*common.TradePair{
				&common.TradePair{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Count:       8,
				},
				&common.TradePair{
					FromAssetID: assetID3,
					ToAssetID:   assetID2,
					Count:       1,
				},
				&common.TradePair{
					FromAssetID: assetID4,
					ToAssetID:   assetID2,
					Count:       1,
				},
			},
			beforeDBStatus: &common.MovDatabaseState{Height: 1, Hash: &bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
			addOrders: []*common.Order{
				mockOrders[12],
				mockOrders[13],
				mockOrders[14],
			},
			delOrders: []*common.Order{
				mockOrders[0],
				mockOrders[1],
				mockOrders[2],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
				mockOrders[6],
				mockOrders[7],
				mockOrders[10],
			},
			blockHeader: &types.BlockHeader{Height: 2, PreviousBlockHash: bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
			wantOrders: []*common.Order{
				mockOrders[11],
				mockOrders[12],
				mockOrders[13],
				mockOrders[14],
			},
			wantTradePairs: []*common.TradePair{
				&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID2, Count: 2},
				&common.TradePair{FromAssetID: assetID5, ToAssetID: assetID2, Count: 1},
				&common.TradePair{FromAssetID: assetID6, ToAssetID: assetID2, Count: 1},
			},
			wantDBState: &common.MovDatabaseState{Height: 2, Hash: &bc.Hash{V0: 3724755213446347384, V1: 158878632373345042, V2: 18283800951484248781, V3: 7520797730449067221}},
		},
	}

	initBlockHeader := &types.BlockHeader{
		Height:  0,
		Version: 1,
	}

	height := initBlockHeader.Height
	hash := initBlockHeader.Hash()

	defer os.RemoveAll("temp")
	for i, c := range cases {
		testDB := dbm.NewDB("testdb", "leveldb", "temp")
		movStore := NewLevelDBMovStore(testDB)
		if err := movStore.InitDBState(height, &hash); err != nil {
			t.Fatalf("case %d: InitDBState error %v.", i, err)
		}

		batch := movStore.db.NewBatch()
		tradePairsCnt := make(map[string]*common.TradePair)
		movStore.addOrders(batch, c.beforeOrders, tradePairsCnt)
		if len(c.beforeOrders) > 0 {
			tradePairsCnt = make(map[string]*common.TradePair)
			for _, tradePair := range c.beforeTradePairs {
				tradePairsCnt[tradePair.Key()] = tradePair
			}
			movStore.updateTradePairs(batch, tradePairsCnt)
			movStore.saveMovDatabaseState(batch, c.beforeDBStatus)
		}
		batch.Write()

		if err := movStore.ProcessOrders(c.addOrders, c.delOrders, c.blockHeader); err != nil {
			t.Fatalf("case %d: ProcessOrders error %v.", i, err)
		}

		var gotOrders []*common.Order

		tmp, err := movStore.ListOrders(&common.Order{FromAssetID: assetID1, ToAssetID: assetID2, RatioNumerator: 0, RatioDenominator: 1})
		if err != nil {
			t.Fatalf("case %d: ListOrders(assetID1 and assetID2) error %v.", i, err)
		}

		gotOrders = append(gotOrders, tmp...)

		tmp, err = movStore.ListOrders(&common.Order{FromAssetID: assetID3, ToAssetID: assetID2, RatioNumerator: 0, RatioDenominator: 1})
		if err != nil {
			t.Fatalf("case %d: ListOrders(assetID3 and assetID2)  error %v.", i, err)
		}

		gotOrders = append(gotOrders, tmp...)

		tmp, err = movStore.ListOrders(&common.Order{FromAssetID: assetID4, ToAssetID: assetID2, RatioNumerator: 0, RatioDenominator: 1})
		if err != nil {
			t.Fatalf("case %d: ListOrders(assetID4 and assetID2)  error %v.", i, err)
		}

		gotOrders = append(gotOrders, tmp...)

		tmp, err = movStore.ListOrders(&common.Order{FromAssetID: assetID5, ToAssetID: assetID2, RatioNumerator: 0, RatioDenominator: 1})
		if err != nil {
			t.Fatalf("case %d: ListOrders(assetID5 and assetID2)  error %v.", i, err)
		}

		gotOrders = append(gotOrders, tmp...)

		tmp, err = movStore.ListOrders(&common.Order{FromAssetID: assetID6, ToAssetID: assetID2, RatioNumerator: 0, RatioDenominator: 1})
		if err != nil {
			t.Fatalf("case %d: ListOrders(assetID6 and assetID2)  error %v.", i, err)
		}

		gotOrders = append(gotOrders, tmp...)

		if !testutil.DeepEqual(gotOrders, c.wantOrders) {
			t.Fatalf("case %d: got orders , gotOrders: %v, wantOrders: %v.", i, gotOrders, c.wantOrders)
		}

		gotTradePairs, err := movStore.ListTradePairsWithStart(nil, nil)
		if err != nil {
			t.Fatalf("case %d: ListTradePairsWithStart error %v.", i, err)
		}

		if !testutil.DeepEqual(gotTradePairs, c.wantTradePairs) {
			t.Fatalf("case %d: got tradePairs, gotTradePairs: %v, wantTradePairs: %v.", i, gotTradePairs, c.wantTradePairs)
		}

		gotDBState, err := movStore.GetMovDatabaseState()
		if err != nil {
			t.Fatalf("case %d: GetMovDatabaseState error %v.", i, err)
		}

		if !testutil.DeepEqual(gotDBState, c.wantDBState) {
			t.Fatalf("case %d: got tradePairs, gotDBState: %v, wantDBStatus: %v.", i, gotDBState, c.wantDBState)
		}

		testDB.Close()
		os.RemoveAll("temp")
	}
}

func TestListOrders(t *testing.T) {
	cases := []struct {
		desc        string
		storeOrders []*common.Order
		query       *common.Order
		wantOrders  []*common.Order
	}{
		{
			desc:       "empty",
			query:      &common.Order{FromAssetID: assetID1, ToAssetID: assetID2},
			wantOrders: []*common.Order{},
		},
		{
			desc: "query from first",
			storeOrders: []*common.Order{
				mockOrders[0],
				mockOrders[1],
				mockOrders[2],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
				mockOrders[6],
				mockOrders[7],
				mockOrders[10],
			},
			query: &common.Order{FromAssetID: assetID1, ToAssetID: assetID2},
			wantOrders: []*common.Order{
				mockOrders[1],
				mockOrders[7],
				mockOrders[6],
				mockOrders[2],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
				mockOrders[0],
			},
		},
		{
			desc: "query from middle",
			storeOrders: []*common.Order{
				mockOrders[0],
				mockOrders[1],
				mockOrders[2],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
				mockOrders[6],
				mockOrders[7],
			},
			query: mockOrders[3],
			wantOrders: []*common.Order{
				mockOrders[4],
				mockOrders[5],
				mockOrders[0],
			},
		},
	}

	initBlockHeader := &types.BlockHeader{
		Height:  0,
		Version: 1,
	}

	height := initBlockHeader.Height
	hash := initBlockHeader.Hash()

	defer os.RemoveAll("temp")
	for i, c := range cases {
		testDB := dbm.NewDB("testdb", "leveldb", "temp")
		movStore := NewLevelDBMovStore(testDB)
		if err := movStore.InitDBState(height, &hash); err != nil {
			t.Fatalf("case %d: InitDBState error %v.", i, err)
		}

		batch := movStore.db.NewBatch()
		tradePairsCnt := make(map[string]*common.TradePair)
		movStore.addOrders(batch, c.storeOrders, tradePairsCnt)
		movStore.updateTradePairs(batch, tradePairsCnt)
		batch.Write()

		gotOrders, err := movStore.ListOrders(c.query)
		if err != nil {
			t.Fatalf("case %d: ListOrders error %v.", i, err)
		}

		if !testutil.DeepEqual(gotOrders, c.wantOrders) {
			t.Fatalf("case %d: got orders , gotOrders: %v, wantOrders: %v.", i, gotOrders, c.wantOrders)
		}

		testDB.Close()
		os.RemoveAll("temp")
	}
}

func TestAddOrders(t *testing.T) {
	cases := []struct {
		desc         string
		beforeOrders []*common.Order
		addOrders    []*common.Order
		wantOrders   []*common.Order
	}{
		{
			desc: "empty",
			addOrders: []*common.Order{
				mockOrders[0],
				mockOrders[1],
				mockOrders[2],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
				mockOrders[6],
				mockOrders[7],
			},
			wantOrders: []*common.Order{
				mockOrders[1],
				mockOrders[7],
				mockOrders[6],
				mockOrders[2],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
				mockOrders[0],
			},
		},
		{
			desc: "Stored data already exists",
			beforeOrders: []*common.Order{
				mockOrders[0],
				mockOrders[1],
				mockOrders[2],
				mockOrders[3],
			},
			addOrders: []*common.Order{
				mockOrders[4],
				mockOrders[5],
				mockOrders[6],
				mockOrders[7],
			},
			wantOrders: []*common.Order{
				mockOrders[1],
				mockOrders[7],
				mockOrders[6],
				mockOrders[2],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
				mockOrders[0],
			},
		},
	}

	initBlockHeader := &types.BlockHeader{
		Height:  0,
		Version: 1,
	}

	height := initBlockHeader.Height
	hash := initBlockHeader.Hash()

	defer os.RemoveAll("temp")
	for i, c := range cases {
		testDB := dbm.NewDB("testdb", "leveldb", "temp")
		movStore := NewLevelDBMovStore(testDB)
		if err := movStore.InitDBState(height, &hash); err != nil {
			t.Fatalf("case %d: InitDBState error %v.", i, err)
		}

		batch := movStore.db.NewBatch()
		tradePairsCnt := make(map[string]*common.TradePair)
		movStore.addOrders(batch, c.beforeOrders, tradePairsCnt)
		movStore.updateTradePairs(batch, tradePairsCnt)
		batch.Write()

		tradePairsCnt = make(map[string]*common.TradePair)
		movStore.addOrders(batch, c.addOrders, tradePairsCnt)
		batch.Write()

		gotOrders, err := movStore.ListOrders(&common.Order{FromAssetID: assetID1, ToAssetID: assetID2})
		if err != nil {
			t.Fatalf("case %d: ListOrders error %v.", i, err)
		}

		if !testutil.DeepEqual(gotOrders, c.wantOrders) {
			t.Fatalf("case %d: got orders , gotOrders: %v, wantOrders: %v.", i, gotOrders, c.wantOrders)
		}

		testDB.Close()
		os.RemoveAll("temp")
	}
}

func TestDelOrders(t *testing.T) {
	cases := []struct {
		desc         string
		beforeOrders []*common.Order
		delOrders    []*common.Order
		wantOrders   []*common.Order
		err          error
	}{
		{
			desc: "empty",
			delOrders: []*common.Order{
				mockOrders[0],
				mockOrders[1],
			},
			wantOrders: []*common.Order{},
			err:        errors.New("don't find trade pair"),
		},
		{
			desc: "Delete existing data",
			beforeOrders: []*common.Order{
				mockOrders[1],
				mockOrders[7],
				mockOrders[6],
				mockOrders[2],
				mockOrders[3],
				mockOrders[4],
				mockOrders[5],
				mockOrders[0],
			},
			delOrders: []*common.Order{
				mockOrders[4],
				mockOrders[5],
				mockOrders[6],
				mockOrders[7],
			},
			wantOrders: []*common.Order{
				mockOrders[1],
				mockOrders[2],
				mockOrders[3],
				mockOrders[0],
			},
			err: nil,
		},
		{
			desc: "Delete all data",
			beforeOrders: []*common.Order{
				mockOrders[7],
				mockOrders[6],
				mockOrders[5],
			},
			delOrders: []*common.Order{
				mockOrders[5],
				mockOrders[6],
				mockOrders[7],
			},
			wantOrders: []*common.Order{},
			err:        nil,
		},
	}

	initBlockHeader := &types.BlockHeader{
		Height:  0,
		Version: 1,
	}

	height := initBlockHeader.Height
	hash := initBlockHeader.Hash()

	defer os.RemoveAll("temp")
	for i, c := range cases {
		testDB := dbm.NewDB("testdb", "leveldb", "temp")
		movStore := NewLevelDBMovStore(testDB)
		if err := movStore.InitDBState(height, &hash); err != nil {
			t.Fatalf("case %d: InitDBState error %v.", i, err)
		}

		batch := movStore.db.NewBatch()
		tradePairsCnt := make(map[string]*common.TradePair)
		movStore.addOrders(batch, c.beforeOrders, tradePairsCnt)
		movStore.updateTradePairs(batch, tradePairsCnt)
		batch.Write()

		tradePairsCnt = make(map[string]*common.TradePair)
		movStore.deleteOrders(batch, c.delOrders, tradePairsCnt)
		movStore.updateTradePairs(batch, tradePairsCnt)
		batch.Write()

		gotOrders, err := movStore.ListOrders(&common.Order{FromAssetID: assetID1, ToAssetID: assetID2})
		if err != nil {
			t.Fatalf("case %d: ListOrders error %v.", i, err)
		}

		if !testutil.DeepEqual(gotOrders, c.wantOrders) {
			t.Fatalf("case %d: got orders , gotOrders: %v, wantOrders: %v.", i, gotOrders, c.wantOrders)
		}

		testDB.Close()
		os.RemoveAll("temp")
	}
}

func TestListTradePairsWithStart(t *testing.T) {
	cases := []struct {
		desc            string
		storeTradePairs map[string]*common.TradePair
		query           *common.TradePair
		wantTradePairs  []*common.TradePair
	}{
		{
			desc:           "empty",
			query:          &common.TradePair{},
			wantTradePairs: []*common.TradePair{},
		},
		{
			desc: "query from first",
			storeTradePairs: map[string]*common.TradePair{
				(&common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2}).Key(): {FromAssetID: assetID1, ToAssetID: assetID2, Count: 1},
				(&common.TradePair{FromAssetID: assetID2, ToAssetID: assetID3}).Key(): {FromAssetID: assetID2, ToAssetID: assetID3, Count: 2},
				(&common.TradePair{FromAssetID: assetID3, ToAssetID: assetID4}).Key(): {FromAssetID: assetID3, ToAssetID: assetID4, Count: 3},
				(&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID5}).Key(): {FromAssetID: assetID4, ToAssetID: assetID5, Count: 4},
				(&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6}).Key(): {FromAssetID: assetID4, ToAssetID: assetID6, Count: 5},
				(&common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7}).Key(): {FromAssetID: assetID5, ToAssetID: assetID7, Count: 6},
			},
			query: &common.TradePair{},
			wantTradePairs: []*common.TradePair{
				&common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2, Count: 1},
				&common.TradePair{FromAssetID: assetID2, ToAssetID: assetID3, Count: 2},
				&common.TradePair{FromAssetID: assetID3, ToAssetID: assetID4, Count: 3},
				&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID5, Count: 4},
				&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6, Count: 5},
				&common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7, Count: 6},
			},
		},
		{
			desc: "query from middle",
			storeTradePairs: map[string]*common.TradePair{
				(&common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2}).Key(): {FromAssetID: assetID1, ToAssetID: assetID2, Count: 1},
				(&common.TradePair{FromAssetID: assetID2, ToAssetID: assetID3}).Key(): {FromAssetID: assetID2, ToAssetID: assetID3, Count: 2},
				(&common.TradePair{FromAssetID: assetID3, ToAssetID: assetID4}).Key(): {FromAssetID: assetID3, ToAssetID: assetID4, Count: 3},
				(&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID5}).Key(): {FromAssetID: assetID4, ToAssetID: assetID5, Count: 4},
				(&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6}).Key(): {FromAssetID: assetID4, ToAssetID: assetID6, Count: 5},
				(&common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7}).Key(): {FromAssetID: assetID5, ToAssetID: assetID7, Count: 6},
				(&common.TradePair{FromAssetID: assetID6, ToAssetID: assetID8}).Key(): {FromAssetID: assetID6, ToAssetID: assetID8, Count: 7},
			},
			query: &common.TradePair{FromAssetID: assetID3, ToAssetID: assetID4, Count: 3},
			wantTradePairs: []*common.TradePair{
				&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID5, Count: 4},
				&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6, Count: 5},
				&common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7, Count: 6},
				&common.TradePair{FromAssetID: assetID6, ToAssetID: assetID8, Count: 7},
			},
		},
	}

	initBlockHeader := &types.BlockHeader{
		Height:  0,
		Version: 1,
	}

	height := initBlockHeader.Height
	hash := initBlockHeader.Hash()

	defer os.RemoveAll("temp")
	for i, c := range cases {
		testDB := dbm.NewDB("testdb", "leveldb", "temp")
		movStore := NewLevelDBMovStore(testDB)
		if err := movStore.InitDBState(height, &hash); err != nil {
			t.Fatalf("case %d: InitDBState error %v.", i, err)
		}

		batch := movStore.db.NewBatch()
		movStore.updateTradePairs(batch, c.storeTradePairs)
		batch.Write()

		gotTradePairs, err := movStore.ListTradePairsWithStart(c.query.FromAssetID, c.query.ToAssetID)
		if err != nil {
			t.Fatalf("case %d: ListTradePairsWithStart error %v.", i, err)
		}

		if !testutil.DeepEqual(gotTradePairs, c.wantTradePairs) {
			t.Fatalf("case %d: got TradePairs , gotTradePairs: %v, wantTradePairs: %v.", i, gotTradePairs, c.wantTradePairs)
		}

		testDB.Close()
		os.RemoveAll("temp")
	}
}

func TestUpdateTradePairs(t *testing.T) {
	cases := []struct {
		desc             string
		beforeTradePairs map[string]*common.TradePair
		addTradePairs    map[string]*common.TradePair
		delTradePairs    map[string]*common.TradePair
		wantTradePairs   []*common.TradePair
	}{
		{
			desc: "empty",
			addTradePairs: map[string]*common.TradePair{
				(&common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2}).Key(): {FromAssetID: assetID1, ToAssetID: assetID2, Count: 1},
				(&common.TradePair{FromAssetID: assetID2, ToAssetID: assetID3}).Key(): {FromAssetID: assetID2, ToAssetID: assetID3, Count: 2},
				(&common.TradePair{FromAssetID: assetID3, ToAssetID: assetID4}).Key(): {FromAssetID: assetID3, ToAssetID: assetID4, Count: 3},
				(&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID5}).Key(): {FromAssetID: assetID4, ToAssetID: assetID5, Count: 4},
				(&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6}).Key(): {FromAssetID: assetID4, ToAssetID: assetID6, Count: 5},
				(&common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7}).Key(): {FromAssetID: assetID5, ToAssetID: assetID7, Count: 6},
			},
			wantTradePairs: []*common.TradePair{
				&common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2, Count: 1},
				&common.TradePair{FromAssetID: assetID2, ToAssetID: assetID3, Count: 2},
				&common.TradePair{FromAssetID: assetID3, ToAssetID: assetID4, Count: 3},
				&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID5, Count: 4},
				&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6, Count: 5},
				&common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7, Count: 6},
			},
		},
		{
			desc: "Stored data already exists",
			beforeTradePairs: map[string]*common.TradePair{
				(&common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2}).Key(): {FromAssetID: assetID1, ToAssetID: assetID2, Count: 1},
				(&common.TradePair{FromAssetID: assetID2, ToAssetID: assetID3}).Key(): {FromAssetID: assetID2, ToAssetID: assetID3, Count: 2},
				(&common.TradePair{FromAssetID: assetID3, ToAssetID: assetID4}).Key(): {FromAssetID: assetID3, ToAssetID: assetID4, Count: 3},
			},
			addTradePairs: map[string]*common.TradePair{
				(&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID5}).Key(): {FromAssetID: assetID4, ToAssetID: assetID5, Count: 4},
				(&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6}).Key(): {FromAssetID: assetID4, ToAssetID: assetID6, Count: 5},
				(&common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7}).Key(): {FromAssetID: assetID5, ToAssetID: assetID7, Count: 6},
			},
			wantTradePairs: []*common.TradePair{
				&common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2, Count: 1},
				&common.TradePair{FromAssetID: assetID2, ToAssetID: assetID3, Count: 2},
				&common.TradePair{FromAssetID: assetID3, ToAssetID: assetID4, Count: 3},
				&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID5, Count: 4},
				&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6, Count: 5},
				&common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7, Count: 6},
			},
		},
		{
			desc: "delete some data",
			beforeTradePairs: map[string]*common.TradePair{
				(&common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2}).Key(): {FromAssetID: assetID1, ToAssetID: assetID2, Count: 1},
				(&common.TradePair{FromAssetID: assetID2, ToAssetID: assetID3}).Key(): {FromAssetID: assetID2, ToAssetID: assetID3, Count: 2},
				(&common.TradePair{FromAssetID: assetID3, ToAssetID: assetID4}).Key(): {FromAssetID: assetID3, ToAssetID: assetID4, Count: 3},
				(&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID5}).Key(): {FromAssetID: assetID4, ToAssetID: assetID5, Count: 4},
				(&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6}).Key(): {FromAssetID: assetID4, ToAssetID: assetID6, Count: 5},
				(&common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7}).Key(): {FromAssetID: assetID5, ToAssetID: assetID7, Count: 6},
			},
			delTradePairs: map[string]*common.TradePair{
				(&common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2}).Key(): {FromAssetID: assetID1, ToAssetID: assetID2, Count: -1},
				(&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID5}).Key(): {FromAssetID: assetID4, ToAssetID: assetID5, Count: -4},
				(&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6}).Key(): {FromAssetID: assetID4, ToAssetID: assetID6, Count: -2},
				(&common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7}).Key(): {FromAssetID: assetID5, ToAssetID: assetID7, Count: -4},
			},
			wantTradePairs: []*common.TradePair{
				&common.TradePair{FromAssetID: assetID2, ToAssetID: assetID3, Count: 2},
				&common.TradePair{FromAssetID: assetID3, ToAssetID: assetID4, Count: 3},
				&common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6, Count: 3},
				&common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7, Count: 2},
			},
		},
	}

	initBlockHeader := &types.BlockHeader{
		Height:  0,
		Version: 1,
	}

	height := initBlockHeader.Height
	hash := initBlockHeader.Hash()

	defer os.RemoveAll("temp")
	for i, c := range cases {
		testDB := dbm.NewDB("testdb", "leveldb", "temp")
		movStore := NewLevelDBMovStore(testDB)
		if err := movStore.InitDBState(height, &hash); err != nil {
			t.Fatalf("case %d: InitDBState error %v.", i, err)
		}

		batch := movStore.db.NewBatch()
		movStore.updateTradePairs(batch, c.beforeTradePairs)
		batch.Write()

		movStore.updateTradePairs(batch, c.addTradePairs)
		movStore.updateTradePairs(batch, c.delTradePairs)
		batch.Write()

		gotTradePairs, err := movStore.ListTradePairsWithStart(nil, nil)
		if err != nil {
			t.Fatalf("case %d: ListTradePairsWithStart error %v.", i, err)
		}

		if !testutil.DeepEqual(gotTradePairs, c.wantTradePairs) {
			t.Fatalf("case %d: got TradePairs , gotTradePairs: %v, wantTradePairs: %v.", i, gotTradePairs, c.wantTradePairs)
		}

		testDB.Close()
		os.RemoveAll("temp")
	}
}

func TestCheckMovDatabaseState(t *testing.T) {
	cases := []struct {
		desc           string
		beforeDBStatus *common.MovDatabaseState
		blockHeader    *types.BlockHeader
		err            error
	}{
		{
			desc:           "attach Block",
			beforeDBStatus: &common.MovDatabaseState{Height: 1, Hash: &bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
			blockHeader:    &types.BlockHeader{Height: 2, PreviousBlockHash: bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
			err:            nil,
		},
		{
			desc:           "error attach Block",
			beforeDBStatus: &common.MovDatabaseState{Height: 1, Hash: &bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
			blockHeader:    &types.BlockHeader{Height: 2, PreviousBlockHash: bc.Hash{}},
			err:            errors.New("the status of the block is inconsistent with that of mov-database"),
		},

		{
			desc:           "detach Block",
			beforeDBStatus: &common.MovDatabaseState{Height: 5, Hash: &bc.Hash{V0: 3724755213446347384, V1: 158878632373345042, V2: 18283800951484248781, V3: 7520797730449067221}},
			blockHeader:    &types.BlockHeader{Height: 4},
			err:            nil,
		},
		{
			desc:           "error detach Block",
			beforeDBStatus: &common.MovDatabaseState{Height: 5, Hash: &bc.Hash{V0: 3724755213446347384, V1: 158878632373345042, V2: 18283800951484248781, V3: 7520797730449067221}},
			blockHeader:    &types.BlockHeader{Height: 3},
			err:            errors.New("the status of the block is inconsistent with that of mov-database"),
		},
	}

	initBlockHeader := &types.BlockHeader{
		Height:  0,
		Version: 1,
	}

	height := initBlockHeader.Height
	hash := initBlockHeader.Hash()

	defer os.RemoveAll("temp")
	for i, c := range cases {
		testDB := dbm.NewDB("testdb", "leveldb", "temp")
		movStore := NewLevelDBMovStore(testDB)
		if err := movStore.InitDBState(height, &hash); err != nil {
			t.Fatalf("case %d: InitDBState error %v.", i, err)
		}

		batch := movStore.db.NewBatch()
		movStore.saveMovDatabaseState(batch, c.beforeDBStatus)
		batch.Write()

		if err := movStore.checkMovDatabaseState(c.blockHeader); c.err != nil && c.err.Error() != err.Error() {
			t.Fatalf("case %d: checkMovDatabaseState error %v.", i, err)
		}

		testDB.Close()
		os.RemoveAll("temp")
	}

}
