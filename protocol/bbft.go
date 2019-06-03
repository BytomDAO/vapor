package protocol

import (
	"encoding/hex"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/config"
	"github.com/vapor/crypto/ed25519"
	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
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
)

func (c *Chain) isIrreversible(block *types.Block) bool {
	consensusNodes, err := c.consensusNodeManager.getConsensusNodesByVoteResult(&block.PreviousBlockHash)
	if err != nil {
		return false
	}

	signNum, err := c.validateSign(block)
	if err != nil {
		return false
	}

	return signNum > (uint64(len(consensusNodes)) * 2 / 3)
}

// NextLeaderTime returns the start time of the specified public key as the next leader node
func (c *Chain) IsBlocker(prevBlockHash *bc.Hash, pubkey string, timeStamp uint64) (bool, error) {
	return c.consensusNodeManager.isBlocker(prevBlockHash, pubkey, timeStamp)
}

func (c *Chain) ApplyBlock(voteResultMap map[uint64]*state.VoteResult, block *types.Block) (err error) {
	return c.consensusNodeManager.applyBlock(voteResultMap, block)
}

func (c *Chain) DetachBlock(voteResultMap map[uint64]*state.VoteResult, block *types.Block) error {
	return c.consensusNodeManager.detachBlock(voteResultMap, block)
}

// ProcessBlockSignature process the received block signature messages
// return whether a block become irreversible, if so, the chain module must update status
func (c *Chain) ProcessBlockSignature(signature []byte, xPub [64]byte, blockHeight uint64, blockHash *bc.Hash) error {
	block, err := c.consensusNodeManager.store.GetBlock(blockHash)
	if err != nil {
		// block is not exist, save the signature
		key := fmt.Sprintf("%s:%s", blockHash.String(), hex.EncodeToString(xPub[:]))
		c.signatureCache.Add(key, signature)
		return err
	}

	consensusNode, err := c.consensusNodeManager.getConsensusNode(&block.PreviousBlockHash, hex.EncodeToString(xPub[:]))
	if err != nil {
		return err
	}

	if chainkd.XPub(xPub).Verify(blockHash.Bytes(), signature) {
		return errInvalidSignature
	}

	isDoubleSign, err := c.checkDoubleSign(consensusNode.order, blockHeight, *blockHash)
	if err != nil {
		return err
	}

	if isDoubleSign {
		log.WithFields(log.Fields{"module": logModule, "blockHash": blockHash.String(), "xPub": hex.EncodeToString(xPub[:])}).Warn("the consensus node double sign the same height of different block")
		return errDoubleSignBlock
	}

	orphanBlock, ok := c.orphanManage.Get(blockHash)
	if ok {
		orphanBlock.Witness[consensusNode.order] = signature
		return nil
	}

	if err := c.updateBlockSignature(block, consensusNode.order, signature); err != nil {
		return err
	}

	if c.isIrreversible(block) && blockHeight > c.bestIrreversibleNode.Height {
		bestIrreversibleNode := c.index.GetNode(blockHash)
		if err := c.store.SaveChainNodeStatus(c.bestNode, bestIrreversibleNode); err != nil {
			return err
		}

		c.bestIrreversibleNode = bestIrreversibleNode
	}
	return nil
}

// ValidateBlock verify whether the block is valid
func (c *Chain) ValidateBlock(block *types.Block) error {
	signNum, err := c.validateSign(block)
	if err != nil {
		return err
	}

	if signNum == 0 {
		return errors.New("no valid signature")
	}
	return nil
}

// validateSign verify the signatures of block, and return the number of correct signature
// if some signature is invalid, they will be reset to nil
// if the block has not the signature of blocker, it will return error
func (c *Chain) validateSign(block *types.Block) (uint64, error) {
	var correctSignNum uint64
	consensusNodeMap, err := c.consensusNodeManager.getConsensusNodesByVoteResult(&block.PreviousBlockHash)
	if err != nil {
		return 0, err
	}

	hasBlockerSign := false
	for pubKey, node := range consensusNodeMap {
		if len(block.Witness) <= int(node.order) {
			continue
		}

		blockHash := block.Hash()
		if block.Witness[node.order] == nil {
			key := fmt.Sprintf("%s:%s", blockHash.String(), pubKey)
			signature, ok := c.signatureCache.Get(key)
			if ok {
				block.Witness[node.order] = signature.([]byte)
			}
		}

		pubKeyBytes, err := hex.DecodeString(pubKey)
		if err != nil {
			return 0, err
		}

		if ed25519.Verify(ed25519.PublicKey(pubKeyBytes[:32]), blockHash.Bytes(), block.Witness[node.order]) {
			isDoubleSign, err := c.checkDoubleSign(node.order, block.Height, block.Hash())
			if err != nil {
				return 0, err
			}

			if isDoubleSign {
				log.WithFields(log.Fields{"module": logModule, "blockHash": blockHash.String(), "pubKey": pubKey}).Warn("the consensus node double sign the same height of different block")
				// Consensus node is signed twice with the same block height, discard the signature
				block.Witness[node.order] = nil
			} else {
				correctSignNum++
				isBlocker, err := c.consensusNodeManager.isBlocker(&block.PreviousBlockHash, pubKey, block.Timestamp)
				if err != nil {
					return 0, err
				}
				if isBlocker {
					hasBlockerSign = true
				}
			}
		} else {
			// discard the invalid signature
			block.Witness[node.order] = nil
		}
	}
	if !hasBlockerSign {
		return 0, errors.New("the block has no signature of the blocker")
	}
	return correctSignNum, nil
}

func (c *Chain) checkDoubleSign(nodeOrder, blockHeight uint64, blockHash bc.Hash) (bool, error) {
	blockNodes := c.consensusNodeManager.blockIndex.NodesByHeight(blockHeight)
	for _, blockNode := range blockNodes {
		if blockNode.Hash == blockHash {
			continue
		}
		if ok, err := blockNode.BlockWitness.Test(uint32(nodeOrder)); err != nil && ok {
			block, err := c.consensusNodeManager.store.GetBlock(&blockHash)
			if err != nil {
				return false, err
			}

			// reset nil to discard signature
			if err := c.updateBlockSignature(block, nodeOrder, nil); err != nil {
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
		if ok, err := blockNode.BlockWitness.Test(uint32(node.order)); err != nil && ok {
			return nil, nil
		}
	}

	signature := block.Witness[node.order]
	if len(signature) == 0 {
		signature = xprv.Sign(block.Hash().Bytes())
		block.Witness[node.order] = signature
	}
	return signature, nil
}

func (c *Chain) updateBlockSignature(block *types.Block, nodeOrder uint64, signature []byte) error {
	blockHash := block.Hash()
	blockNode := c.consensusNodeManager.blockIndex.GetNode(&blockHash)

	if len(signature) != 0 {
		if err := blockNode.BlockWitness.Set(uint32(nodeOrder)); err != nil {
			return err
		}
	} else {
		if err := blockNode.BlockWitness.Clean(uint32(nodeOrder)); err != nil {
			return err
		}
	}

	block.Witness[nodeOrder] = signature
	txStatus, err := c.consensusNodeManager.store.GetTransactionStatus(&blockHash)
	if err != nil {
		return err
	}

	return c.consensusNodeManager.store.SaveBlock(block, txStatus)
}
