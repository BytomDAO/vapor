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

	"github.com/vapor/application/dex/common"
	"github.com/vapor/database/leveldb"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/testutil"
)

func TestOrderKey(t *testing.T) {
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
					ToAssetID: bc.AssetID{V0: 0},
					Rate:      1.00090,
					Utxo: common.DexUtxo{
						SourceID:       bc.Hash{V0: 21},
						AssetID:        bc.AssetID{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					ToAssetID: bc.AssetID{V0: 0},
					Rate:      0.00090,
					Utxo: common.DexUtxo{
						SourceID:       bc.Hash{V0: 22},
						AssetID:        bc.AssetID{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					ToAssetID: bc.AssetID{V0: 0},
					Rate:      0.00097,
					Utxo: common.DexUtxo{
						SourceID:       bc.Hash{V0: 23},
						AssetID:        bc.AssetID{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					ToAssetID: bc.AssetID{V0: 0},
					Rate:      0.00098,
					Utxo: common.DexUtxo{
						SourceID:       bc.Hash{V0: 13},
						AssetID:        bc.AssetID{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					ToAssetID: bc.AssetID{V0: 0},
					Rate:      0.00098,
					Utxo: common.DexUtxo{
						SourceID:       bc.Hash{V0: 24},
						AssetID:        bc.AssetID{V0: 1},
						Amount:         10,
						SourcePos:      1,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					ToAssetID: bc.AssetID{V0: 0},
					Rate:      0.00099,
					Utxo: common.DexUtxo{
						SourceID:       bc.Hash{V0: 24},
						AssetID:        bc.AssetID{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					ToAssetID: bc.AssetID{V0: 0},
					Rate:      0.00096,
					Utxo: common.DexUtxo{
						SourceID:       bc.Hash{V0: 25},
						AssetID:        bc.AssetID{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					ToAssetID: bc.AssetID{V0: 0},
					Rate:      0.00095,
					Utxo: common.DexUtxo{
						SourceID:       bc.Hash{V0: 26},
						AssetID:        bc.AssetID{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					ToAssetID: bc.AssetID{V0: 0},
					Rate:      0.00091,
					Utxo: common.DexUtxo{
						SourceID:       bc.Hash{V0: 26},
						AssetID:        bc.AssetID{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					ToAssetID: bc.AssetID{V0: 0},
					Rate:      0.00092,
					Utxo: common.DexUtxo{
						SourceID:       bc.Hash{V0: 27},
						AssetID:        bc.AssetID{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					ToAssetID: bc.AssetID{V0: 0},
					Rate:      0.00093,
					Utxo: common.DexUtxo{
						SourceID:       bc.Hash{V0: 28},
						AssetID:        bc.AssetID{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					ToAssetID: bc.AssetID{V0: 0},
					Rate:      0.00094,
					Utxo: common.DexUtxo{
						SourceID:       bc.Hash{V0: 29},
						AssetID:        bc.AssetID{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					ToAssetID: bc.AssetID{V0: 0},
					Rate:      0.00077,
					Utxo: common.DexUtxo{
						SourceID:       bc.Hash{V0: 30},
						AssetID:        bc.AssetID{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					ToAssetID: bc.AssetID{V0: 0},
					Rate:      0.00088,
					Utxo: common.DexUtxo{
						SourceID:       bc.Hash{V0: 31},
						AssetID:        bc.AssetID{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					ToAssetID: bc.AssetID{V0: 0},
					Rate:      999999.9521,
					Utxo: common.DexUtxo{
						SourceID:       bc.Hash{V0: 32},
						AssetID:        bc.AssetID{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
				common.Order{
					ToAssetID: bc.AssetID{V0: 0},
					Rate:      888888.7954,
					Utxo: common.DexUtxo{
						SourceID:       bc.Hash{V0: 33},
						AssetID:        bc.AssetID{V0: 1},
						Amount:         1,
						SourcePos:      0,
						ControlProgram: []byte("aa"),
					},
				},
			},
			want: []expectedData{
				expectedData{
					rate:     0.00077,
					utxoHash: "1a347f38978f4116667880cd650f90f7117f9e5fa664bf35a762b7542521ecfd",
				},
				expectedData{
					rate:     0.00088,
					utxoHash: "390f96ae1f453bc9b8274b4e309853df9821f09e73e5d4294a21de94a3e82c9d",
				},
				expectedData{
					rate:     0.00090,
					utxoHash: "9b6906500e0468b46b05d1cc389e987503d8ff8f6521a9b3cb2e34812edd7e8b",
				},
				expectedData{
					rate:     0.00091,
					utxoHash: "6be9a8fe98dc3d9404a163d6c25d4c86c07725b2b5274792e3d3991a183b916d",
				},
				expectedData{
					rate:     0.00092,
					utxoHash: "a8c1cca596b76468f52a4559c468e865781e82a39b68fb337f4fe8cae84df38f",
				},
				expectedData{
					rate:     0.00093,
					utxoHash: "675344ad2a35fed61ef4028c89a9bd757b5c2d77a0151cf9e7077231d70a5768",
				},
				expectedData{
					rate:     0.00094,
					utxoHash: "98efa3949171cb01fef826d02fb995351ee833380e34dbe186a663bfe514aeec",
				},
				expectedData{
					rate:     0.00095,
					utxoHash: "6be9a8fe98dc3d9404a163d6c25d4c86c07725b2b5274792e3d3991a183b916d",
				},
				expectedData{
					rate:     0.00096,
					utxoHash: "617f55abaaca68f3f4967823448fd6351b84c8104890b1091db070670e456c21",
				},
				expectedData{
					rate:     0.00097,
					utxoHash: "c4ccb9843d0bb1dbf20d441b3f5c4374421dbfe505d1fb7f3fe551f63e2244d2",
				},
				expectedData{
					rate:     0.00098,
					utxoHash: "658ad7a433b689bd1b2d167c66eb065b5cc16015bfc0bbc122a8e3c370274ead",
				},
				expectedData{
					rate:     0.00098,
					utxoHash: "9dcde7ab0f2f88ed5203e4156160cedc21d68defee6f0e41413757b13da2535c",
				},
				expectedData{
					rate:     0.00099,
					utxoHash: "f2484ada2a12220b20f54552f7a329efdfa455fe062c6247dbe33453c9ff9f3e",
				},
				expectedData{
					rate:     1.0009,
					utxoHash: "f39ed5eaba9d9e4913a9ae32f6922c056c905d9d0bdc628ae6378fe6cb5196a2",
				},
				expectedData{
					rate:     888888.7954,
					utxoHash: "08001794e5ad54d5ee70adfdff186996876b7595d205ca7565d8b6131d36a88a",
				},
				expectedData{
					rate:     999999.9521,
					utxoHash: "cc6dfe6629db9694026ccf8058fc2d4fb251367e2c38c359342976183e58f4f5",
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
			key := calcOrdersPrefix(&order.Utxo.AssetID, &order.ToAssetID, &utxoHash, order.Rate)
			db.SetSync(key, data)
		}

		got := []expectedData{}

		itr := db.IteratorPrefixWithStart(nil, nil, false)
		for itr.Next() {
			key := itr.Key()
			pos := len(OrdersPreFix) + 32*2
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
