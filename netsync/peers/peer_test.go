package peers

import (
	"net"
	"reflect"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/tendermint/tmlibs/flowrate"
	"github.com/vapor/consensus"
	"github.com/vapor/p2p/security"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

var (
	peer1ID = "PEER1"
	peer2ID = "PEER2"
	peer3ID = "PEER3"
	peer4ID = "PEER4"

	block1000Hash = bc.NewHash([32]byte{0x01, 0x02})
	block2000Hash = bc.NewHash([32]byte{0x02, 0x03})
	block3000Hash = bc.NewHash([32]byte{0x03, 0x04})
)

type basePeer struct {
	id          string
	serviceFlag consensus.ServiceFlag
	isLan       bool
}

func (bp *basePeer) Addr() net.Addr {
	return nil
}

func (bp *basePeer) ID() string {
	return bp.id
}

func (bp *basePeer) RemoteAddrHost() string {
	switch bp.ID() {
	case peer1ID:
		return peer1ID
	case peer2ID:
		return peer2ID
	case peer3ID:
		return peer3ID
	case peer4ID:
		return peer4ID
	}
	return ""
}

func (bp *basePeer) ServiceFlag() consensus.ServiceFlag {
	return bp.serviceFlag
}

func (bp *basePeer) TrafficStatus() (*flowrate.Status, *flowrate.Status) {
	return nil, nil
}

func (bp *basePeer) TrySend(byte, interface{}) bool {
	return true
}

func (bp *basePeer) IsLAN() bool {
	return bp.isLan
}

func TestSetPeerStatus(t *testing.T) {
	peer := newPeer(&basePeer{})
	height := uint64(100)
	hash := bc.NewHash([32]byte{0x1, 0x2})
	peer.SetBestStatus(height, &hash)
	if peer.Height() != height {
		t.Fatalf("test set best status err. got %d want %d", peer.Height(), height)
	}
}

func TestSetIrreversibleStatus(t *testing.T) {
	peer := newPeer(&basePeer{})
	height := uint64(100)
	hash := bc.NewHash([32]byte{0x1, 0x2})
	peer.SetIrreversibleStatus(height, &hash)
	if peer.IrreversibleHeight() != height {
		t.Fatalf("test set Irreversible status err. got %d want %d", peer.Height(), height)
	}
}

func TestAddFilterAddresses(t *testing.T) {
	peer := newPeer(&basePeer{})
	tx := types.NewTx(types.TxData{
		Inputs: []*types.TxInput{
			types.NewSpendInput(nil, bc.Hash{}, bc.NewAssetID([32]byte{1}), 5, 1, []byte("spendProgram")),
		},
		Outputs: []*types.TxOutput{
			types.NewIntraChainOutput(bc.NewAssetID([32]byte{3}), 8, []byte("outProgram")),
		},
	})

	peer.AddFilterAddresses([][]byte{[]byte("spendProgram")})
	if !peer.isRelatedTx(tx) {
		t.Fatal("test filter addresses error.")
	}

	peer.AddFilterAddresses([][]byte{[]byte("testProgram")})
	if peer.isRelatedTx(tx) {
		t.Fatal("test filter addresses error.")
	}
}

func TestFilterClear(t *testing.T) {
	peer := newPeer(&basePeer{})
	tx := types.NewTx(types.TxData{
		Inputs: []*types.TxInput{
			types.NewSpendInput(nil, bc.Hash{}, bc.NewAssetID([32]byte{1}), 5, 1, []byte("spendProgram")),
		},
		Outputs: []*types.TxOutput{
			types.NewIntraChainOutput(bc.NewAssetID([32]byte{3}), 8, []byte("outProgram")),
		},
	})

	peer.AddFilterAddresses([][]byte{[]byte("spendProgram")})
	if !peer.isRelatedTx(tx) {
		t.Fatal("test filter addresses error.")
	}

	peer.FilterClear()
	if peer.isRelatedTx(tx) {
		t.Fatal("test filter addresses error.")
	}
}

func TestGetRelatedTxAndStatus(t *testing.T) {
	peer := newPeer(&basePeer{})
	txs := []*types.Tx{
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, bc.Hash{}, bc.NewAssetID([32]byte{1}), 5, 1, []byte("spendProgram1")),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(bc.NewAssetID([32]byte{3}), 8, []byte("outProgram1")),
			},
		}),
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, bc.Hash{}, bc.NewAssetID([32]byte{1}), 5, 1, []byte("spendProgram2")),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(bc.NewAssetID([32]byte{3}), 8, []byte("outProgram2")),
			},
		}),
		types.NewTx(types.TxData{
			Inputs: []*types.TxInput{
				types.NewSpendInput(nil, bc.Hash{}, bc.NewAssetID([32]byte{1}), 5, 1, []byte("spendProgram3")),
			},
			Outputs: []*types.TxOutput{
				types.NewIntraChainOutput(bc.NewAssetID([32]byte{3}), 8, []byte("outProgram3")),
			},
		}),
	}
	txStatuses := &bc.TransactionStatus{
		VerifyStatus: []*bc.TxVerifyResult{{StatusFail: true}, {StatusFail: false}, {StatusFail: false}},
	}
	peer.AddFilterAddresses([][]byte{[]byte("spendProgram1"), []byte("outProgram3")})
	gotTxs, gotStatus := peer.getRelatedTxAndStatus(txs, txStatuses)
	if len(gotTxs) != 2 {
		t.Error("TestGetRelatedTxAndStatus txs size error")
	}

	if !reflect.DeepEqual(*gotTxs[0].Tx, *txs[0].Tx) {
		t.Errorf("txs msg test err: got %s\nwant %s", spew.Sdump(gotTxs[0].Tx), spew.Sdump(txs[0].Tx))
	}

	if !reflect.DeepEqual(*gotTxs[1].Tx, *txs[2].Tx) {
		t.Errorf("txs msg test err: got %s\nwant %s", spew.Sdump(gotTxs[1].Tx), spew.Sdump(txs[2].Tx))
	}

	if gotStatus[0].StatusFail != true || gotStatus[1].StatusFail != false {
		t.Error("TestGetRelatedTxAndStatus txs status error")
	}
}

