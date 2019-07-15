package chainmgr

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/vapor/consensus"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/errors"
	msgs "github.com/vapor/netsync/messages"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/test/mock"
	"github.com/vapor/testutil"
)

func TestRegularBlockSync(t *testing.T) {
	baseChain := mockBlocks(nil, 50)
	chainX := append(baseChain, mockBlocks(baseChain[50], 60)...)
	chainY := append(baseChain, mockBlocks(baseChain[50], 70)...)
	chainZ := append(baseChain, mockBlocks(baseChain[50], 200)...)

	cases := []struct {
		syncTimeout time.Duration
		aBlocks     []*types.Block
		bBlocks     []*types.Block
		want        []*types.Block
		err         error
	}{
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     baseChain[:20],
			bBlocks:     baseChain[:50],
			want:        baseChain[:50],
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     chainX,
			bBlocks:     chainY,
			want:        chainY,
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     chainX[:52],
			bBlocks:     chainY[:53],
			want:        chainY[:53],
			err:         nil,
		},
		{
			syncTimeout: 30 * time.Second,
			aBlocks:     chainX[:52],
			bBlocks:     chainZ,
			want:        chainZ[:180],
			err:         nil,
		},
	}
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

	for i, c := range cases {
		a := mockSync(c.aBlocks, nil, testDBA)
		b := mockSync(c.bBlocks, nil, testDBB)
		netWork := NewNetWork()
		netWork.Register(a, "192.168.0.1", "test node A", consensus.SFFullNode)
		netWork.Register(b, "192.168.0.2", "test node B", consensus.SFFullNode)
		if B2A, A2B, err := netWork.HandsShake(a, b); err != nil {
			t.Errorf("fail on peer hands shake %v", err)
		} else {
			go B2A.postMan()
			go A2B.postMan()
		}

		a.blockKeeper.syncPeer = a.peers.GetPeer("test node B")
		if err := a.blockKeeper.regularBlockSync(); errors.Root(err) != c.err {
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

func TestRequireBlock(t *testing.T) {
	tmp, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatalf("failed to create temporary data folder: %v", err)
	}
	testDBA := dbm.NewDB("testdba", "leveldb", tmp)
	testDBB := dbm.NewDB("testdbb", "leveldb", tmp)
	defer func() {
		testDBB.Close()
		testDBA.Close()
		os.RemoveAll(tmp)
	}()

	blocks := mockBlocks(nil, 5)
	a := mockSync(blocks[:1], nil, testDBA)
	b := mockSync(blocks[:5], nil, testDBB)
	netWork := NewNetWork()
	netWork.Register(a, "192.168.0.1", "test node A", consensus.SFFullNode)
	netWork.Register(b, "192.168.0.2", "test node B", consensus.SFFullNode)
	if B2A, A2B, err := netWork.HandsShake(a, b); err != nil {
		t.Errorf("fail on peer hands shake %v", err)
	} else {
		go B2A.postMan()
		go A2B.postMan()
	}

	a.blockKeeper.syncPeer = a.peers.GetPeer("test node B")
	b.blockKeeper.syncPeer = b.peers.GetPeer("test node A")
	cases := []struct {
		syncTimeout   time.Duration
		testNode      *Manager
		requireHeight uint64
		want          *types.Block
		err           error
	}{
		{
			syncTimeout:   30 * time.Second,
			testNode:      a,
			requireHeight: 4,
			want:          blocks[4],
			err:           nil,
		},
		{
			syncTimeout:   1 * time.Millisecond,
			testNode:      b,
			requireHeight: 4,
			want:          nil,
			err:           errRequestTimeout,
		},
	}

	defer func() {
		requireBlockTimeout = 20 * time.Second
	}()

	for i, c := range cases {
		requireBlockTimeout = c.syncTimeout
		got, err := c.testNode.blockKeeper.msgFetcher.requireBlock(c.testNode.blockKeeper.syncPeer.ID(), c.requireHeight)
		if !testutil.DeepEqual(got, c.want) {
			t.Errorf("case %d: got %v want %v", i, got, c.want)
		}
		if errors.Root(err) != c.err {
			t.Errorf("case %d: got %v want %v", i, err, c.err)
		}
	}
}

func TestSendMerkleBlock(t *testing.T) {
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

	cases := []struct {
		txCount        int
		relatedTxIndex []int
	}{
		{
			txCount:        10,
			relatedTxIndex: []int{0, 2, 5},
		},
		{
			txCount:        0,
			relatedTxIndex: []int{},
		},
		{
			txCount:        10,
			relatedTxIndex: []int{},
		},
		{
			txCount:        5,
			relatedTxIndex: []int{0, 1, 2, 3, 4},
		},
		{
			txCount:        20,
			relatedTxIndex: []int{1, 6, 3, 9, 10, 19},
		},
	}

	for _, c := range cases {
		blocks := mockBlocks(nil, 2)
		targetBlock := blocks[1]
		txs, bcTxs := mockTxs(c.txCount)
		var err error

		targetBlock.Transactions = txs
		if targetBlock.TransactionsMerkleRoot, err = types.TxMerkleRoot(bcTxs); err != nil {
			t.Fatal(err)
		}

		spvNode := mockSync(blocks, nil, testDBA)
		blockHash := targetBlock.Hash()
		var statusResult *bc.TransactionStatus
		if statusResult, err = spvNode.chain.GetTransactionStatus(&blockHash); err != nil {
			t.Fatal(err)
		}

		if targetBlock.TransactionStatusHash, err = types.TxStatusMerkleRoot(statusResult.VerifyStatus); err != nil {
			t.Fatal(err)
		}

		fullNode := mockSync(blocks, nil, testDBB)
		netWork := NewNetWork()
		netWork.Register(spvNode, "192.168.0.1", "spv_node", consensus.SFFastSync)
		netWork.Register(fullNode, "192.168.0.2", "full_node", consensus.DefaultServices)

		var F2S *P2PPeer
		if F2S, _, err = netWork.HandsShake(spvNode, fullNode); err != nil {
			t.Errorf("fail on peer hands shake %v", err)
		}

		completed := make(chan error)
		go func() {
			msgBytes := <-F2S.msgCh
			_, msg, _ := decodeMessage(msgBytes)
			switch m := msg.(type) {
			case *msgs.MerkleBlockMessage:
				var relatedTxIDs []*bc.Hash
				for _, rawTx := range m.RawTxDatas {
					tx := &types.Tx{}
					if err := tx.UnmarshalText(rawTx); err != nil {
						completed <- err
					}

					relatedTxIDs = append(relatedTxIDs, &tx.ID)
				}
				var txHashes []*bc.Hash
				for _, hashByte := range m.TxHashes {
					hash := bc.NewHash(hashByte)
					txHashes = append(txHashes, &hash)
				}
				if ok := types.ValidateTxMerkleTreeProof(txHashes, m.Flags, relatedTxIDs, targetBlock.TransactionsMerkleRoot); !ok {
					completed <- errors.New("validate tx fail")
				}

				var statusHashes []*bc.Hash
				for _, statusByte := range m.StatusHashes {
					hash := bc.NewHash(statusByte)
					statusHashes = append(statusHashes, &hash)
				}
				var relatedStatuses []*bc.TxVerifyResult
				for _, statusByte := range m.RawTxStatuses {
					status := &bc.TxVerifyResult{}
					err := json.Unmarshal(statusByte, status)
					if err != nil {
						completed <- err
					}
					relatedStatuses = append(relatedStatuses, status)
				}
				if ok := types.ValidateStatusMerkleTreeProof(statusHashes, m.Flags, relatedStatuses, targetBlock.TransactionStatusHash); !ok {
					completed <- errors.New("validate status fail")
				}

				completed <- nil
			}
		}()

		spvPeer := fullNode.peers.GetPeer("spv_node")
		for i := 0; i < len(c.relatedTxIndex); i++ {
			spvPeer.AddFilterAddress(txs[c.relatedTxIndex[i]].Outputs[0].ControlProgram())
		}
		msg := &msgs.GetMerkleBlockMessage{RawHash: targetBlock.Hash().Byte32()}
		fullNode.handleGetMerkleBlockMsg(spvPeer, msg)
		if err := <-completed; err != nil {
			t.Fatal(err)
		}
	}
}

func TestLocateBlocks(t *testing.T) {
	maxNumOfBlocksPerMsg = 5
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
	bk := &blockKeeper{chain: mockChain}
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

		got, _ := bk.locateBlocks(locator, &c.stopHash)
		if !testutil.DeepEqual(got, want) {
			t.Errorf("case %d: got %v want %v", i, got, want)
		}
	}
}

func TestLocateHeaders(t *testing.T) {
	defer func() {
		maxNumOfHeadersPerMsg = 1000
	}()
	maxNumOfHeadersPerMsg = 10
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
			wantHeight:  []uint64{15, 26, 37, 48, 59, 70, 80},
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
		bk := &blockKeeper{chain: mockChain}
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

		got, err := bk.locateHeaders(locator, c.stopHash, c.skip, maxNumOfHeadersPerMsg)
		if err != nil != c.err {
			t.Errorf("case %d: got %v want err = %v", i, err, c.err)
		}
		if !testutil.DeepEqual(got, want) {
			t.Errorf("case %d: got %v want %v", i, got, want)
		}
	}
}
