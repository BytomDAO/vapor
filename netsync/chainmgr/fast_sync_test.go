package chainmgr

import (
	"testing"
	"time"

	"github.com/vapor/consensus"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/test/mock"
	"github.com/vapor/testutil"
)

func TestBlockLocator(t *testing.T) {
	blocks := mockBlocks(nil, 500)
	cases := []struct {
		bestHeight uint64
		wantHeight []uint64
	}{
		{
			bestHeight: 0,
			wantHeight: []uint64{0},
		},
		{
			bestHeight: 1,
			wantHeight: []uint64{1, 0},
		},
		{
			bestHeight: 7,
			wantHeight: []uint64{7, 6, 5, 4, 3, 2, 1, 0},
		},
		{
			bestHeight: 10,
			wantHeight: []uint64{10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0},
		},
		{
			bestHeight: 100,
			wantHeight: []uint64{100, 99, 98, 97, 96, 95, 94, 93, 92, 91, 89, 85, 77, 61, 29, 0},
		},
		{
			bestHeight: 500,
			wantHeight: []uint64{500, 499, 498, 497, 496, 495, 494, 493, 492, 491, 489, 485, 477, 461, 429, 365, 237, 0},
		},
	}

	for i, c := range cases {
		mockChain := mock.NewChain(nil)
		fs := &fastSync{chain: mockChain}
		mockChain.SetBestBlockHeader(&blocks[c.bestHeight].BlockHeader)
		for i := uint64(0); i <= c.bestHeight; i++ {
			mockChain.SetBlockByHeight(i, blocks[i])
		}

		want := []*bc.Hash{}
		for _, i := range c.wantHeight {
			hash := blocks[i].Hash()
			want = append(want, &hash)
		}

		if got := fs.blockLocator(); !testutil.DeepEqual(got, want) {
			t.Errorf("case %d: got %v want %v", i, got, want)
		}
	}
}

func TestFastBlockSync(t *testing.T) {
	maxBlocksPerMsg = 10
	maxHeadersPerMsg = 10
	maxFastSyncBlocksNum = 200
	baseChain := mockBlocks(nil, 300)

	cases := []struct {
		syncTimeout time.Duration
		aBlocks     []*types.Block
		bBlocks     []*types.Block
		want        []*types.Block
		err         error
	}{
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     baseChain[:50],
			bBlocks:     baseChain[:301],
			want:        baseChain[:237],
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     baseChain[:2],
			bBlocks:     baseChain[:300],
			want:        baseChain[:202],
			err:         nil,
		},
	}

	for i, c := range cases {
		syncTimeout = c.syncTimeout
		a := mockSync(c.aBlocks, nil)
		b := mockSync(c.bBlocks, nil)
		netWork := NewNetWork()
		netWork.Register(a, "192.168.0.1", "test node A", consensus.SFFullNode|consensus.SFFastSync)
		netWork.Register(b, "192.168.0.2", "test node B", consensus.SFFullNode|consensus.SFFastSync)
		if B2A, A2B, err := netWork.HandsShake(a, b); err != nil {
			t.Errorf("fail on peer hands shake %v", err)
		} else {
			go B2A.postMan()
			go A2B.postMan()
		}
		a.blockKeeper.syncPeer = a.peers.GetPeer("test node B")
		a.blockKeeper.fastSync.setSyncPeer(a.blockKeeper.syncPeer)

		if err := a.blockKeeper.fastSync.process(); errors.Root(err) != c.err {
			t.Errorf("case %d: got %v want %v", i, err, c.err)
		}

		got := []*types.Block{}
		for i := uint64(0); i <= a.chain.BestBlockHeight(); i++ {
			block, err := a.chain.GetBlockByHeight(i)
			if err != nil {
				t.Errorf("case %d got err %v", i, err)
			}
			got = append(got, block)
		}
		if !testutil.DeepEqual(got, c.want) {
			t.Errorf("case %d: got %v want %v", i, got, c.want)
		}
	}
}

