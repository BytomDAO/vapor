package types

import (
	"testing"

	"github.com/vapor/protocol/bc"
)

func TestRawBlock(t *testing.T) {
	strRawBlock := "0301629f91cdd7f923fbb5283390f3c52b3c2559fcf15de352bae10b7ea0fc50f3e650dfe89de0054066408c9a91229b63bbb2b7510887a0519dcc6446c5b045bb8444afa939368e4f6978a65b4ee5b6f4914fe5c05000459a803ecf59132604e5d334d64249c5e50a0ecc99b38080808080200207010001010502030039380001013effffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa090fbd59901011600143143baf513dead7f3478479401eb1ed38eb1e19f00070100010160015e89a6c0f09e3e512f01dc1e4cf42d28eb67b76def47c59257448af26945f542d8ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffc09af7934f0001160014cc443b29275271cd5e793430fdb7e57e3eae247663024024245117a930a6d398c854fb31bb5e3921d82e357d58778d72e0b4b1f0a3583d0d02a399b06ec5209c86b01f590c36fb29b3f2a3dbbe4b5f6bd9d852085967022013604b73c1df1aff3c2503eff8aa93498dcc7800051f74814123980e09e76a0702013dffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffa0db90f3290116001497e164c8f82edc1412f7d87e37f84fb44b55c012000149ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff80c8afa02501220020df81990a397ca4df7afb786e5fcd33951259e9b7341934c15c7362997a09c51e00"
	block := &Block{}
	err := block.UnmarshalText([]byte(strRawBlock))
	if err != nil {
		t.Errorf("UnmarshalText err : %s", err)
	}
	strTxID := "f7a046ac947efa7da14696c2aa667f4657802609a6aafc056ee4c8d7f4940a83"
	txID := bc.Hash{}
	txID.UnmarshalText([]byte(strTxID))
	for _, tx := range block.Transactions {
		tmp := tx.ID.String()
		if tmp == txID.String() {
			break
		}
	}
	block.Hash()
}
