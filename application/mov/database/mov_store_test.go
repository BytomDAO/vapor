package database

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"

	"github.com/vapor/application/mov/common"
	"github.com/vapor/database/leveldb"
	dbm "github.com/vapor/database/leveldb"
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
					utxoHash: "7967bb8c4cca951749553e9c7787255d35a032d9e1acecefe4011c8095dc8e6f",
				},
				expectedData{
					rate:     0.00088,
					utxoHash: "215a6e7e3a5006151bd0b81c54fcccda0381f3a22e7b6646ed201c35f9fa6c5a",
				},
				expectedData{
					rate:     0.00090,
					utxoHash: "cb373d3a383d30eb2863317ea2cfb5b4b269772fbc0fb8413a2be7d7b69ec2b9",
				},
				expectedData{
					rate:     0.00091,
					utxoHash: "298c39d327cb4b0dcefcf701aa8d1b559f1de0148e9bcbe14da48cfa268c01ea",
				},
				expectedData{
					rate:     0.00092,
					utxoHash: "b2c59190fb0d948c9545c146a69b1f17503b2d280b2f3f45ecc0a7b7e2cd1784",
				},
				expectedData{
					rate:     0.00093,
					utxoHash: "80b44aae2b2cf57bd2cf77cb88f0d8363066f5f16a17a3e85224ecbbc6387d8b",
				},
				expectedData{
					rate:     0.00094,
					utxoHash: "4843adc8c4a50672a022e5f377dfd2ac11119364dc0a547be45b4a5edacef33b",
				},
				expectedData{
					rate:     0.00095,
					utxoHash: "298c39d327cb4b0dcefcf701aa8d1b559f1de0148e9bcbe14da48cfa268c01ea",
				},
				expectedData{
					rate:     0.00096,
					utxoHash: "d8d1a85303e9ac738e675b874b866e5ffbfa10e05201113404dde544055a18b9",
				},
				expectedData{
					rate:     0.00097,
					utxoHash: "2305be66ab9648b713a58e3807fa1cba1f84e5d11359b316e967d98e9a7667da",
				},
				expectedData{
					rate:     0.00098,
					utxoHash: "030dc8a868a3e534799d465ebc8209eb32d9465985dc8c35e731b124bf3ffbcf",
				},
				expectedData{
					rate:     0.00098,
					utxoHash: "8222a9a43b3951f247612ddce2fe36f96cd843bc0dfef86c7d0ef5335331f11f",
				},
				expectedData{
					rate:     0.00099,
					utxoHash: "a40bd183cd2ff2b52faac5ebc2cfc1e36104cbc92bcebac011b45792b39e380e",
				},
				expectedData{
					rate:     1.0009,
					utxoHash: "118b2c40848887614d99b0e7eb6c88a10b47196e6aca3ff2eeab452bfdb9cfcb",
				},
				expectedData{
					rate:     888888.7954,
					utxoHash: "545a5c6f7ff9be19ed07a7246277c67d661f9cc7d8956bb81ce9a4045fba3720",
				},
				expectedData{
					rate:     999999.9521,
					utxoHash: "d9f7725d908510268c7bdecd29cb2031baab93b9bfa69108eb0a926ba7ae18f9",
				},
			},
		},
	}

	for i, c := range cases {
		for _, order := range c.orders {
			data, err := json.Marshal(order.Utxo)
			if err != nil {
				t.Fatal(err)
			}
			utxoHash := bc.NewHash(sha3.Sum256(data))
			key := calcOrderKey(order.FromAssetID, order.ToAssetID, &utxoHash, order.Rate)
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

			rate := getRateFromKey(key, ordersPrefix)
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

func TestDexStore(t *testing.T) {
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
		dexStore, err := NewMovStore(testDB, height, &hash)
		if err != nil {
			t.Fatalf("case %d: NewMovStore error %v.", i, err)
		}

		batch := dexStore.db.NewBatch()
		dexStore.addOrders(batch, c.beforeOrders)
		if len(c.beforeOrders) > 0 {
			tradePairs := make(map[common.TradePair]int)
			tradePairs[*c.beforeTradePairs[0]] = c.beforeTradePairs[0].Count
			dexStore.updateTradePairs(batch, tradePairs)
			dexStore.saveMovDatabaseState(batch, c.beforeDBStatus)
		}
		batch.Write()

		if err := dexStore.ProcessOrders(c.addOrders, c.delOrders, c.blockHeader); err != nil {
			t.Fatalf("case %d: ProcessOrders error %v.", i, err)
		}

		gotOrders, err := dexStore.ListOrders(&common.Order{FromAssetID: assetID1, ToAssetID: assetID2, Rate: 0})
		if err != nil {
			t.Fatalf("case %d: ListOrders error %v.", i, err)
		}

		if !testutil.DeepEqual(gotOrders, c.wantOrders) {
			t.Fatalf("case %d: got orders , gotOrders: %v, wantOrders: %v.", i, gotOrders, c.wantOrders)
		}

		gotTradePairs, err := dexStore.ListTradePairsWithStart(nil, nil)
		if err != nil {
			t.Fatalf("case %d: GetDexDatabaseState error %v.", i, err)
		}

		if !testutil.DeepEqual(gotTradePairs, c.wantTradePairs) {
			t.Fatalf("case %d: got tradePairs, gotTradePairs: %v, wantTradePairs: %v.", i, gotTradePairs, c.wantTradePairs)
		}

		gotDBState, err := dexStore.GetMovDatabaseState()
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
		dexStore, err := NewMovStore(testDB, height, &hash)
		if err != nil {
			t.Fatalf("case %d: NewMovStore error %v.", i, err)
		}

		batch := dexStore.db.NewBatch()
		dexStore.addOrders(batch, c.storeOrders)
		batch.Write()

		gotOrders, err := dexStore.ListOrders(c.query)
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
		dexStore, err := NewMovStore(testDB, height, &hash)
		if err != nil {
			t.Fatalf("case %d: NewMovStore error %v.", i, err)
		}

		batch := dexStore.db.NewBatch()
		dexStore.addOrders(batch, c.beforeOrders)
		batch.Write()

		dexStore.addOrders(batch, c.addOrders)
		batch.Write()

		gotOrders, err := dexStore.ListOrders(&common.Order{FromAssetID: assetID1, ToAssetID: assetID2})
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
		dexStore, err := NewMovStore(testDB, height, &hash)
		if err != nil {
			t.Fatalf("case %d: NewMovStore error %v.", i, err)
		}

		batch := dexStore.db.NewBatch()
		dexStore.addOrders(batch, c.beforeOrders)
		batch.Write()

		if err := dexStore.deleteOrders(batch, c.delOrders); c.err != nil && err.Error() != c.err.Error() {
			t.Fatalf("case %d: deleteOrder error %v.", i, err)
		}
		batch.Write()

		gotOrders, err := dexStore.ListOrders(&common.Order{FromAssetID: assetID1, ToAssetID: assetID2})
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
		dexStore, err := NewMovStore(testDB, height, &hash)
		if err != nil {
			t.Fatalf("case %d: NewMovStore error %v.", i, err)
		}

		batch := dexStore.db.NewBatch()
		dexStore.updateTradePairs(batch, c.storeTradePairs)
		batch.Write()

		gotTradePairs, err := dexStore.ListTradePairsWithStart(c.query.FromAssetID, c.query.ToAssetID)
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
		dexStore, err := NewMovStore(testDB, height, &hash)
		if err != nil {
			t.Fatalf("case %d: NewMovStore error %v.", i, err)
		}

		batch := dexStore.db.NewBatch()
		dexStore.updateTradePairs(batch, c.beforeTradePairs)
		batch.Write()

		dexStore.updateTradePairs(batch, c.addTradePairs)
		dexStore.updateTradePairs(batch, c.delTradePairs)
		batch.Write()

		gotTradePairs, err := dexStore.ListTradePairsWithStart(nil, nil)
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
		dexStore, err := NewMovStore(testDB, height, &hash)
		if err != nil {
			t.Fatalf("case %d: NewMovStore error %v.", i, err)
		}

		batch := dexStore.db.NewBatch()
		dexStore.saveMovDatabaseState(batch, c.beforeDBStatus)
		batch.Write()

		if err := dexStore.checkMovDatabaseState(c.blockHeader); c.err != nil && c.err.Error() != err.Error() {
			t.Fatalf("case %d: checkMovDatabaseState error %v.", i, err)
		}

		testDB.Close()
		os.RemoveAll("temp")
	}

}
