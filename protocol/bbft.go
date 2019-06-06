package protocol

import (
	"encoding/hex"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/config"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/state"
	"github.com/vapor/event"
)

const (
	maxSignatureCacheSize = 10000
)

var (
	errVotingOperationOverFlow = errors.New("voting operation result overflow")
	errDoubleSignBlock         = errors.New("the consensus is double sign in same height of different block")
	errInvalidSignature        = errors.New("the signature of block is invalid")
)

func signCacheKey(blockHash, pubkey string) string {
	return fmt.Sprintf("%s:%s", blockHash, pubkey)
}

func (c *Chain) isIrreversible(blockNode *state.BlockNode) bool {
	consensusNodes, err := c.consensusNodeManager.getConsensusNodes(&blockNode.Parent.Hash)
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

// NextLeaderTime returns the start time of the specified public key as the next leader node
func (c *Chain) IsBlocker(prevBlockHash *bc.Hash, pubkey string, timeStamp uint64) (bool, error) {
	return c.consensusNodeManager.isBlocker(prevBlockHash, pubkey, timeStamp)
}

// ProcessBlockSignature process the received block signature messages
// return whether a block become irreversible, if so, the chain module must update status
func (c *Chain) ProcessBlockSignature(signature []byte, xPub [64]byte, blockHash *bc.Hash) error {
	xpubStr := hex.EncodeToString(xPub[:])
	blockNode := c.index.GetNode(blockHash)
	// save the signature if the block is not exist
	if blockNode == nil {
		cacheKey := signCacheKey(blockHash.String(), xpubStr)
		c.signatureCache.Add(cacheKey, signature)
		return nil
	}

	consensusNode, err := c.consensusNodeManager.getConsensusNode(&blockNode.Parent.Hash, xpubStr)
	if err != nil {
		return err
	}

	if exist, err := blockNode.BlockWitness.Test(uint32(consensusNode.Order)); err != nil && exist {
		return nil
	}

	if !consensusNode.XPub.Verify(blockHash.Bytes(), signature) {
		return errInvalidSignature
	}

	isDoubleSign, err := c.checkDoubleSign(consensusNode.Order, blockNode.Height, *blockHash)
	if err != nil {
		return err
	}

	if isDoubleSign {
		return errDoubleSignBlock
	}

	if err := c.updateBlockSignature(&blockNode.Hash, consensusNode.Order, signature); err != nil {
		return err
	}

	if c.isIrreversible(blockNode) && blockNode.Height > c.bestIrreversibleNode.Height {
		bestIrreversibleNode := c.index.GetNode(blockHash)
		if err := c.store.SaveChainNodeStatus(c.bestNode, bestIrreversibleNode); err != nil {
			return err
		}

		c.bestIrreversibleNode = bestIrreversibleNode
	}

	return c.eventDispatcher.Post(event.BlockSignatureEvent{BlockHash: *blockHash, Signature: signature, XPub: xPub})
}

// validateSign verify the signatures of block, and return the number of correct signature
// if some signature is invalid, they will be reset to nil
// if the block has not the signature of blocker, it will return error
func (c *Chain) validateSign(block *types.Block) (uint64, error) {
	consensusNodeMap, err := c.consensusNodeManager.getConsensusNodes(&block.PreviousBlockHash)
	if err != nil {
		return 0, err
	}

	hasBlockerSign := false
	signCount := uint64(0)
	blockHash := block.Hash()
	for pubKey, node := range consensusNodeMap {
		if len(block.Witness) <= int(node.Order) {
			continue
		}

		if block.Witness[node.Order] == nil {
			cachekey := signCacheKey(blockHash.String(), pubKey)
			if signature, ok := c.signatureCache.Get(cachekey); ok {
				block.Witness[node.Order] = signature.([]byte)
			} else {
				continue
			}
		}

		if ok := node.XPub.Verify(blockHash.Bytes(), block.Witness[node.Order]); !ok {
			block.Witness[node.Order] = nil
			continue
		}

		isDoubleSign, err := c.checkDoubleSign(node.Order, block.Height, block.Hash())
		if err != nil {
			return 0, err
		}

		if isDoubleSign {
			// Consensus node is signed twice with the same block height, discard the signature
			log.WithFields(log.Fields{"module": logModule, "blockHash": blockHash.String(), "pubKey": pubKey}).Warn("the consensus node double sign the same height of different block")
			block.Witness[node.Order] = nil
			continue
		}

		signCount++
		isBlocker, err := c.consensusNodeManager.isBlocker(&block.PreviousBlockHash, pubKey, block.Timestamp)
		if err != nil {
			return 0, err
		}

		if isBlocker {
			hasBlockerSign = true
		}

	}

	if !hasBlockerSign {
		return 0, errors.New("the block has no signature of the blocker")
	}
	return signCount, nil
}

func (c *Chain) checkDoubleSign(nodeOrder, blockHeight uint64, blockHash bc.Hash) (bool, error) {
	blockNodes := c.consensusNodeManager.blockIndex.NodesByHeight(blockHeight)
	for _, blockNode := range blockNodes {
		if blockNode.Hash == blockHash {
			continue
		}
		if ok, err := blockNode.BlockWitness.Test(uint32(nodeOrder)); err != nil && ok {
			if err := c.updateBlockSignature(&blockHash, nodeOrder, nil); err != nil {
				return false, err
			}

			return true, nil
		}
	}
	return false, nil
}

// SignBlock signing the block if current node is consensus node
func (c *Chain) SignBlock(block *types.Block) ([]byte, error) {
	xprv := config.CommonConfig.PrivateKey()
	xpub := [64]byte(xprv.XPub())
	node, err := c.consensusNodeManager.getConsensusNode(&block.PreviousBlockHash, hex.EncodeToString(xpub[:]))
	if err != nil && err != errNotFoundConsensusNode {
		return nil, err
	}

	if node == nil {
		return nil, nil
	}

	blockNodes := c.consensusNodeManager.blockIndex.NodesByHeight(block.Height)
	for _, blockNode := range blockNodes {
		// Has already signed the same height block
		if ok, err := blockNode.BlockWitness.Test(uint32(node.Order)); err != nil && ok {
			return nil, nil
		}
	}

	signature := block.Witness[node.Order]
	if len(signature) == 0 {
		signature = xprv.Sign(block.Hash().Bytes())
		block.Witness[node.Order] = signature
	}
	return signature, nil
}

func (c *Chain) updateBlockSignature(blockHash *bc.Hash, nodeOrder uint64, signature []byte) error {
	blockNode := c.consensusNodeManager.blockIndex.GetNode(blockHash)
	if len(signature) != 0 {
		if err := blockNode.BlockWitness.Set(uint32(nodeOrder)); err != nil {
			return err
		}
	} else {
		if err := blockNode.BlockWitness.Clean(uint32(nodeOrder)); err != nil {
			return err
		}
	}

	block, err := c.store.GetBlock(blockHash)
	if err != nil {
		return err
	}

	block.Witness[nodeOrder] = signature
	txStatus, err := c.consensusNodeManager.store.GetTransactionStatus(blockHash)
	if err != nil {
		return err
	}

	return c.consensusNodeManager.store.SaveBlock(block, txStatus)
}