type basePeerSet struct {
}

func (bp *basePeerSet) StopPeerGracefully(string) {

}

func (bp *basePeerSet) IsBanned(ip string, level byte, reason string) bool {
	switch ip {
	case peer1ID:
		return true
	case peer2ID:
		return false
	case peer3ID:
		return true
	case peer4ID:
		return false
	}
	return false
}

func TestMarkBlock(t *testing.T) {
	ps := NewPeerSet(&basePeerSet{})
	ps.AddPeer(&basePeer{id: peer1ID})
	ps.AddPeer(&basePeer{id: peer2ID})
	ps.AddPeer(&basePeer{id: peer3ID})

	blockHash := bc.NewHash([32]byte{0x01, 0x02})
	ps.MarkBlock(peer1ID, &blockHash)
	targetPeers := []string{peer2ID, peer3ID}

	peers := ps.PeersWithoutBlock(blockHash)
	if len(peers) != len(targetPeers) {
		t.Fatalf("test mark block err. Number of target peers %d got %d", 1, len(peers))
	}

	for _, targetPeer := range targetPeers {
		flag := false
		for _, gotPeer := range peers {
			if gotPeer == targetPeer {
				flag = true
				break
			}
		}
		if !flag {
			t.Errorf("test mark block err. can't found target peer %s ", targetPeer)
		}
	}
}

func TestMarkStatus(t *testing.T) {
	ps := NewPeerSet(&basePeerSet{})
	ps.AddPeer(&basePeer{id: peer1ID})
	ps.AddPeer(&basePeer{id: peer2ID})
	ps.AddPeer(&basePeer{id: peer3ID})

	height := uint64(1024)
	ps.MarkStatus(peer1ID, height)
	targetPeers := []string{peer2ID, peer3ID}

	peers := ps.peersWithoutNewStatus(height)
	if len(peers) != len(targetPeers) {
		t.Fatalf("test mark status err. Number of target peers %d got %d", 1, len(peers))
	}

	for _, targetPeer := range targetPeers {
		flag := false
		for _, gotPeer := range peers {
			if gotPeer.ID() == targetPeer {
				flag = true
				break
			}
		}
		if !flag {
			t.Errorf("test mark status err. can't found target peer %s ", targetPeer)
		}
	}
}

func TestMarkBlockSignature(t *testing.T) {
	ps := NewPeerSet(&basePeerSet{})
	ps.AddPeer(&basePeer{id: peer1ID})
	ps.AddPeer(&basePeer{id: peer2ID})
	ps.AddPeer(&basePeer{id: peer3ID})

	signature := []byte{0x01, 0x02}
	ps.MarkBlockSignature(peer1ID, signature)
	targetPeers := []string{peer2ID, peer3ID}

	peers := ps.PeersWithoutSignature(signature)
	if len(peers) != len(targetPeers) {
		t.Fatalf("test mark block signature err. Number of target peers %d got %d", 1, len(peers))
	}

	for _, targetPeer := range targetPeers {
		flag := false
		for _, gotPeer := range peers {
			if gotPeer == targetPeer {
				flag = true
				break
			}
		}
		if !flag {
			t.Errorf("test mark block signature err. can't found target peer %s ", targetPeer)
		}
	}
}

func TestMarkTx(t *testing.T) {
	ps := NewPeerSet(&basePeerSet{})
	ps.AddPeer(&basePeer{id: peer1ID})
	ps.AddPeer(&basePeer{id: peer2ID})
	ps.AddPeer(&basePeer{id: peer3ID})

	txHash := bc.NewHash([32]byte{0x01, 0x02})
	ps.MarkTx(peer1ID, txHash)
	peers := ps.peersWithoutTx(&txHash)
	targetPeers := []string{peer2ID, peer3ID}
	if len(peers) != len(targetPeers) {
		t.Fatalf("test mark tx err. Number of target peers %d got %d", 1, len(peers))
	}

	for _, targetPeer := range targetPeers {
		flag := false
		for _, gotPeer := range peers {
			if gotPeer.ID() == targetPeer {
				flag = true
				break
			}
		}
		if !flag {
			t.Errorf("test mark tx err. can't found target peer %s ", targetPeer)
		}
	}
}

