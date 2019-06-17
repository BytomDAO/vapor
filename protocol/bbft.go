package protocol

import (
	"encoding/hex"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/config"
	"github.com/vapor/errors"
	"github.com/vapor/event"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
)

const (
	maxSignatureCacheSize = 10000
)

var (
	errVotingOperationOverFlow = errors.New("voting operation result overflow")
	errDoubleSignBlock         = errors.New("the consensus is double sign in same height of different block")
	errInvalidSignature        = errors.New("the signature of block is invalid")
	errSignForkChain           = errors.New("can not sign fork before the irreversible block")
)

func signCacheKey(blockHash, pubkey string) string {
	return fmt.Sprintf("%s:%s", blockHash, pubkey)
}

func (c *Chain) isIrreversible(blockNode *state.BlockNode) bool {
	consensusNodes, err := c.consensusNodeManager.getConsensusNodes(blockNode.Parent)
	if err != nil {
		return false
	}

	signCount := 0
	for i := 0; i < len(consensusNodes); i++ {
		if ok, _ := blockNode.BlockWitness.Test(uint32(i)); ok {
			signCount++
		}
	}

	return signCount > len(consensusNodes)*2/3
}

// GetVoteResultByHash return vote result by block hash
func (c *Chain) GetVoteResultByHash(blockHash *bc.Hash) (*state.VoteResult, error) {
	blockNode := c.index.GetNode(blockHash)
	return c.consensusNodeManager.getVoteResult(state.CalcVoteSeq(blockNode.Height), blockNode)
}

// IsBlocker returns whether the consensus node is a blocker at the specified time
func (c *Chain) IsBlocker(prevBlockHash *bc.Hash, pubKey string, timeStamp uint64) (bool, error) {
	xPub, err := c.consensusNodeManager.getBlocker(prevBlockHash, timeStamp)
	if err != nil {
		return false, err
	}
	return xPub == pubKey, nil
}

// GetBlock return blocker by specified timestamp
func (c *Chain) GetBlocker(prevBlockHash *bc.Hash, timestamp uint64) (string, error) {
	return c.consensusNodeManager.getBlocker(prevBlockHash, timestamp)
}

// ProcessBlockSignature process the received block signature messages
// return whether a block become irreversible, if so, the chain module must update status
func (c *Chain) ProcessBlockSignature(signature, xPub []byte, blockHash *bc.Hash) error {
	xpubStr := hex.EncodeToString(xPub[:])
	blockNode := c.index.GetNode(blockHash)
	// save the signature if the block is not exist
	if blockNode == nil {
		cacheKey := signCacheKey(blockHash.String(), xpubStr)
		c.signatureCache.Add(cacheKey, signature)
		return nil
	}

	consensusNode, err := c.consensusNodeManager.getConsensusNode(blockNode.Parent, xpubStr)
	if err != nil {
		return err
	}

	if exist, _ := blockNode.BlockWitness.Test(uint32(consensusNode.Order)); exist {
		return nil
	}

	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	if err := c.checkNodeSign(blockNode.BlockHeader(), consensusNode, signature); err != nil {
		return err
	}

	if err := c.updateBlockSignature(blockNode, consensusNode.Order, signature); err != nil {
		return err
	}
	return c.eventDispatcher.Post(event.BlockSignatureEvent{BlockHash: *blockHash, Signature: signature, XPub: xPub})
}

