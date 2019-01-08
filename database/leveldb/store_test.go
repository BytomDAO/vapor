package leveldb

import (
	"os"
	"testing"

	"github.com/vapor/common"
	"github.com/vapor/config"
	"github.com/vapor/consensus"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"

	dbm "github.com/tendermint/tmlibs/db"
)

func TestLoadBlockIndex(t *testing.T) {
	defer os.RemoveAll("temp")
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	store := NewStore(testDB)
	config.CommonConfig = config.DefaultConfig()
	config.CommonConfig.Consensus.Dpos.SelfVoteSigners = append(config.CommonConfig.Consensus.Dpos.SelfVoteSigners, "vsm1qkm743xmgnvh84pmjchq2s4tnfpgu9ae2f9slep")
	config.CommonConfig.Consensus.Dpos.XPrv = "a8e281b615809046698fb0b0f2804a36d824d48fa443350f10f1b80649d39e5f1e85cf9855548915e36137345910606cbc8e7dd8497c831dce899ee6ac112445"
	for _, v := range config.CommonConfig.Consensus.Dpos.SelfVoteSigners {
		address, err := common.DecodeAddress(v, &consensus.SoloNetParams)
		if err != nil {
			t.Fatal(err)
		}
		config.CommonConfig.Consensus.Dpos.Signers = append(config.CommonConfig.Consensus.Dpos.Signers, address)
	}
	block := config.GenesisBlock()
	txStatus := bc.NewTransactionStatus()

	if err := store.SaveBlock(block, txStatus); err != nil {
		t.Fatal(err)
	}

	for block.Height <= 128 {
		preHash := block.Hash()
		block.PreviousBlockHash = preHash
		block.Height += 1
		if err := store.SaveBlock(block, txStatus); err != nil {
			t.Fatal(err)
		}

		if block.Height%32 != 0 {
			continue
		}

		for i := uint64(0); i < block.Height/32; i++ {
			if err := store.SaveBlock(block, txStatus); err != nil {
				t.Fatal(err)
			}
		}
	}

	if _, err := store.LoadBlockIndex(128); err != nil {
		t.Fatal(err)
	}
}

func TestLoadBlockIndexBestHeight(t *testing.T) {
	cases := []struct {
		blockBestHeight uint64
		stateBestHeight uint64
	}{
		{
			blockBestHeight: 100,
			stateBestHeight: 90,
		},
		{
			blockBestHeight: 100,
			stateBestHeight: 0,
		},
		{
			blockBestHeight: 100,
			stateBestHeight: 100,
		},
	}

	defer os.RemoveAll("temp")
	testDB := dbm.NewDB("testdb", "leveldb", "temp")
	store := NewStore(testDB)
	var savedBlocks []types.Block
	config.CommonConfig = config.DefaultConfig()
	config.CommonConfig.Consensus.Dpos.SelfVoteSigners = append(config.CommonConfig.Consensus.Dpos.SelfVoteSigners, "vsm1qkm743xmgnvh84pmjchq2s4tnfpgu9ae2f9slep")
	config.CommonConfig.Consensus.Dpos.XPrv = "a8e281b615809046698fb0b0f2804a36d824d48fa443350f10f1b80649d39e5f1e85cf9855548915e36137345910606cbc8e7dd8497c831dce899ee6ac112445"
	for _, v := range config.CommonConfig.Consensus.Dpos.SelfVoteSigners {
		address, err := common.DecodeAddress(v, &consensus.SoloNetParams)
		if err != nil {
			t.Fatal(err)
		}
		config.CommonConfig.Consensus.Dpos.Signers = append(config.CommonConfig.Consensus.Dpos.Signers, address)
	}

	for _, c := range cases {
		block := config.GenesisBlock()
		txStatus := bc.NewTransactionStatus()

		for i := uint64(0); i < c.blockBestHeight; i++ {
			if err := store.SaveBlock(block, txStatus); err != nil {
				t.Fatal(err)
			}

			savedBlocks = append(savedBlocks, *block)
			block.PreviousBlockHash = block.Hash()
			block.Height++
		}

		index, err := store.LoadBlockIndex(c.stateBestHeight)
		if err != nil {
			t.Fatal(err)
		}

		for _, block := range savedBlocks {
			blockHash := block.Hash()
			if block.Height <= c.stateBestHeight != index.BlockExist(&blockHash) {
				t.Errorf("Error in load block index")
			}
		}
	}
}
