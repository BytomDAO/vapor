package protocol

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/common"
	"github.com/vapor/consensus"
	engine "github.com/vapor/consensus/consensus"
	"github.com/vapor/errors"
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
	ErrBadStateRoot = errors.New("invalid state merkle root")
)

// BlockExist check is a block in chain or orphan
func (c *Chain) BlockExist(hash *bc.Hash) bool {
	return c.index.BlockExist(hash) || c.orphanManage.BlockExist(hash)
}

// GetBlockByHash return a block by given hash
func (c *Chain) GetBlockByHash(hash *bc.Hash) (*types.Block, error) {
	return c.store.GetBlock(hash)
}

// GetBlockByHeight return a block header by given height
func (c *Chain) GetBlockByHeight(height uint64) (*types.Block, error) {
	node := c.index.NodeByHeight(height)
	if node == nil {
		return nil, errors.New("can't find block in given height")
	}
	return c.store.GetBlock(&node.Hash)
}

// GetHeaderByHash return a block header by given hash
func (c *Chain) GetHeaderByHash(hash *bc.Hash) (*types.BlockHeader, error) {
	node := c.index.GetNode(hash)
	if node == nil {
		return nil, errors.New("can't find block header in given hash")
	}
	return node.BlockHeader(), nil
}

// GetHeaderByHeight return a block header by given height
func (c *Chain) GetHeaderByHeight(height uint64) (*types.BlockHeader, error) {
	node := c.index.NodeByHeight(height)
	if node == nil {
		return nil, errors.New("can't find block header in given height")
	}
	return node.BlockHeader(), nil
}

func (c *Chain) calcReorganizeNodes(node *state.BlockNode) ([]*state.BlockNode, []*state.BlockNode) {
	var attachNodes []*state.BlockNode
	var detachNodes []*state.BlockNode

	attachNode := node
	for c.index.NodeByHeight(attachNode.Height) != attachNode {
		attachNodes = append([]*state.BlockNode{attachNode}, attachNodes...)
		attachNode = attachNode.Parent
	}

	detachNode := c.bestNode
	for detachNode != attachNode {
		detachNodes = append(detachNodes, detachNode)
		detachNode = detachNode.Parent
	}
	return attachNodes, detachNodes
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
	node := c.index.GetNode(&bcBlock.ID)
	if err := c.setState(node, utxoView); err != nil {
		return err
	}
	for _, tx := range block.Transactions {
		for key, value := range tx.Entries {
			switch value.(type) {
			case *bc.Claim:
				c.store.SetWithdrawSpent(&key)
			default:
				continue
			}
		}
		c.txPool.RemoveTransaction(&tx.Tx.ID)
	}
	return nil
}

func (c *Chain) reorganizeChain(node *state.BlockNode) error {
	attachNodes, detachNodes := c.calcReorganizeNodes(node)
	utxoView := state.NewUtxoViewpoint()

	for _, detachNode := range detachNodes {
		b, err := c.store.GetBlock(&detachNode.Hash)
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

		log.WithFields(log.Fields{"height": node.Height, "hash": node.Hash.String()}).Debug("detach from mainchain")
	}

	for _, attachNode := range attachNodes {
		b, err := c.store.GetBlock(&attachNode.Hash)
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

		log.WithFields(log.Fields{"height": node.Height, "hash": node.Hash.String()}).Debug("attach from mainchain")
	}

	return c.setState(node, utxoView)
}

// SaveBlock will validate and save block into storage
func (c *Chain) saveBlock(block *types.Block) error {
	bcBlock := types.MapBlock(block)
	parent := c.index.GetNode(&block.PreviousBlockHash)

	if err := validation.ValidateBlock(bcBlock, parent, block, c); err != nil {
		return errors.Sub(ErrBadBlock, err)
	}

	if err := c.store.SaveBlock(block, bcBlock.TransactionStatus); err != nil {
		return err
	}

	c.orphanManage.Delete(&bcBlock.ID)
	node, err := state.NewBlockNode(&block.BlockHeader, parent)
	if err != nil {
		return err
	}

	c.index.AddNode(node)
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
			log.WithFields(log.Fields{"hash": prevOrphan.String()}).Warning("saveSubBlock fail to get block from orphanManage")
			continue
		}
		if err := c.saveBlock(orphanBlock); err != nil {
			log.WithFields(log.Fields{"hash": prevOrphan.String(), "height": orphanBlock.Height}).Warning("saveSubBlock fail to save block")
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
		log.WithFields(log.Fields{"hash": blockHash.String(), "height": block.Height}).Info("block has been processed")
		return c.orphanManage.BlockExist(&blockHash), nil
	}

	if parent := c.index.GetNode(&block.PreviousBlockHash); parent == nil {
		c.orphanManage.Add(block)
		return true, nil
	}

	if err := c.saveBlock(block); err != nil {
		return false, err
	}

	bestBlock := c.saveSubBlock(block)
	bestBlockHash := bestBlock.Hash()
	bestNode := c.index.GetNode(&bestBlockHash)

	if bestNode.Parent == c.bestNode {
		log.Debug("append block to the end of mainchain")
		return false, c.connectBlock(bestBlock)
	}

	if bestNode.Height > c.bestNode.Height && bestNode.WorkSum.Cmp(c.bestNode.WorkSum) >= 0 {
		log.Debug("start to reorganize chain")
		return false, c.reorganizeChain(bestNode)
	}
	return false, nil
}

