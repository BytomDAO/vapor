package protocol

import (
	log "github.com/sirupsen/logrus"

	"github.com/vapor/config"
	"github.com/vapor/errors"
	"github.com/vapor/event"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
	"github.com/vapor/protocol/validation"
)

var (
	// ErrBadBlock is returned when a block is invalid.
	ErrBadBlock = errors.New("invalid block")
	// ErrBadStateRoot is returned when the computed assets merkle root
	// disagrees with the one declared in a block header.
	ErrBadStateRoot           = errors.New("invalid state merkle root")
	errBelowIrreversibleBlock = errors.New("the height of block below irreversible block")
)

// BlockExist check is a block in chain or orphan
func (c *Chain) BlockExist(hash *bc.Hash) bool {
	if _, err := c.store.GetBlockHeader(hash); err == nil {
		return true
	}
	return c.orphanManage.BlockExist(hash)
}

// GetBlockByHash return a block by given hash
func (c *Chain) GetBlockByHash(hash *bc.Hash) (*types.Block, error) {
	return c.store.GetBlock(hash)
}

// GetBlockByHeight return a block header by given height
func (c *Chain) GetBlockByHeight(height uint64) (*types.Block, error) {
	hash, err := c.store.GetMainChainHash(height)
	if err != nil {
		return nil, errors.Wrap(err, "can't find block in given height")
	}
	return c.store.GetBlock(hash)
}

// GetHeaderByHash return a block header by given hash
func (c *Chain) GetHeaderByHash(hash *bc.Hash) (*types.BlockHeader, error) {
	return c.store.GetBlockHeader(hash)
}

// GetHeaderByHeight return a block header by given height
func (c *Chain) GetHeaderByHeight(height uint64) (*types.BlockHeader, error) {
	hash, err := c.store.GetMainChainHash(height)
	if err != nil {
		return nil, errors.Wrap(err, "can't find block header in given height")
	}
	return c.store.GetBlockHeader(hash)
}

func (c *Chain) calcReorganizeNodes(node *types.BlockHeader) ([]*types.BlockHeader, []*types.BlockHeader, error) {
	var attachNodes []*types.BlockHeader
	var detachNodes []*types.BlockHeader
	var err error

	attachNode := node
	for {
		getBlockHash, err := c.store.GetMainChainHash(attachNode.Height)
		if err != nil {
			return nil, nil, err
		}

		if *getBlockHash == attachNode.Hash() {
			break
		}

		attachNodes = append([]*types.BlockHeader{attachNode}, attachNodes...)
		attachNode, err = c.store.GetBlockHeader(&attachNode.PreviousBlockHash)
		if err != nil {
			return nil, nil, err
		}
	}

	detachNode := c.bestNode
	for {
		if detachNode.Hash() == attachNode.Hash() {
			break
		}

		detachNodes = append(detachNodes, detachNode)
		detachNode, err = c.store.GetBlockHeader(&detachNode.PreviousBlockHash)
		if err != nil {
			return nil, nil, err
		}
	}
	return attachNodes, detachNodes, nil
}

func (c *Chain) connectBlock(block *types.Block) (err error) {
	irreversibleNode := c.bestIrreversibleNode
	bcBlock := types.MapBlock(block)
	if bcBlock.TransactionStatus, err = c.store.GetTransactionStatus(&bcBlock.ID); err != nil {
		return err
	}

	utxoView := state.NewUtxoViewpoint()
	if err := c.store.GetTransactionsUtxo(utxoView, bcBlock.Transactions); err != nil {
		return err
	}
	if err := utxoView.ApplyBlock(bcBlock, bcBlock.TransactionStatus); err != nil {
		return err
	}

	voteResult, err := c.getBestVoteResult()
	if err != nil {
		return err
	}
	if err := voteResult.ApplyBlock(block); err != nil {
		return err
	}

	// pointer copy?
	blockHeader := &block.BlockHeader
	if c.isIrreversible(blockHeader) && block.Height > irreversibleNode.Height {
		irreversibleNode = blockHeader
	}

	if err := c.setState(blockHeader, irreversibleNode, utxoView, []*state.VoteResult{voteResult}); err != nil {
		return err
	}

	for _, tx := range block.Transactions {
		c.txPool.RemoveTransaction(&tx.Tx.ID)
	}
	return nil
}