func TestLocateBlocks(t *testing.T) {
	maxBlocksPerMsg = 5
	blocks := mockBlocks(nil, 100)
	cases := []struct {
		locator    []uint64
		stopHash   bc.Hash
		wantHeight []uint64
	}{
		{
			locator:    []uint64{20},
			stopHash:   blocks[100].Hash(),
			wantHeight: []uint64{20, 21, 22, 23, 24},
		},
	}

	mockChain := mock.NewChain(nil)
	fs := &fastSync{chain: mockChain}
	for _, block := range blocks {
		mockChain.SetBlockByHeight(block.Height, block)
	}

	for i, c := range cases {
		locator := []*bc.Hash{}
		for _, i := range c.locator {
			hash := blocks[i].Hash()
			locator = append(locator, &hash)
		}

		want := []*types.Block{}
		for _, i := range c.wantHeight {
			want = append(want, blocks[i])
		}

		got, _ := fs.locateBlocks(locator, &c.stopHash)
		if !testutil.DeepEqual(got, want) {
			t.Errorf("case %d: got %v want %v", i, got, want)
		}
	}
}

func TestLocateHeaders(t *testing.T) {
	maxHeadersPerMsg = 10
	blocks := mockBlocks(nil, 150)
	blocksHash := []bc.Hash{}
	for _, block := range blocks {
		blocksHash = append(blocksHash, block.Hash())
	}

	cases := []struct {
		chainHeight uint64
		locator     []uint64
		stopHash    *bc.Hash
		skip        uint64
		wantHeight  []uint64
		err         bool
	}{
		{
			chainHeight: 100,
			locator:     []uint64{90},
			stopHash:    &blocksHash[100],
			skip:        0,
			wantHeight:  []uint64{90, 91, 92, 93, 94, 95, 96, 97, 98, 99},
			err:         false,
		},
		{
			chainHeight: 100,
			locator:     []uint64{20},
			stopHash:    &blocksHash[24],
			skip:        0,
			wantHeight:  []uint64{20, 21, 22, 23, 24},
			err:         false,
		},
		{
			chainHeight: 100,
			locator:     []uint64{20},
			stopHash:    &blocksHash[20],
			wantHeight:  []uint64{20},
			err:         false,
		},
		{
			chainHeight: 100,
			locator:     []uint64{20},
			stopHash:    &blocksHash[120],
			wantHeight:  []uint64{},
			err:         false,
		},
		{
			chainHeight: 100,
			locator:     []uint64{120, 70},
			stopHash:    &blocksHash[78],
			wantHeight:  []uint64{70, 71, 72, 73, 74, 75, 76, 77, 78},
			err:         false,
		},
		{
			chainHeight: 100,
			locator:     []uint64{15},
			stopHash:    &blocksHash[10],
			skip:        10,
			wantHeight:  []uint64{},
			err:         false,
		},
		{
			chainHeight: 100,
			locator:     []uint64{15},
			stopHash:    &blocksHash[80],
			skip:        10,
			wantHeight:  []uint64{15, 26, 37, 48, 59, 70},
			err:         false,
		},
		{
			chainHeight: 100,
			locator:     []uint64{0},
			stopHash:    &blocksHash[100],
			skip:        9,
			wantHeight:  []uint64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90},
			err:         false,
		},
	}

	for i, c := range cases {
		mockChain := mock.NewChain(nil)
		fs := &fastSync{chain: mockChain}
		for i := uint64(0); i <= c.chainHeight; i++ {
			mockChain.SetBlockByHeight(i, blocks[i])
		}

		locator := []*bc.Hash{}
		for _, i := range c.locator {
			hash := blocks[i].Hash()
			locator = append(locator, &hash)
		}

		want := []*types.BlockHeader{}
		for _, i := range c.wantHeight {
			want = append(want, &blocks[i].BlockHeader)
		}

		got, err := fs.locateHeaders(locator, c.stopHash, c.skip, maxHeadersPerMsg)
		if err != nil != c.err {
			t.Errorf("case %d: got %v want err = %v", i, err, c.err)
		}
		if !testutil.DeepEqual(got, want) {
			t.Errorf("case %d: got %v want %v", i, got, want)
		}
	}
}
