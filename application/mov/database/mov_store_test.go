package database

import (
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"math"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/sha3"

	"github.com/vapor/application/mov/common"
	"github.com/vapor/database/leveldb"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/testutil"
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
			key := calcOrdersKey(order.FromAssetID, order.ToAssetID, &utxoHash, order.Rate)
			db.SetSync(key, data)
		}

		got := []expectedData{}

		itr := db.IteratorPrefixWithStart(nil, nil, false)
		for itr.Next() {
			key := itr.Key()
			pos := len(ordersPreFix) + 32*2
			b := [32]byte{}
			copy(b[:], key[pos+8:])
			utxoHash := bc.NewHash(b)
			got = append(got, expectedData{
				rate:     math.Float64frombits(binary.BigEndian.Uint64(key[pos : pos+8])),
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

	assetID1 := &bc.AssetID{V0: 1}
	assetID2 := &bc.AssetID{V0: 2}

	cases := []struct {
		desc           string
		beforeOrders   []*common.Order
		addOrders      []*common.Order
		delOrders      []*common.Order
		Height         uint64
		blockHash      *bc.Hash
		wantOrders     []*common.Order
		wantTradePairs []*common.TradePair
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
			Height:    10,
			blockHash: &bc.Hash{},
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
			Height:    10,
			blockHash: &bc.Hash{},
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
			Height:         10,
			blockHash:      &bc.Hash{},
			wantOrders:     []*common.Order{},
			wantTradePairs: []*common.TradePair{},
		},
	}
	defer os.RemoveAll("temp")
	for i, c := range cases {
		testDB := dbm.NewDB("testdb", "leveldb", "temp")
		dexStore := NewMovStore(testDB)

		batch := dexStore.db.NewBatch()
		dexStore.addOrders(batch, c.beforeOrders)
		batch.Write()

		if err := dexStore.ProcessOrders(c.addOrders, c.delOrders, c.Height, c.blockHash); err != nil {
			t.Fatalf("case %d: ProcessOrders error %v.", i, err)
		}

		gotOrders, err := dexStore.ListOrders(assetID1, assetID2, 0)
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

		testDB.Close()
		os.RemoveAll("temp")
	}
}
