package database

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vapor/application/mov/common"
	"github.com/vapor/consensus"
	"github.com/vapor/database/leveldb"
	dbm "github.com/vapor/database/leveldb"
	chainjson "github.com/vapor/encoding/json"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/testutil"
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
)

func TestCalcUTXOHash(t *testing.T) {
	wantHash := "7cbaf92f950f2a6bededd6cc5ec08c924505f5365b0a8af963e1d52912c99667"
	controlProgramStr := "0014ab5acbea076f269bfdc8ededbed7d0a13e6e0b19"

	var controlProgram chainjson.HexBytes
	controlProgram.UnmarshalText([]byte(controlProgramStr))

	sourceID := testutil.MustDecodeHash("ca2faf5fcbf8ee2b43560a32594f608528b12a1fe79cee85252564f886f91060")
	order := &common.Order{
		FromAssetID: consensus.BTMAssetID,
		Utxo: &common.MovUtxo{
			SourceID:       &sourceID,
			SourcePos:      0,
			Amount:         31249300000,
			ControlProgram: controlProgram[:],
		},
	}

	hash := calcUTXOHash(order)
	if hash.String() != wantHash {
		t.Fatal("The function is incorrect")
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
		orders []common.Order
		want   []expectedData
	}{
		{
			orders: []common.Order{
				common.Order{
					FromAssetID: &bc.AssetID{V0: 1},
					ToAssetID:   &bc.AssetID{V0: 0},
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					FromAssetID: &bc.AssetID{V0: 1},
					ToAssetID:   &bc.AssetID{V0: 0},
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					FromAssetID: &bc.AssetID{V0: 1},
					ToAssetID:   &bc.AssetID{V0: 0},
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					FromAssetID: &bc.AssetID{V0: 1},
					ToAssetID:   &bc.AssetID{V0: 0},
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					FromAssetID: &bc.AssetID{V0: 1},
					ToAssetID:   &bc.AssetID{V0: 0},
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					FromAssetID: &bc.AssetID{V0: 1},
					ToAssetID:   &bc.AssetID{V0: 0},
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					FromAssetID: &bc.AssetID{V0: 1},
					ToAssetID:   &bc.AssetID{V0: 0},
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					FromAssetID: &bc.AssetID{V0: 1},
					ToAssetID:   &bc.AssetID{V0: 0},
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					FromAssetID: &bc.AssetID{V0: 1},
					ToAssetID:   &bc.AssetID{V0: 0},
					Rate:        0.00091,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					FromAssetID: &bc.AssetID{V0: 1},
					ToAssetID:   &bc.AssetID{V0: 0},
					Rate:        0.00092,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 27},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					FromAssetID: &bc.AssetID{V0: 1},
					ToAssetID:   &bc.AssetID{V0: 0},
					Rate:        0.00093,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 28},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					FromAssetID: &bc.AssetID{V0: 1},
					ToAssetID:   &bc.AssetID{V0: 0},
					Rate:        0.00094,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 29},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					FromAssetID: &bc.AssetID{V0: 1},
					ToAssetID:   &bc.AssetID{V0: 0},
					Rate:        0.00077,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 30},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					FromAssetID: &bc.AssetID{V0: 1},
					ToAssetID:   &bc.AssetID{V0: 0},
					Rate:        0.00088,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 31},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					FromAssetID: &bc.AssetID{V0: 1},
					ToAssetID:   &bc.AssetID{V0: 0},
					Rate:        999999.9521,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 32},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					FromAssetID: &bc.AssetID{V0: 1},
					ToAssetID:   &bc.AssetID{V0: 0},
					Rate:        888888.7954,
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
					utxoHash: "1fa9fae83d0a5401a4e92f80636966486e763eecca588aa11dff02b415320602",
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
			key := calcOrderKey(order.FromAssetID, order.ToAssetID, calcUTXOHash(&order), order.Rate)
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

			rate := getRateFromOrderKey(key, ordersPrefix)
			got = append(got, expectedData{
				rate:     rate,
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
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			blockHeader: &types.BlockHeader{Height: 1, PreviousBlockHash: bc.Hash{V0: 524821139490765641, V1: 2484214155808702787, V2: 9108473449351508820, V3: 7972721253564512122}},
			wantOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			wantTradePairs: []*common.TradePair{
				&common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2, Count: 8},
			},
			wantDBState: &common.MovDatabaseState{Height: 1, Hash: &bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
		},
		{
			desc: "del some order",
			beforeOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
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
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			blockHeader: &types.BlockHeader{Height: 2, PreviousBlockHash: bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
			wantOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			wantTradePairs: []*common.TradePair{
				&common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2, Count: 5},
			},
			wantDBState: &common.MovDatabaseState{Height: 2, Hash: &bc.Hash{V0: 3724755213446347384, V1: 158878632373345042, V2: 18283800951484248781, V3: 7520797730449067221}},
		},
		{
			desc: "del all order",
			beforeOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
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
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			blockHeader:    &types.BlockHeader{Height: 2, PreviousBlockHash: bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
			wantOrders:     []*common.Order{},
			wantTradePairs: []*common.TradePair{},
			wantDBState:    &common.MovDatabaseState{Height: 2, Hash: &bc.Hash{V0: 3724755213446347384, V1: 158878632373345042, V2: 18283800951484248781, V3: 7520797730449067221}},
		},
		{
			desc: "Add and delete the same trade pair", //Add and delete different transaction pairs
			beforeOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
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
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 2},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			delOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			blockHeader: &types.BlockHeader{Height: 2, PreviousBlockHash: bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
			wantOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 2},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			wantTradePairs: []*common.TradePair{
				&common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2, Count: 2},
			},
			wantDBState: &common.MovDatabaseState{Height: 2, Hash: &bc.Hash{V0: 3724755213446347384, V1: 158878632373345042, V2: 18283800951484248781, V3: 7520797730449067221}},
		},
		{
			desc: "Add and delete different transaction pairs",
			beforeOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID3,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 33},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID4,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 34},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
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
				&common.Order{
					FromAssetID: assetID4,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 36},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID5,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 37},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID6,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 38},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			delOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID3,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 33},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			blockHeader: &types.BlockHeader{Height: 2, PreviousBlockHash: bc.Hash{V0: 14213576368347360351, V1: 16287398171800437029, V2: 9513543230620030445, V3: 8534035697182508177}},
			wantOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID4,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 34},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID4,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 36},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID5,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 37},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID6,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 38},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
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
		movStore, err := NewMovStore(testDB, height, &hash)
		if err != nil {
			t.Fatalf("case %d: NewMovStore error %v.", i, err)
		}

		batch := movStore.db.NewBatch()
		tradePairsCnt := make(map[common.TradePair]int)
		movStore.addOrders(batch, c.beforeOrders, tradePairsCnt)
		if len(c.beforeOrders) > 0 {
			tradePairsCnt = make(map[common.TradePair]int)
			for _, tradePair := range c.beforeTradePairs {
				tradePairsCnt[*tradePair] = tradePair.Count
			}
			movStore.updateTradePairs(batch, tradePairsCnt)
			movStore.saveMovDatabaseState(batch, c.beforeDBStatus)
		}
		batch.Write()

		if err := movStore.ProcessOrders(c.addOrders, c.delOrders, c.blockHeader); err != nil {
			t.Fatalf("case %d: ProcessOrders error %v.", i, err)
		}

		var gotOrders []*common.Order

		tmp, err := movStore.ListOrders(&common.Order{FromAssetID: assetID1, ToAssetID: assetID2, Rate: 0})
		if err != nil {
			t.Fatalf("case %d: ListOrders(assetID1 and assetID2) error %v.", i, err)
		}

		gotOrders = append(gotOrders, tmp...)

		tmp, err = movStore.ListOrders(&common.Order{FromAssetID: assetID3, ToAssetID: assetID2, Rate: 0})
		if err != nil {
			t.Fatalf("case %d: ListOrders(assetID3 and assetID2)  error %v.", i, err)
		}

		gotOrders = append(gotOrders, tmp...)

		tmp, err = movStore.ListOrders(&common.Order{FromAssetID: assetID4, ToAssetID: assetID2, Rate: 0})
		if err != nil {
			t.Fatalf("case %d: ListOrders(assetID4 and assetID2)  error %v.", i, err)
		}

		gotOrders = append(gotOrders, tmp...)

		tmp, err = movStore.ListOrders(&common.Order{FromAssetID: assetID5, ToAssetID: assetID2, Rate: 0})
		if err != nil {
			t.Fatalf("case %d: ListOrders(assetID5 and assetID2)  error %v.", i, err)
		}

		gotOrders = append(gotOrders, tmp...)

		tmp, err = movStore.ListOrders(&common.Order{FromAssetID: assetID6, ToAssetID: assetID2, Rate: 0})
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
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			query: &common.Order{FromAssetID: assetID1, ToAssetID: assetID2},
			wantOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
		},
		{
			desc: "query from middle",
			storeOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			query: &common.Order{
				FromAssetID: assetID1,
				ToAssetID:   assetID2,
				Rate:        0.00098,
				Utxo: &common.MovUtxo{
					SourceID:       &bc.Hash{V0: 13},
					Amount:         1,
					SourcePos:      0,
					ControlProgram: []byte("aa"),
				},
			},
			wantOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
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
		movStore, err := NewMovStore(testDB, height, &hash)
		if err != nil {
			t.Fatalf("case %d: NewMovStore error %v.", i, err)
		}

		batch := movStore.db.NewBatch()
		tradePairsCnt := make(map[common.TradePair]int)
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
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			wantOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
		},
		{
			desc: "Stored data already exists",
			beforeOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			addOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			wantOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
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
		movStore, err := NewMovStore(testDB, height, &hash)
		if err != nil {
			t.Fatalf("case %d: NewMovStore error %v.", i, err)
		}

		batch := movStore.db.NewBatch()
		tradePairsCnt := make(map[common.TradePair]int)
		movStore.addOrders(batch, c.beforeOrders, tradePairsCnt)
		movStore.updateTradePairs(batch, tradePairsCnt)
		batch.Write()

		tradePairsCnt = make(map[common.TradePair]int)
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
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			wantOrders: []*common.Order{},
			err:        errors.New("don't find trade pair"),
		},
		{
			desc: "Delete existing data",
			beforeOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			delOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			wantOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 22},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00097,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 23},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00098,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 13},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        1.00090,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 21},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			err: nil,
		},
		{
			desc: "Delete all data",
			beforeOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			delOrders: []*common.Order{
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00099,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 24},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00096,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 25},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				&common.Order{
					FromAssetID: assetID1,
					ToAssetID:   assetID2,
					Rate:        0.00095,
					Utxo: &common.MovUtxo{
						SourceID:       &bc.Hash{V0: 26},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
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
		movStore, err := NewMovStore(testDB, height, &hash)
		if err != nil {
			t.Fatalf("case %d: NewMovStore error %v.", i, err)
		}

		batch := movStore.db.NewBatch()
		tradePairsCnt := make(map[common.TradePair]int)
		movStore.addOrders(batch, c.beforeOrders, tradePairsCnt)
		movStore.updateTradePairs(batch, tradePairsCnt)
		batch.Write()

		tradePairsCnt = make(map[common.TradePair]int)
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
		storeTradePairs map[common.TradePair]int
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
			storeTradePairs: map[common.TradePair]int{
				common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2}: 1,
				common.TradePair{FromAssetID: assetID2, ToAssetID: assetID3}: 2,
				common.TradePair{FromAssetID: assetID3, ToAssetID: assetID4}: 3,
				common.TradePair{FromAssetID: assetID4, ToAssetID: assetID5}: 4,
				common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6}: 5,
				common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7}: 6,
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
			storeTradePairs: map[common.TradePair]int{
				common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2}: 1,
				common.TradePair{FromAssetID: assetID2, ToAssetID: assetID3}: 2,
				common.TradePair{FromAssetID: assetID3, ToAssetID: assetID4}: 3,
				common.TradePair{FromAssetID: assetID4, ToAssetID: assetID5}: 4,
				common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6}: 5,
				common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7}: 6,
				common.TradePair{FromAssetID: assetID6, ToAssetID: assetID8}: 7,
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
		movStore, err := NewMovStore(testDB, height, &hash)
		if err != nil {
			t.Fatalf("case %d: NewMovStore error %v.", i, err)
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
		beforeTradePairs map[common.TradePair]int
		addTradePairs    map[common.TradePair]int
		delTradePairs    map[common.TradePair]int
		wantTradePairs   []*common.TradePair
	}{
		{
			desc: "empty",
			addTradePairs: map[common.TradePair]int{
				common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2}: 1,
				common.TradePair{FromAssetID: assetID2, ToAssetID: assetID3}: 2,
				common.TradePair{FromAssetID: assetID3, ToAssetID: assetID4}: 3,
				common.TradePair{FromAssetID: assetID4, ToAssetID: assetID5}: 4,
				common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6}: 5,
				common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7}: 6,
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
			beforeTradePairs: map[common.TradePair]int{
				common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2}: 1,
				common.TradePair{FromAssetID: assetID2, ToAssetID: assetID3}: 2,
				common.TradePair{FromAssetID: assetID3, ToAssetID: assetID4}: 3,
			},
			addTradePairs: map[common.TradePair]int{
				common.TradePair{FromAssetID: assetID4, ToAssetID: assetID5}: 4,
				common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6}: 5,
				common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7}: 6,
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
			beforeTradePairs: map[common.TradePair]int{
				common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2}: 1,
				common.TradePair{FromAssetID: assetID2, ToAssetID: assetID3}: 2,
				common.TradePair{FromAssetID: assetID3, ToAssetID: assetID4}: 3,
				common.TradePair{FromAssetID: assetID4, ToAssetID: assetID5}: 4,
				common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6}: 5,
				common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7}: 6,
			},
			delTradePairs: map[common.TradePair]int{
				common.TradePair{FromAssetID: assetID1, ToAssetID: assetID2}: -1,
				common.TradePair{FromAssetID: assetID4, ToAssetID: assetID5}: -4,
				common.TradePair{FromAssetID: assetID4, ToAssetID: assetID6}: -2,
				common.TradePair{FromAssetID: assetID5, ToAssetID: assetID7}: -4,
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
		movStore, err := NewMovStore(testDB, height, &hash)
		if err != nil {
			t.Fatalf("case %d: NewMovStore error %v.", i, err)
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
		movStore, err := NewMovStore(testDB, height, &hash)
		if err != nil {
			t.Fatalf("case %d: NewMovStore error %v.", i, err)
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