func TestSetStatus(t *testing.T) {
	ps := NewPeerSet(&basePeerSet{})
	ps.AddPeer(&basePeer{id: peer1ID, serviceFlag: consensus.SFFullNode})
	ps.AddPeer(&basePeer{id: peer2ID, serviceFlag: consensus.SFFullNode})
	ps.AddPeer(&basePeer{id: peer3ID, serviceFlag: consensus.SFFastSync})
	ps.AddPeer(&basePeer{id: peer4ID, serviceFlag: consensus.SFFullNode, isLan: true})
	ps.SetStatus(peer1ID, 1000, &block1000Hash)
	ps.SetStatus(peer2ID, 2000, &block2000Hash)
	ps.SetStatus(peer3ID, 3000, &block3000Hash)
	ps.SetStatus(peer4ID, 2000, &block2000Hash)
	targetPeer := peer4ID

	peer := ps.BestPeer(consensus.SFFullNode)

	if peer.ID() != targetPeer {
		t.Fatalf("test set status err. Name of target peer %s got %s", peer4ID, peer.ID())
	}
}

func TestIrreversibleStatus(t *testing.T) {
	ps := NewPeerSet(&basePeerSet{})
	ps.AddPeer(&basePeer{id: peer1ID, serviceFlag: consensus.SFFullNode})
	ps.AddPeer(&basePeer{id: peer2ID, serviceFlag: consensus.SFFullNode})
	ps.AddPeer(&basePeer{id: peer3ID, serviceFlag: consensus.SFFastSync})
	ps.AddPeer(&basePeer{id: peer4ID, serviceFlag: consensus.SFFastSync, isLan: true})
	ps.SetIrreversibleStatus(peer1ID, 1000, &block1000Hash)
	ps.SetIrreversibleStatus(peer2ID, 2000, &block2000Hash)
	ps.SetIrreversibleStatus(peer3ID, 3000, &block3000Hash)
	ps.SetIrreversibleStatus(peer4ID, 3000, &block3000Hash)
	targetPeer := peer4ID
	peer := ps.BestIrreversiblePeer(consensus.SFFastSync)

	if peer.ID() != targetPeer {
		t.Fatalf("test set status err. Name of target peer %s got %s", peer4ID, peer.ID())
	}
}

func TestGetPeersByHeight(t *testing.T) {
	ps := NewPeerSet(&basePeerSet{})
	ps.AddPeer(&basePeer{id: peer1ID, serviceFlag: consensus.SFFullNode})
	ps.AddPeer(&basePeer{id: peer2ID, serviceFlag: consensus.SFFullNode})
	ps.AddPeer(&basePeer{id: peer3ID, serviceFlag: consensus.SFFastSync})
	ps.AddPeer(&basePeer{id: peer4ID, serviceFlag: consensus.SFFullNode, isLan: true})
	ps.SetStatus(peer1ID, 1000, &block1000Hash)
	ps.SetStatus(peer2ID, 2000, &block2000Hash)
	ps.SetStatus(peer3ID, 3000, &block3000Hash)
	ps.SetStatus(peer4ID, 2000, &block2000Hash)
	peers := ps.GetPeersByHeight(2000)
	targetPeers := []string{peer2ID, peer3ID, peer4ID}
	if len(peers) != len(targetPeers) {
		t.Fatalf("test get peers by height err. Number of target peers %d got %d", 3, len(peers))
	}

	for _, targetPeer := range targetPeers {
		flag := false
		for _, gotPeer := range peers {
			if gotPeer.ID() == targetPeer {
				flag = true
				break
			}
		}
		if !flag {
			t.Errorf("test get peers by height err. can't found target peer %s ", targetPeer)
		}
	}
}

func TestRemovePeer(t *testing.T) {
	ps := NewPeerSet(&basePeerSet{})
	ps.AddPeer(&basePeer{id: peer1ID})
	ps.AddPeer(&basePeer{id: peer2ID})

	ps.RemovePeer(peer1ID)
	if peer := ps.GetPeer(peer1ID); peer != nil {
		t.Fatalf("remove peer %s err", peer1ID)
	}

	if peer := ps.GetPeer(peer2ID); peer == nil {
		t.Fatalf("Error remove peer %s err", peer2ID)
	}
}

func TestProcessIllegal(t *testing.T) {
	ps := NewPeerSet(&basePeerSet{})
	ps.AddPeer(&basePeer{id: peer1ID})
	ps.AddPeer(&basePeer{id: peer2ID})

	ps.ProcessIllegal(peer1ID, security.LevelMsgIllegal, "test")
	if peer := ps.GetPeer(peer1ID); peer != nil {
		t.Fatalf("remove peer %s err", peer1ID)
	}

	ps.ProcessIllegal(peer2ID, security.LevelMsgIllegal, "test")
	if peer := ps.GetPeer(peer2ID); peer == nil {
		t.Fatalf("Error remove peer %s err", peer2ID)
	}
}