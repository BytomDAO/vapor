package chainmgr

import (
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/bytom/vapor/consensus"
	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/test/mock"
	"github.com/bytom/vapor/testutil"
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
	tmp, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatalf("failed to create temporary data folder: %v", err)
	}
	testDBA := dbm.NewDB("testdba", "leveldb", tmp)
	testDBB := dbm.NewDB("testdbb", "leveldb", tmp)
	defer func() {
		testDBA.Close()
		testDBB.Close()
		os.RemoveAll(tmp)
	}()

	maxSizeOfSyncSkeleton = 11
	numOfBlocksSkeletonGap = 10
	maxNumOfBlocksPerSync = numOfBlocksSkeletonGap * uint64(maxSizeOfSyncSkeleton-1)
	fastSyncPivotGap = uint64(5)
	minGapStartFastSync = uint64(6)

	defer func() {
		maxSizeOfSyncSkeleton = 11
		numOfBlocksSkeletonGap = maxNumOfBlocksPerMsg
		maxNumOfBlocksPerSync = numOfBlocksSkeletonGap * uint64(maxSizeOfSyncSkeleton-1)
		fastSyncPivotGap = uint64(64)
		minGapStartFastSync = uint64(128)
		requireHeadersTimeout = 30 * time.Second
	}()

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
			want:        baseChain[:150],
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     baseChain[:2],
			bBlocks:     baseChain[:300],
			want:        baseChain[:102],
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     baseChain[:2],
			bBlocks:     baseChain[:53],
			want:        baseChain[:48],
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     baseChain[:2],
			bBlocks:     baseChain[:53],
			want:        baseChain[:48],
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     baseChain[:2],
			bBlocks:     baseChain[:10],
			want:        baseChain[:5],
			err:         nil,
		},
		{
			syncTimeout: 0 * time.Second,
			aBlocks:     baseChain[:50],
			bBlocks:     baseChain[:301],
			want:        baseChain[:50],
			err:         errSkeletonSize,
		},
	}

	for i, c := range cases {
		a := mockSync(c.aBlocks, nil, testDBA)
		b := mockSync(c.bBlocks, nil, testDBB)
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

		requireHeadersTimeout = c.syncTimeout
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
