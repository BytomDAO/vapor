package test

import (
	"os"
	"testing"

	"github.com/bytom/vapor/application/mov"
	movDatabase "github.com/bytom/vapor/application/mov/database"
	"github.com/bytom/vapor/database"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/protocol"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
	"github.com/bytom/vapor/testutil"
)

type chainStatus struct {
	blockHeight uint64
	blockHash   bc.Hash
}

type chainBlock struct {
	block       *types.Block
	inMainChain bool
}

var blocks = map[uint64][]*types.Block{
	0: {
		{
			BlockHeader: types.BlockHeader{
				Height:            0,
				Timestamp:         1585814309,
				PreviousBlockHash: bc.Hash{},
			},
		},
	},
	1: {
		// prev block is [0][0]
		{
			BlockHeader: types.BlockHeader{
				Height:            1,
				Timestamp:         1585814310,
				PreviousBlockHash: testutil.MustDecodeHash("2e5406c82fe34f6ee44fe694b05ffc8fb5918a026415b086df03fb03760b42a9"),
			},
		},
		// prev block is [0][0]
		{
			BlockHeader: types.BlockHeader{
				Height:            1,
				Timestamp:         1585814311,
				PreviousBlockHash: testutil.MustDecodeHash("2e5406c82fe34f6ee44fe694b05ffc8fb5918a026415b086df03fb03760b42a9"),
			},
		},
	},
	2: {
		// prev block is [1][0]
		{
			BlockHeader: types.BlockHeader{
				Height:            2,
				Timestamp:         1585814320,
				PreviousBlockHash: testutil.MustDecodeHash("5bc198f4c0198e7e8b52173a82836cfd3f124d88bf052f53390948d845bf6fe0"),
			},
		},
	},
}

func TestSyncProtocolStatus(t *testing.T) {
	cases := []struct {
		desc            string
		savedBlocks     []*chainBlock
		startHeight     uint64
		startHash       *bc.Hash
		wantChainStatus *chainStatus
	}{
		{
			desc: "start height from 0, mov is not init",
			savedBlocks: []*chainBlock{
				{
					block:       blocks[0][0],
					inMainChain: true,
				},
				{
					block:       blocks[1][0],
					inMainChain: true,
				},
				{
					block:       blocks[2][0],
					inMainChain: true,
				},
			},
			startHeight: 0,
			wantChainStatus: &chainStatus{
				blockHeight: 2,
				blockHash:   blocks[2][0].Hash(),
			},
		},
		{
			desc: "start height from 1, mov is not init",
			savedBlocks: []*chainBlock{
				{
					block:       blocks[0][0],
					inMainChain: true,
				},
				{
					block:       blocks[1][0],
					inMainChain: true,
				},
				{
					block:       blocks[2][0],
					inMainChain: true,
				},
			},
			startHeight: 1,
			wantChainStatus: &chainStatus{
				blockHeight: 2,
				blockHash:   blocks[2][0].Hash(),
			},
		},
		{
			desc: "start height from 1, state of mov is not sync completed",
			savedBlocks: []*chainBlock{
				{
					block:       blocks[0][0],
					inMainChain: true,
				},
				{
					block:       blocks[1][0],
					inMainChain: true,
				},
				{
					block:       blocks[2][0],
					inMainChain: true,
				},
			},
			startHeight: 1,
			startHash:   hashPtr(blocks[1][0].Hash()),
			wantChainStatus: &chainStatus{
				blockHeight: 2,
				blockHash:   blocks[2][0].Hash(),
			},
		},
		{
			desc: "chain status of mov is forked",
			savedBlocks: []*chainBlock{
				{
					block:       blocks[0][0],
					inMainChain: true,
				},
				{
					block:       blocks[1][0],
					inMainChain: true,
				},
				{
					block:       blocks[1][1],
					inMainChain: false,
				},
				{
					block:       blocks[2][0],
					inMainChain: true,
				},
			},
			startHeight: 1,
			startHash:   hashPtr(blocks[1][1].Hash()),
			wantChainStatus: &chainStatus{
				blockHeight: 2,
				blockHash:   blocks[2][0].Hash(),
			},
		},
	}

	defer os.RemoveAll("temp")

	for i, c := range cases {
		chainDB := dbm.NewDB("core", "leveldb", "temp")
		store := database.NewStore(chainDB)
		if err := initStore(store, c.savedBlocks); err != nil {
			t.Fatal(err)
		}

		movDB := dbm.NewDB("mov", "leveldb", "temp")
		movCore := mov.NewCoreWithDB(movDatabase.NewLevelDBMovStore(movDB), c.startHeight)
		if c.startHash != nil {
			if err := movCore.InitChainStatus(c.startHash); err != nil {
				t.Fatal(err)
			}
		}

		_, err := protocol.NewChain(store, nil, []protocol.SubProtocol{movCore}, nil)
		if err != nil {
			t.Fatal(err)
		}

		gotHeight, gotHash, err := movCore.ChainStatus()
		if err != nil {
			t.Fatal(err)
		}

		if gotHeight != c.wantChainStatus.blockHeight || *gotHash != c.wantChainStatus.blockHash {
			t.Logf("#%d(%s): got chain status of sub protocol is not equals want chain status", i, c.desc)
		}

		movDB.Close()
		chainDB.Close()
		os.RemoveAll("temp")
	}
}

func initStore(store *database.Store, savedBlocks []*chainBlock) error {
	var mainBlockHeaders []*types.BlockHeader
	for _, block := range savedBlocks {
		if err := store.SaveBlock(block.block, bc.NewTransactionStatus()); err != nil {
			return err
		}

		if block.inMainChain {
			mainBlockHeaders = append(mainBlockHeaders, &block.block.BlockHeader)
		}

		last := len(mainBlockHeaders) - 1
		if err := store.SaveChainStatus(mainBlockHeaders[last], mainBlockHeaders[last], mainBlockHeaders, state.NewUtxoViewpoint(), nil); err != nil {
			return err
		}
	}
	return nil
}

func hashPtr(hash bc.Hash) *bc.Hash {
	return &hash
}

func TestBlockHash(t *testing.T) {
	blockHash := blocks[1][0].Hash()
	t.Log(blockHash.String())
}
