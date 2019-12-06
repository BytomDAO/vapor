package chainmgr

import (
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	dbm "github.com/bytom/vapor/database/leveldb"
	"github.com/bytom/vapor/test/mock"
)

func TestBlockProcess(t *testing.T) {
	tmp, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmp)

	testDB := dbm.NewDB("testdb", "leveldb", tmp)
	defer testDB.Close()

	s := newStorage(testDB)
	mockChain := mock.NewChain(nil)
	blockNum := 200
	blocks := mockBlocks(nil, uint64(blockNum))
	for i := 0; i <= blockNum/2; i++ {
		mockChain.SetBlockByHeight(uint64(i), blocks[i])
		mockChain.SetBestBlockHeader(&blocks[i].BlockHeader)
	}

	if err := s.writeBlocks("testPeer", blocks); err != nil {
		t.Fatal(err)
	}

	bp := newBlockProcessor(mockChain, s, nil)
	downloadNotifyCh := make(chan struct{}, 1)
	ProcessStopCh := make(chan struct{})
	var wg sync.WaitGroup
	go func() {
		time.Sleep(1 * time.Second)
		close(downloadNotifyCh)
	}()
	wg.Add(1)
	bp.process(downloadNotifyCh, ProcessStopCh, uint64(blockNum/2), &wg)
	if bp.chain.BestBlockHeight() != uint64(blockNum) {
		t.Fatalf("TestBlockProcess fail: got %d want %d", bp.chain.BestBlockHeight(), blockNum)
	}
}
