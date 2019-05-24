package consensus

import (
	log "github.com/sirupsen/logrus"
	"gopkg.in/karalabe/cookiejar.v2/collections/prque"

	"github.com/vapor/netsync/peers"
	"github.com/vapor/protocol/bc"
)

const (
	maxBlockDistance = 64
	maxMsgSetSize    = 128
	newBlockChSize   = 64
)

// blockFetcher is responsible for accumulating block announcements from various peers
// and scheduling them for retrieval.
type blockFetcher struct {
	chain Chain
	peers *peers.PeerSet

	newBlockCh chan *blockMsg
	queue      *prque.Prque
	msgSet     map[bc.Hash]*blockMsg
}

//NewBlockFetcher creates a block fetcher to retrieve blocks of the new mined.
func newBlockFetcher(chain Chain, peers *peers.PeerSet) *blockFetcher {
	f := &blockFetcher{
		chain:      chain,
		peers:      peers,
		newBlockCh: make(chan *blockMsg, newBlockChSize),
		queue:      prque.New(),
		msgSet:     make(map[bc.Hash]*blockMsg),
	}
	go f.blockProcessor()
	return f
}

func (f *blockFetcher) blockProcessor() {
	for {
		height := f.chain.BestBlockHeight()
		for !f.queue.Empty() {
			msg := f.queue.PopItem().(*blockMsg)
			if msg.block.Height > height+1 {
				f.queue.Push(msg, -float32(msg.block.Height))
				break
			}

			f.insert(msg)
			delete(f.msgSet, msg.block.Hash())
		}
		f.add(<-f.newBlockCh)
	}
}

func (f *blockFetcher) add(msg *blockMsg) {
	bestHeight := f.chain.BestBlockHeight()
	if len(f.msgSet) > maxMsgSetSize || bestHeight > msg.block.Height || msg.block.Height-bestHeight > maxBlockDistance {
		return
	}

	blockHash := msg.block.Hash()
	if _, ok := f.msgSet[blockHash]; !ok {
		f.msgSet[blockHash] = msg
		f.queue.Push(msg, -float32(msg.block.Height))
		log.WithFields(log.Fields{
			"module":       logModule,
			"block height": msg.block.Height,
			"block hash":   blockHash.String(),
		}).Debug("blockFetcher receive mine block")
	}
}

func (f *blockFetcher) insert(msg *blockMsg) {
	isOrphan, err := f.chain.ProcessBlock(msg.block)
	if err != nil {
		peer := f.peers.GetPeer(msg.peerID)
		if peer == nil {
			return
		}

		f.peers.AddBanScore(msg.peerID, 20, 0, err.Error())
		return
	}

	if isOrphan {
		return
	}

	hash := msg.block.Hash()
	f.peers.SetStatus(msg.peerID, msg.block.Height, &hash)

	proposedMsg, err := NewBlockProposeBroadcastMsg(msg.block, ConsensusChannel)
	if err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("failed on create new propose block msg")
		return
	}

	if err := f.peers.BroadcastMsg(proposedMsg); err != nil {
		log.WithFields(log.Fields{"module": logModule, "err": err}).Error("failed on broadcast proposed block")
		return
	}
}

func (f *blockFetcher) processNewBlock(msg *blockMsg) {
	f.newBlockCh <- msg
}
