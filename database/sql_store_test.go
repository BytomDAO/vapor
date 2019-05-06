package database

import (
	"os"
	"testing"

	"github.com/vapor/common"
	"github.com/vapor/config"
	"github.com/vapor/consensus"
	dbm "github.com/vapor/database/db"
	_ "github.com/vapor/database/leveldb"
	"github.com/vapor/database/orm"
	_ "github.com/vapor/database/sqlite"
	"github.com/vapor/database/storage"
	"github.com/vapor/protocol"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
	"github.com/vapor/testutil"
)

func TestLoadBlockIndexFromSQLDB(t *testing.T) {
	sqlDB := dbm.NewSqlDB("sql", "sqlitedb", "temp")
	defer func() {
		sqlDB.Db().Close()
		os.RemoveAll("temp")
	}()

	sqlDB.Db().AutoMigrate(&orm.BlockHeader{}, &orm.Transaction{}, &orm.BlockStoreState{}, &orm.ClaimTxState{}, &orm.Utxo{})

	sqlStore := NewSQLStore(sqlDB)

	config.CommonConfig = config.DefaultConfig()
	config.CommonConfig.Consensus.SelfVoteSigners = append(config.CommonConfig.Consensus.SelfVoteSigners, "vsm1qkm743xmgnvh84pmjchq2s4tnfpgu9ae2f9slep")
	config.CommonConfig.Consensus.XPrv = "a8e281b615809046698fb0b0f2804a36d824d48fa443350f10f1b80649d39e5f1e85cf9855548915e36137345910606cbc8e7dd8497c831dce899ee6ac112445"
	for _, v := range config.CommonConfig.Consensus.SelfVoteSigners {
		address, err := common.DecodeAddress(v, &consensus.SoloNetParams)
		if err != nil {
			t.Fatal(err)
		}
		config.CommonConfig.Consensus.Signers = append(config.CommonConfig.Consensus.Signers, address)
	}
	block := config.GenesisBlock()
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)

	if err := sqlStore.SaveBlock(block, txStatus); err != nil {
		t.Fatal(err)
	}

	for block.Height <= 128 {
		preHash := block.Hash()
		block.PreviousBlockHash = preHash
		block.Height += 1
		if err := sqlStore.SaveBlock(block, txStatus); err != nil {
			t.Fatal(err)
		}

		if block.Height%32 != 0 {
			continue
		}

		for i := uint64(0); i < block.Height/32; i++ {
			if err := sqlStore.SaveBlock(block, txStatus); err != nil {
				t.Fatal(err)
			}
		}
	}

	if _, err := sqlStore.LoadBlockIndex(128); err != nil {
		t.Fatal(err)
	}
}