func (c *Chain) reorganizeChain(node *types.BlockHeader) error {
	attachNodes, detachNodes, err := c.calcReorganizeNodes(node)
	if err != nil {
		return err
	}

	utxoView := state.NewUtxoViewpoint()
	voteResults := []*state.VoteResult{}
	irreversibleNode := c.bestIrreversibleNode
	voteResult, err := c.getBestVoteResult()
	if err != nil {
		return err
	}

	for _, detachNode := range detachNodes {
		detachNodeHash := detachNode.Hash()
		b, err := c.store.GetBlock(&detachNodeHash)
		if err != nil {
			return err
		}

		detachBlock := types.MapBlock(b)
		if err := c.store.GetTransactionsUtxo(utxoView, detachBlock.Transactions); err != nil {
			return err
		}

		txStatus, err := c.GetTransactionStatus(&detachBlock.ID)
		if err != nil {
			return err
		}

		if err := utxoView.DetachBlock(detachBlock, txStatus); err != nil {
			return err
		}

		if err := voteResult.DetachBlock(b); err != nil {
			return err
		}

		nodeHash := node.Hash()
		log.WithFields(log.Fields{"module": logModule, "height": node.Height, "hash": nodeHash.String()}).Debug("detach from mainchain")
	}

	for _, attachNode := range attachNodes {
		attachNodeHash := attachNode.Hash()
		b, err := c.store.GetBlock(&attachNodeHash)
		if err != nil {
			return err
		}

		attachBlock := types.MapBlock(b)
		if err := c.store.GetTransactionsUtxo(utxoView, attachBlock.Transactions); err != nil {
			return err
		}

		txStatus, err := c.GetTransactionStatus(&attachBlock.ID)
		if err != nil {
			return err
		}

		if err := utxoView.ApplyBlock(attachBlock, txStatus); err != nil {
			return err
		}

		if err := voteResult.ApplyBlock(b); err != nil {
			return err
		}

		if voteResult.IsFinalize() {
			voteResults = append(voteResults, voteResult.Fork())
		}

		if c.isIrreversible(attachNode) && attachNode.Height > irreversibleNode.Height {
			irreversibleNode = attachNode
		}

		nodeHash := node.Hash()
		log.WithFields(log.Fields{"module": logModule, "height": node.Height, "hash": nodeHash.String()}).Debug("attach from mainchain")
	}

	if detachNodes[len(detachNodes)-1].Height <= c.bestIrreversibleNode.Height && irreversibleNode.Height <= c.bestIrreversibleNode.Height {
		return errors.New("rollback block below the height of irreversible block")
	}
	voteResults = append(voteResults, voteResult.Fork())
	return c.setState(node, irreversibleNode, utxoView, voteResults)
}

// SaveBlock will validate and save block into storage
func (c *Chain) saveBlock(block *types.Block) error {
	if err := c.validateSign(block); err != nil {
		return errors.Sub(ErrBadBlock, err)
	}

	parent, err := c.store.GetBlockHeader(&block.PreviousBlockHash)
	if err != nil {
		return err
	}

	bcBlock := types.MapBlock(block)
	if err := validation.ValidateBlock(bcBlock, parent); err != nil {
		return errors.Sub(ErrBadBlock, err)
	}

	signature, err := c.SignBlock(block)
	if err != nil {
		return errors.Sub(ErrBadBlock, err)
	}

	if err := c.store.SaveBlock(block, bcBlock.TransactionStatus); err != nil {
		return err
	}
	c.orphanManage.Delete(&bcBlock.ID)

	if len(signature) != 0 {
		xPub := config.CommonConfig.PrivateKey().XPub()
		if err := c.eventDispatcher.Post(event.BlockSignatureEvent{BlockHash: block.Hash(), Signature: signature, XPub: xPub[:]}); err != nil {
			return err
		}
	}
	return nil
}

func (c *Chain) saveSubBlock(block *types.Block) *types.Block {
	blockHash := block.Hash()
	prevOrphans, ok := c.orphanManage.GetPrevOrphans(&blockHash)
	if !ok {
		return block
	}

	bestBlock := block
	for _, prevOrphan := range prevOrphans {
		orphanBlock, ok := c.orphanManage.Get(prevOrphan)
		if !ok {
			log.WithFields(log.Fields{"module": logModule, "hash": prevOrphan.String()}).Warning("saveSubBlock fail to get block from orphanManage")
			continue
		}
		if err := c.saveBlock(orphanBlock); err != nil {
			log.WithFields(log.Fields{"module": logModule, "hash": prevOrphan.String(), "height": orphanBlock.Height}).Warning("saveSubBlock fail to save block")
			continue
		}

		if subBestBlock := c.saveSubBlock(orphanBlock); subBestBlock.Height > bestBlock.Height {
			bestBlock = subBestBlock
		}
	}
	return bestBlock
}

type processBlockResponse struct {
	isOrphan bool
	err      error
}

type processBlockMsg struct {
	block *types.Block
	reply chan processBlockResponse
}

// ProcessBlock is the entry for chain update
func (c *Chain) ProcessBlock(block *types.Block) (bool, error) {
	reply := make(chan processBlockResponse, 1)
	c.processBlockCh <- &processBlockMsg{block: block, reply: reply}
	response := <-reply
	return response.isOrphan, response.err
}

func (c *Chain) blockProcesser() {
	for msg := range c.processBlockCh {
		isOrphan, err := c.processBlock(msg.block)
		msg.reply <- processBlockResponse{isOrphan: isOrphan, err: err}
	}
}

// ProcessBlock is the entry for handle block insert
func (c *Chain) processBlock(block *types.Block) (bool, error) {
	blockHash := block.Hash()
	if c.BlockExist(&blockHash) {
		log.WithFields(log.Fields{"module": logModule, "hash": blockHash.String(), "height": block.Height}).Info("block has been processed")
		return c.orphanManage.BlockExist(&blockHash), nil
	}

	if _, err := c.store.GetBlockHeader(&block.PreviousBlockHash); err != nil {
		c.orphanManage.Add(block)
		return true, nil
	}

	if err := c.saveBlock(block); err != nil {
		return false, err
	}

	bestBlock := c.saveSubBlock(block)
	bestBlockHeader := &block.BlockHeader
	parentBestBlockHeader, err := c.store.GetBlockHeader(&bestBlockHeader.PreviousBlockHash)
	if err != nil {
		return false, err
	}

	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	if parentBestBlockHeader.Hash() == c.bestNode.Hash() {
		log.WithFields(log.Fields{"module": logModule}).Debug("append block to the end of mainchain")
		return false, c.connectBlock(bestBlock)
	}

	if bestBlockHeader.Height > c.bestNode.Height {
		log.WithFields(log.Fields{"module": logModule}).Debug("start to reorganize chain")
		return false, c.reorganizeChain(bestBlockHeader)
	}
	return false, nil
}
