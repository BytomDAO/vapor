package protocol

import (
	"testing"

	"github.com/vapor/common"
	"github.com/vapor/config"
	"github.com/vapor/consensus"
	"github.com/vapor/protocol/state"
	"github.com/vapor/testutil"
)

func TestCalcReorganizeNodes(t *testing.T) {
	c := &Chain{index: state.NewBlockIndex()}
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
	header := config.GenesisBlock().BlockHeader
	initNode, err := state.NewBlockNode(&header, nil)
	if err != nil {
		t.Fatal(err)
	}

	c.index.AddNode(initNode)
	var wantAttachNodes []*state.BlockNode
	var wantDetachNodes []*state.BlockNode

	mainChainNode := initNode
	for i := 1; i <= 7; i++ {
		header.Height = uint64(i)
		mainChainNode, err = state.NewBlockNode(&header, mainChainNode)
		if err != nil {
			t.Fatal(err)
		}
		wantDetachNodes = append([]*state.BlockNode{mainChainNode}, wantDetachNodes...)
		c.index.AddNode(mainChainNode)
	}
	c.bestNode = mainChainNode
	c.index.SetMainChain(mainChainNode)

	sideChainNode := initNode
	for i := 1; i <= 13; i++ {
		header.Height = uint64(i)
		sideChainNode, err = state.NewBlockNode(&header, sideChainNode)
		if err != nil {
			t.Fatal(err)
		}
		wantAttachNodes = append(wantAttachNodes, sideChainNode)
		c.index.AddNode(sideChainNode)
	}

	getAttachNodes, getDetachNodes := c.calcReorganizeNodes(sideChainNode)
	if !testutil.DeepEqual(wantAttachNodes, getAttachNodes) {
		t.Errorf("attach nodes want %v but get %v", wantAttachNodes, getAttachNodes)
	}
	if !testutil.DeepEqual(wantDetachNodes, getDetachNodes) {
		t.Errorf("detach nodes want %v but get %v", wantDetachNodes, getDetachNodes)
	}
}