func TestLoadBlockIndexBestHeightFromSQLDB(t *testing.T) {
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

	sqlDB := dbm.NewSqlDB("sql", "sqlitedb", "temp")
	defer func() {
		sqlDB.Db().Close()
		os.RemoveAll("temp")
	}()

	sqlDB.Db().AutoMigrate(&orm.BlockHeader{}, &orm.Transaction{}, &orm.BlockStoreState{}, &orm.ClaimTxState{}, &orm.Utxo{})

	sqlStore := NewSQLStore(sqlDB)
	var savedBlocks []types.Block
	config.CommonConfig = config.DefaultConfig()
	config.CommonConfig.Consensus.SelfVoteSigners = append(config.CommonConfig.Consensus.SelfVoteSigners, "vsm1qkm743xmgnvh84pmjchq2s4tnfpgu9ae2f9slep")
	config.CommonConfig.Consensus.XPrv = "a8e281b615809046698fb0b0f2804a36d824d48fa443350f10f1b80649d39e5f1e85cf9855548915e36137345910606cbc8e7dd8497c831dce899ee6ac112445"
	for _, v := range config.CommonConfig.Consensus.SelfVoteSigners {
		address, err := common.DecodeAddress(v, &consensus.SoloNetParams)
		if err != nil {
			t.Fatal(err)
		}
		config.CommonConfig.Consensus.Signers = append(config.CommonConfig.Consensus.Signers, address)
	}

	for _, c := range cases {
		block := config.GenesisBlock()
		txStatus := bc.NewTransactionStatus()
		txStatus.SetStatus(0, false)

		for i := uint64(0); i < c.blockBestHeight; i++ {
			if err := sqlStore.SaveBlock(block, txStatus); err != nil {
				t.Fatal(err)
			}

			savedBlocks = append(savedBlocks, *block)
			block.PreviousBlockHash = block.Hash()
			block.Height++
		}

		index, err := sqlStore.LoadBlockIndex(c.stateBestHeight)
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

func TestLoadBlockIndexEqualsFromSQLDB(t *testing.T) {
	sqlDB := dbm.NewSqlDB("sql", "sqlitedb", "temp")
	defer func() {
		sqlDB.Db().Close()
		os.RemoveAll("temp")
	}()

	sqlDB.Db().AutoMigrate(&orm.BlockHeader{}, &orm.Transaction{}, &orm.BlockStoreState{}, &orm.ClaimTxState{}, &orm.Utxo{})

	sqlStore := NewSQLStore(sqlDB)
	config.CommonConfig = config.DefaultConfig()
	config.CommonConfig.Consensus.SelfVoteSigners = append(config.CommonConfig.Consensus.SelfVoteSigners, "vsm1qkm743xmgnvh84pmjchq2s4tnfpgu9ae2f9slep")
	config.CommonConfig.Consensus.XPrv = "a8e281b615809046698fb0b0f2804a36d824d48fa443350f10f1b80649d39e5f1e85cf9855548915e36137345910606cbc8e7dd8497c831dce899ee6ac112445"
	for _, v := range config.CommonConfig.Consensus.SelfVoteSigners {
		address, err := common.DecodeAddress(v, &consensus.SoloNetParams)
		if err != nil {
			t.Fatal(err)
		}
		config.CommonConfig.Consensus.Signers = append(config.CommonConfig.Consensus.Signers, address)
	}

	block := config.GenesisBlock()
	txStatus := bc.NewTransactionStatus()
	txStatus.SetStatus(0, false)

	expectBlockIndex := state.NewBlockIndex()
	var parent *state.BlockNode

	for block.Height <= 100 {
		if err := sqlStore.SaveBlock(block, txStatus); err != nil {
			t.Fatal(err)
		}

		if block.Height != 0 {
			parent = expectBlockIndex.GetNode(&block.PreviousBlockHash)
		}

		node, err := state.NewBlockNode(&block.BlockHeader, parent)
		if err != nil {
			t.Fatal(err)
		}

		expectBlockIndex.AddNode(node)
		block.PreviousBlockHash = block.Hash()
		block.Height++
	}

	index, err := sqlStore.LoadBlockIndex(100)
	if err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(expectBlockIndex, index) {
		t.Errorf("got block index:%v, expect block index:%v", index, expectBlockIndex)
	}
}
func TestSaveChainStatusToSQLDB(t *testing.T) {
	sqlDB := dbm.NewSqlDB("sql", "sqlitedb", "temp")
	defer func() {
		sqlDB.Db().Close()
		os.RemoveAll("temp")
	}()

	sqlDB.Db().AutoMigrate(&orm.BlockHeader{}, &orm.Transaction{}, &orm.BlockStoreState{}, &orm.ClaimTxState{}, &orm.Utxo{})

	sqlStore := NewSQLStore(sqlDB)

	node := &state.BlockNode{Height: 100, Hash: bc.Hash{V0: 0, V1: 1, V2: 2, V3: 3}}
	view := &state.UtxoViewpoint{
		Entries: map[bc.Hash]*storage.UtxoEntry{
			bc.Hash{V0: 1, V1: 2, V2: 3, V3: 4}: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 100, Spent: false},
			bc.Hash{V0: 1, V1: 2, V2: 3, V3: 4}: &storage.UtxoEntry{IsCoinBase: true, BlockHeight: 100, Spent: true},
			bc.Hash{V0: 1, V1: 1, V2: 3, V3: 4}: &storage.UtxoEntry{IsCoinBase: false, BlockHeight: 100, Spent: true},
		},
	}

	if err := sqlStore.SaveChainStatus(node, view); err != nil {
		t.Fatal(err)
	}

	expectStatus := &protocol.BlockStoreState{Height: node.Height, Hash: &node.Hash}
	if !testutil.DeepEqual(sqlStore.GetStoreStatus(), expectStatus) {
		t.Errorf("got block status:%v, expect block status:%v", sqlStore.GetStoreStatus(), expectStatus)
	}

	for hash, utxo := range view.Entries {
		if utxo.Spent && !utxo.IsCoinBase {
			continue
		}

		gotUtxo, err := sqlStore.GetUtxo(&hash)
		if err != nil {
			t.Fatal(err)
		}

		if !testutil.DeepEqual(utxo, gotUtxo) {
			t.Errorf("got utxo entry:%v, expect utxo entry:%v", gotUtxo, utxo)
		}
	}
}

func TestSaveBlockToSQLDB(t *testing.T) {
	sqlDB := dbm.NewSqlDB("sql", "sqlitedb", "temp")
	defer func() {
		sqlDB.Db().Close()
		os.RemoveAll("temp")
	}()

	sqlDB.Db().AutoMigrate(&orm.BlockHeader{}, &orm.Transaction{}, &orm.BlockStoreState{}, &orm.ClaimTxState{}, &orm.Utxo{})

	sqlStore := NewSQLStore(sqlDB)
	config.CommonConfig = config.DefaultConfig()
	config.CommonConfig.Consensus.SelfVoteSigners = append(config.CommonConfig.Consensus.SelfVoteSigners, "vsm1qkm743xmgnvh84pmjchq2s4tnfpgu9ae2f9slep")
	config.CommonConfig.Consensus.XPrv = "a8e281b615809046698fb0b0f2804a36d824d48fa443350f10f1b80649d39e5f1e85cf9855548915e36137345910606cbc8e7dd8497c831dce899ee6ac112445"
	for _, v := range config.CommonConfig.Consensus.SelfVoteSigners {
		address, err := common.DecodeAddress(v, &consensus.SoloNetParams)
		if err != nil {
			t.Fatal(err)
		}
		config.CommonConfig.Consensus.Signers = append(config.CommonConfig.Consensus.Signers, address)
	}

	block := config.GenesisBlock()
	status := &bc.TransactionStatus{Version: block.Version, VerifyStatus: []*bc.TxVerifyResult{{StatusFail: true}}}
	if err := sqlStore.SaveBlock(block, status); err != nil {
		t.Fatal(err)
	}

	blockHash := block.Hash()
	gotBlock, err := sqlStore.GetBlock(&blockHash)
	if err != nil {
		t.Fatal(err)
	}

	gotBlock.Transactions[0].Tx.SerializedSize = 0
	gotBlock.Transactions[0].SerializedSize = 0
	if !testutil.DeepEqual(block, gotBlock) {
		t.Errorf("got block:%v, expect block:%v", gotBlock, block)
	}

	gotStatus, err := sqlStore.GetTransactionStatus(&blockHash)
	if err != nil {
		t.Fatal(err)
	}

	if !testutil.DeepEqual(status, gotStatus) {
		t.Errorf("got status:%v, expect status:%v", gotStatus, status)
	}

}