// validateSign verify the signatures of block, and return the number of correct signature
// if some signature is invalid, they will be reset to nil
// if the block has not the signature of blocker, it will return error
func (c *Chain) validateSign(block *types.Block) error {
	consensusNodeMap, err := c.consensusNodeManager.getConsensusNodes(&block.PreviousBlockHash)
	if err != nil {
		return err
	}

	hasBlockerSign := false
	blockHash := block.Hash()
	for pubKey, node := range consensusNodeMap {
		if len(block.Witness) <= int(node.Order) {
			continue
		}

		if block.Get(node.Order) == nil {
			cachekey := signCacheKey(blockHash.String(), pubKey)
			if signature, ok := c.signatureCache.Get(cachekey); ok {
				block.Set(node.Order, signature.([]byte))
			} else {
				continue
			}
		}

		if err := c.checkNodeSign(&block.BlockHeader, node, block.Get(node.Order)); err == errDoubleSignBlock {
			log.WithFields(log.Fields{"module": logModule, "blockHash": blockHash.String(), "pubKey": pubKey}).Warn("the consensus node double sign the same height of different block")
			block.Delete(node.Order)
			continue
		} else if err != nil {
			return err
		}

		isBlocker, err := c.IsBlocker(&block.PreviousBlockHash, pubKey, block.Timestamp)
		if err != nil {
			return err
		}

		if isBlocker {
			hasBlockerSign = true
		}

	}

	if !hasBlockerSign {
		return errors.New("the block has no signature of the blocker")
	}
	return nil
}

func (c *Chain) checkNodeSign(bh *types.BlockHeader, consensusNode *state.ConsensusNode, signature []byte) error {
	if !consensusNode.XPub.Verify(bh.Hash().Bytes(), signature) {
		return errInvalidSignature
	}

	blockNodes := c.consensusNodeManager.blockIndex.NodesByHeight(bh.Height)
	for _, blockNode := range blockNodes {
		if blockNode.Hash == bh.Hash() {
			continue
		}

		consensusNode, err := c.consensusNodeManager.getConsensusNode(blockNode.Parent, consensusNode.XPub.String())
		if err != nil && err != errNotFoundConsensusNode {
			return err
		}

		if err == errNotFoundConsensusNode {
			continue
		}

		if ok, err := blockNode.BlockWitness.Test(uint32(consensusNode.Order)); err == nil && ok {
			return errDoubleSignBlock
		}
	}
	return nil
}

// SignBlock signing the block if current node is consensus node
func (c *Chain) SignBlock(block *types.Block) ([]byte, error) {
	xprv := config.CommonConfig.PrivateKey()
	xpubStr := xprv.XPub().String()
	node, err := c.consensusNodeManager.getConsensusNode(&block.PreviousBlockHash, xpubStr)
	if err == errNotFoundConsensusNode {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	//check double sign in same block height
	blockNodes := c.consensusNodeManager.blockIndex.NodesByHeight(block.Height)
	for _, blockNode := range blockNodes {
		// Has already signed the same height block
		if ok, err := blockNode.BlockWitness.Test(uint32(node.Order)); err == nil && ok {
			return nil, nil
		}
	}

	for blockNode := c.index.GetNode(&block.PreviousBlockHash); !c.index.InMainchain(&blockNode.Hash); {
		if blockNode.Height <= c.bestIrreversibleNode.Height {
			return nil, errSignForkChain
		}
		blockNode = c.index.GetNode(blockNode.Parent)
	}

	signature := block.Get(node.Order)
	if len(signature) == 0 {
		signature = xprv.Sign(block.Hash().Bytes())
		block.Set(node.Order, signature)
	}
	return signature, nil
}

func (c *Chain) updateBlockSignature(blockNode *state.BlockNode, nodeOrder uint64, signature []byte) error {
	if err := blockNode.BlockWitness.Set(uint32(nodeOrder)); err != nil {
		return err
	}

	blockHeader, err := c.store.GetBlockHeader(&blockNode.Hash)
	if err != nil {
		return err
	}

	blockHeader.Set(nodeOrder, signature)

	if err := c.store.SaveBlockHeader(blockHeader); err != nil {
		return err
	}

	if c.isIrreversible(blockNode) && blockNode.Height > c.bestIrreversibleNode.Height {
		if err := c.store.SaveChainStatus(c.bestNode, blockNode, state.NewUtxoViewpoint(), []*state.VoteResult{}); err != nil {
			return err
		}

		c.bestIrreversibleNode = blockNode
	}
	return nil
}
