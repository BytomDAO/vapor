package protocol

import (
	log "github.com/sirupsen/logrus"

	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
	"github.com/bytom/vapor/protocol/validation"
)

var (
	// ErrBadBlock is returned when a block is invalid.
	ErrBadBlock = errors.New("invalid block")
	// ErrBadStateRoot is returned when the computed assets merkle root
	// disagrees with the one declared in a block header.
	ErrBadStateRoot = errors.New("invalid state merkle root")
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

// GetBlockByHeight return a block by given height
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

func (c *Chain) calcReorganizeChain(beginAttach *types.BlockHeader, beginDetach *types.BlockHeader) ([]*types.BlockHeader, []*types.BlockHeader, error) {
	var err error
	var attachBlockHeaders []*types.BlockHeader
	var detachBlockHeaders []*types.BlockHeader

	for attachBlockHeader, detachBlockHeader := beginAttach, beginDetach; detachBlockHeader.Hash() != attachBlockHeader.Hash(); {
		var attachRollback, detachRollBack bool
		if attachRollback = attachBlockHeader.Height >= detachBlockHeader.Height; attachRollback {
			attachBlockHeaders = append([]*types.BlockHeader{attachBlockHeader}, attachBlockHeaders...)
		}

		if detachRollBack = attachBlockHeader.Height <= detachBlockHeader.Height; detachRollBack {
			detachBlockHeaders = append(detachBlockHeaders, detachBlockHeader)
		}

		if attachRollback {
			attachBlockHeader, err = c.store.GetBlockHeader(&attachBlockHeader.PreviousBlockHash)
			if err != nil {
				return nil, nil, err
			}
		}

		if detachRollBack {
			detachBlockHeader, err = c.store.GetBlockHeader(&detachBlockHeader.PreviousBlockHash)
			if err != nil {
				return nil, nil, err
			}
		}
	}
	return attachBlockHeaders, detachBlockHeaders, nil
}

func (c *Chain) connectBlock(block *types.Block) (err error) {
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

	consensusResult, err := c.getBestConsensusResult()
	if err != nil {
		return err
	}

	if err := consensusResult.ApplyBlock(block); err != nil {
		return err
	}

	for _, p := range c.subProtocols {
		if err := c.syncProtocolStatus(p); err != nil {
			return errors.Wrap(err, p.Name(), "sync sub protocol status")
		}

		if err := p.ApplyBlock(block); err != nil {
			return errors.Wrap(err, p.Name(), "sub protocol connect block")
		}
	}

	if err := c.applyBlockSign(&block.BlockHeader); err != nil {
		return err
	}

	irrBlockHeader := c.lastIrrBlockHeader
	if c.isIrreversible(&block.BlockHeader) && block.Height > irrBlockHeader.Height {
		irrBlockHeader = &block.BlockHeader
	}

	if err := c.setState(&block.BlockHeader, irrBlockHeader, []*types.BlockHeader{&block.BlockHeader}, utxoView, []*state.ConsensusResult{consensusResult}); err != nil {
		return err
	}

	for _, tx := range block.Transactions {
		c.txPool.RemoveTransaction(&tx.Tx.ID)
	}
	return nil
}

func (c *Chain) detachBlock(detachBlockHeader *types.BlockHeader, consensusResult *state.ConsensusResult, utxoView *state.UtxoViewpoint) (*types.Block, error) {
	detachHash := detachBlockHeader.Hash()
	block, err := c.store.GetBlock(&detachHash)
	if err != nil {
		return block, err
	}

	detachBlock := types.MapBlock(block)
	if err := consensusResult.DetachBlock(block); err != nil {
		return block, err
	}

	if err := c.store.GetTransactionsUtxo(utxoView, detachBlock.Transactions); err != nil {
		return block, err
	}

	txStatus, err := c.GetTransactionStatus(&detachBlock.ID)
	if err != nil {
		return block, err
	}

	if err := utxoView.DetachBlock(detachBlock, txStatus); err != nil {
		return block, err
	}

	for _, p := range c.subProtocols {
		if err := p.DetachBlock(block); err != nil {
			return block, errors.Wrap(err, p.Name(), "sub protocol detach block")
		}
	}

	log.WithFields(log.Fields{"module": logModule, "height": detachBlockHeader.Height, "hash": detachHash.String()}).Debug("detach from mainchain")
	return block, nil
}

func (c *Chain) syncSubProtocols() error {
	for _, p := range c.subProtocols {
		if err := c.syncProtocolStatus(p); err != nil {
			return errors.Wrap(err, p.Name(), "sync sub protocol status")
		}
	}
	return nil
}

// Rollback rollback the chain from one blockHeight to targetBlockHeight
// WARNING: we recommend to use this only in commond line
func (c *Chain) Rollback(targetHeight uint64) error {
	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	utxoView := state.NewUtxoViewpoint()
	consensusResult, err := c.getBestConsensusResult()
	if err != nil {
		return err
	}

	if err = c.syncSubProtocols(); err != nil {
		return err
	}

	targetBlockHeader, err := c.GetHeaderByHeight(targetHeight)
	if err != nil {
		return err
	}

	_, deletedBlockHeaders, err := c.calcReorganizeChain(targetBlockHeader, c.bestBlockHeader)
	if err != nil {
		return err
	}

	deletedBlocks := []*types.Block{}
	for _, deletedBlockHeader := range deletedBlockHeaders {
		block, err := c.detachBlock(deletedBlockHeader, consensusResult, utxoView)
		if err != nil {
			return err
		}

		deletedBlocks = append(deletedBlocks, block)
	}

	setIrrBlockHeader := c.lastIrrBlockHeader
	if c.lastIrrBlockHeader.Height > targetBlockHeader.Height {
		setIrrBlockHeader = targetBlockHeader
	}

	startSeq := state.CalcVoteSeq(c.bestBlockHeader.Height)

	if err = c.setState(targetBlockHeader, setIrrBlockHeader, nil, utxoView, []*state.ConsensusResult{consensusResult.Fork()}); err != nil {
		return err
	}

	for _, block := range deletedBlocks {
		if err := c.store.DeleteBlock(block); err != nil {
			return err
		}
	}

	endSeq := state.CalcVoteSeq(targetHeight)
	for nowSeq := startSeq; nowSeq > endSeq; nowSeq-- {
		if err := c.store.DeleteConsensusResult(nowSeq); err != nil {
			return err
		}
	}

	return nil
}

func (c *Chain) reorganizeChain(blockHeader *types.BlockHeader) error {
	attachBlockHeaders, detachBlockHeaders, err := c.calcReorganizeChain(blockHeader, c.bestBlockHeader)
	if err != nil {
		return err
	}

	utxoView := state.NewUtxoViewpoint()
	consensusResults := []*state.ConsensusResult{}
	consensusResult, err := c.getBestConsensusResult()
	if err != nil {
		return err
	}

	if err = c.syncSubProtocols(); err != nil {
		return err
	}

	txsToRestore := map[bc.Hash]*types.Tx{}
	for _, detachBlockHeader := range detachBlockHeaders {
		b, err := c.detachBlock(detachBlockHeader, consensusResult, utxoView)
		if err != nil {
			return err
		}

		for _, tx := range b.Transactions {
			txsToRestore[tx.ID] = tx
		}
	}

	txsToRemove := map[bc.Hash]*types.Tx{}
	irrBlockHeader := c.lastIrrBlockHeader
	for _, attachBlockHeader := range attachBlockHeaders {
		attachHash := attachBlockHeader.Hash()
		b, err := c.store.GetBlock(&attachHash)
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

		if err := consensusResult.ApplyBlock(b); err != nil {
			return err
		}

		for _, p := range c.subProtocols {
			if err := p.ApplyBlock(b); err != nil {
				return errors.Wrap(err, p.Name(), "sub protocol attach block")
			}
		}

		if consensusResult.IsFinalize() {
			consensusResults = append(consensusResults, consensusResult.Fork())
		}

		if err := c.applyBlockSign(attachBlockHeader); err != nil {
			return err
		}

		if c.isIrreversible(attachBlockHeader) && attachBlockHeader.Height > irrBlockHeader.Height {
			irrBlockHeader = attachBlockHeader
		}

		for _, tx := range b.Transactions {
			if _, ok := txsToRestore[tx.ID]; !ok {
				txsToRemove[tx.ID] = tx
			} else {
				delete(txsToRestore, tx.ID)
			}
		}

		blockHash := blockHeader.Hash()
		log.WithFields(log.Fields{"module": logModule, "height": blockHeader.Height, "hash": blockHash.String()}).Debug("attach from mainchain")
	}

	if len(detachBlockHeaders) > 0 &&
		detachBlockHeaders[len(detachBlockHeaders)-1].Height <= c.lastIrrBlockHeader.Height &&
		irrBlockHeader.Height <= c.lastIrrBlockHeader.Height {
		return errors.New("rollback block below the height of irreversible block")
	}

	consensusResults = append(consensusResults, consensusResult.Fork())
	if err := c.setState(blockHeader, irrBlockHeader, attachBlockHeaders, utxoView, consensusResults); err != nil {
		return err
	}

	for txHash := range txsToRemove {
		c.txPool.RemoveTransaction(&txHash)
	}

	for _, tx := range txsToRestore {
		// the number of restored Tx should be very small or most of time ZERO
		// Error returned from validation is ignored, tx could still be lost if validation fails.
		// TODO: adjust tx timestamp so that it won't starve in pool.
		if _, err := c.validateTx(tx, blockHeader); err != nil {
			log.WithFields(log.Fields{"module": logModule, "tx_id": tx.Tx.ID.String(), "error": err}).Info("restore tx fail")
		}
	}

	if len(txsToRestore) > 0 {
		log.WithFields(log.Fields{"module": logModule, "num": len(txsToRestore)}).Debug("restore txs back to pool")
	}
	return nil
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

	consensusResult, err := c.GetConsensusResultByHash(&block.PreviousBlockHash)
	if err != nil {
		return err
	}

	rewards, err := consensusResult.GetCoinbaseRewards(parent.Height)
	if err != nil {
		return err
	}

	bcBlock := types.MapBlock(block)
	if err := validation.ValidateBlock(bcBlock, parent, rewards); err != nil {
		return errors.Sub(ErrBadBlock, err)
	}

	for _, p := range c.subProtocols {
		if err := p.ValidateBlock(block, bcBlock.TransactionStatus.GetVerifyStatus()); err != nil {
			return errors.Wrap(err, "sub protocol save block")
		}
	}

	if err := c.store.SaveBlock(block, bcBlock.TransactionStatus); err != nil {
		return err
	}

	c.orphanManage.Delete(&bcBlock.ID)
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
		log.WithFields(log.Fields{"module": logModule, "hash": blockHash.String(), "height": block.Height}).Debug("block has been processed")
		return c.orphanManage.BlockExist(&blockHash), nil
	}

	c.markTransactions(block.Transactions...)

	if _, err := c.store.GetBlockHeader(&block.PreviousBlockHash); err != nil {
		c.orphanManage.Add(block)
		return true, nil
	}

	if err := c.saveBlock(block); err != nil {
		return false, err
	}

	bestBlock := c.saveSubBlock(block)
	bestBlockHeader := &bestBlock.BlockHeader

	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	if bestBlockHeader.PreviousBlockHash == c.bestBlockHeader.Hash() {
		log.WithFields(log.Fields{"module": logModule}).Debug("append block to the end of mainchain")
		return false, c.connectBlock(bestBlock)
	}

	if bestBlockHeader.Height > c.bestBlockHeader.Height {
		log.WithFields(log.Fields{"module": logModule}).Debug("start to reorganize chain")
		return false, c.reorganizeChain(bestBlockHeader)
	}
	return false, nil
}