func (c *Chain) ProcessDPoSConnectBlock(block *types.Block) {
	mapTxFee := c.CalculateBalance(block, true)
	fmt.Println(mapTxFee)
}

func DoVoting(block *types.Block, mapTxFee map[bc.Hash]uint64) error {
	for _, tx := range block.Transactions {
		to := tx.Outputs[0]
		var delegate engine.TypedData

		if err := json.Unmarshal(tx.TxData.ReferenceData, delegate); err != nil && to.Amount > 0 {
			continue
		}
		var (
			address common.Address
			err     error
		)
		address, err = common.NewAddressWitnessPubKeyHash(to.ControlProgram[2:], &consensus.ActiveNetParams)
		if err != nil {
			address, err = common.NewAddressWitnessScriptHash(to.ControlProgram[2:], &consensus.ActiveNetParams)
			if err != nil {
				return errors.New("ControlProgram cannot be converted to address")
			}
		}
		hash := block.Hash()
		height := block.Height
		switch data := delegate.(type) {
		case *engine.DelegateInfoList:
			continue
		case *engine.RegisterForgerData:
			if mapTxFee[tx.Tx.ID] >= consensus.RegisrerForgerFee {
				engine.DposVote.ProcessRegister(address.EncodeAddress(), data.Name, hash, height)
			}
		case *engine.VoteForgerData:
			if mapTxFee[tx.Tx.ID] >= consensus.VoteForgerFee {
				engine.DposVote.ProcessVote(address.EncodeAddress(), data.Forgers, hash, height)
			}
		case *engine.CancelVoteForgerData:
			if mapTxFee[tx.Tx.ID] >= consensus.CancelVoteForgerFee {
				engine.DposVote.ProcessCancelVote(address.EncodeAddress(), data.Forgers, hash, height)
			}
		}
	}
	return nil
}

func (c *Chain) CalculateBalance(block *types.Block, fIsAdd bool) map[bc.Hash]uint64 {

	addressBalances := []engine.AddressBalance{}
	mapTxFee := make(map[bc.Hash]uint64)
	var (
		address common.Address
		err     error
	)

	for _, tx := range block.Transactions {
		fee := uint64(0)
		for _, input := range tx.Inputs {
			fee += input.Amount()
			value := int64(input.Amount())
			address, err = common.NewAddressWitnessPubKeyHash(input.ControlProgram()[2:], &consensus.ActiveNetParams)
			if err != nil {
				address, err = common.NewAddressWitnessScriptHash(input.ControlProgram()[2:], &consensus.ActiveNetParams)
				if err != nil {
					continue
				}
			}
			if fIsAdd {
				value = 0 - value
			}
			addressBalances = append(addressBalances, engine.AddressBalance{address.EncodeAddress(), value})
		}
		for _, output := range tx.Outputs {
			fee -= output.Amount
			value := int64(output.Amount)
			address, err = common.NewAddressWitnessPubKeyHash(output.ControlProgram[2:], &consensus.ActiveNetParams)
			if err != nil {
				address, err = common.NewAddressWitnessScriptHash(output.ControlProgram[2:], &consensus.ActiveNetParams)
				if err != nil {
					continue
				}
			}
			if fIsAdd {
				value = 0 - value
			}
			addressBalances = append(addressBalances, engine.AddressBalance{address.EncodeAddress(), value})
		}
		mapTxFee[tx.Tx.ID] = fee
	}

	engine.DposVote.UpdateAddressBalance(addressBalances)
	return mapTxFee
}
