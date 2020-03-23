package protocol

import (
	"encoding/hex"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/vapor/config"
	"github.com/bytom/vapor/crypto/ed25519/chainkd"
	"github.com/bytom/vapor/errors"
	"github.com/bytom/vapor/event"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
)

const (
	maxSignatureCacheSize = 10000
)

var (
	errDoubleSignBlock  = errors.New("the consensus is double sign in same height of different block")
	errInvalidSignature = errors.New("the signature of block is invalid")
	errSignForkChain    = errors.New("can not sign fork before the irreversible block")
)

func signCacheKey(blockHash, pubkey string) string {
	return fmt.Sprintf("%s:%s", blockHash, pubkey)
}

func (c *Chain) checkDoubleSign(bh *types.BlockHeader, xPub string) error {
	blockHashes, err := c.store.GetBlockHashesByHeight(bh.Height)
	if err != nil {
		return err
	}

	for _, blockHash := range blockHashes {
		if *blockHash == bh.Hash() {
			continue
		}

		blockHeader, err := c.store.GetBlockHeader(blockHash)
		if err != nil {
			return err
		}

		consensusNode, err := c.getConsensusNode(&blockHeader.PreviousBlockHash, xPub)
		if err == errNotFoundConsensusNode {
			continue
		} else if err != nil {
			return err
		}

		if blockHeader.BlockWitness.Get(consensusNode.Order) != nil {
			return errDoubleSignBlock
		}
	}
	return nil
}

func (c *Chain) checkNodeSign(bh *types.BlockHeader, consensusNode *state.ConsensusNode, signature []byte) error {
	if !consensusNode.XPub.Verify(bh.Hash().Bytes(), signature) {
		return errInvalidSignature
	}

	return c.checkDoubleSign(bh, consensusNode.XPub.String())
}

func (c *Chain) isIrreversible(blockHeader *types.BlockHeader) bool {
	consensusNodes, err := c.getConsensusNodes(&blockHeader.PreviousBlockHash)
	if err != nil {
		return false
	}

	signCount := 0
	for i := 0; i < len(consensusNodes); i++ {
		if blockHeader.BlockWitness.Get(uint64(i)) != nil {
			signCount++
		}
	}

	return signCount > len(consensusNodes)*2/3
}

func (c *Chain) updateBlockSignature(blockHeader *types.BlockHeader, nodeOrder uint64, signature []byte) error {
	blockHeader.Set(nodeOrder, signature)
	if err := c.store.SaveBlockHeader(blockHeader); err != nil {
		return err
	}

	if !c.isIrreversible(blockHeader) || blockHeader.Height <= c.lastIrrBlockHeader.Height {
		return nil
	}

	if c.InMainChain(blockHeader.Hash()) {
		if err := c.store.SaveChainStatus(c.bestBlockHeader, blockHeader, []*types.BlockHeader{}, state.NewUtxoViewpoint(), []*state.ConsensusResult{}); err != nil {
			return err
		}

		c.lastIrrBlockHeader = blockHeader
	} else {
		// block is on a forked chain
		log.WithFields(log.Fields{"module": logModule}).Info("majority votes received on forked chain")
		tail, err := c.traceLongestChainTail(blockHeader)
		if err != nil {
			return err
		}

		return c.reorganizeChain(tail)
	}
	return nil
}

// validateSign verify the signatures of block, and return the number of correct signature
// if some signature is invalid, they will be reset to nil
// if the block does not have the signature of blocker, it will return error
func (c *Chain) validateSign(block *types.Block) error {
	consensusNodeMap, err := c.getConsensusNodes(&block.PreviousBlockHash)
	if err != nil {
		return err
	}

	blocker, err := c.GetBlocker(&block.PreviousBlockHash, block.Timestamp)
	if err != nil {
		return err
	}

	hasBlockerSign := false
	blockHash := block.Hash()
	for pubKey, node := range consensusNodeMap {
		if block.BlockWitness.Get(node.Order) == nil {
			cachekey := signCacheKey(blockHash.String(), pubKey)
			if signature, ok := c.signatureCache.Get(cachekey); ok {
				block.Set(node.Order, signature.([]byte))
				c.eventDispatcher.Post(event.BlockSignatureEvent{BlockHash: blockHash, Signature: signature.([]byte), XPub: node.XPub[:]})
				c.signatureCache.Remove(cachekey)
			} else {
				continue
			}
		}

		if err := c.checkNodeSign(&block.BlockHeader, node, block.Get(node.Order)); err == errDoubleSignBlock {
			log.WithFields(log.Fields{"module": logModule, "blockHash": blockHash.String(), "pubKey": pubKey}).Warn("the consensus node double sign the same height of different block")
			// if the blocker double sign & become the mainchain, that means
			// all the side chain will reject the main chain make the chain
			// fork. All the node will ban each other & can't roll back
			if blocker != pubKey {
				block.BlockWitness.Delete(node.Order)
				continue
			}
		} else if err != nil {
			return err
		}

		if blocker == pubKey {
			hasBlockerSign = true
		}
	}

	if !hasBlockerSign {
		return errors.New("the block has no signature of the blocker")
	}
	return nil
}

// ProcessBlockSignature process the received block signature messages
// return whether a block become irreversible, if so, the chain module must update status
func (c *Chain) ProcessBlockSignature(signature, xPub []byte, blockHash *bc.Hash) error {
	xpubStr := hex.EncodeToString(xPub[:])
	blockHeader, _ := c.store.GetBlockHeader(blockHash)

	// save the signature if the block is not exist
	if blockHeader == nil {
		var xPubKey chainkd.XPub
		copy(xPubKey[:], xPub[:])
		if !xPubKey.Verify(blockHash.Bytes(), signature) {
			return errInvalidSignature
		}

		cacheKey := signCacheKey(blockHash.String(), xpubStr)
		c.signatureCache.Add(cacheKey, signature)
		return nil
	}

	consensusNode, err := c.getConsensusNode(&blockHeader.PreviousBlockHash, xpubStr)
	if err != nil {
		return err
	}

	if blockHeader.BlockWitness.Get(consensusNode.Order) != nil {
		return nil
	}

	c.cond.L.Lock()
	defer c.cond.L.Unlock()
	if err := c.checkNodeSign(blockHeader, consensusNode, signature); err != nil {
		return err
	}

	if err := c.updateBlockSignature(blockHeader, consensusNode.Order, signature); err != nil {
		return err
	}
	return c.eventDispatcher.Post(event.BlockSignatureEvent{BlockHash: *blockHash, Signature: signature, XPub: xPub})
}

// SignBlockHeader signing the block if current node is consensus node
func (c *Chain) SignBlockHeader(blockHeader *types.BlockHeader) error {
	_, err := c.signBlockHeader(blockHeader)
	return err
}

func (c *Chain) applyBlockSign(blockHeader *types.BlockHeader) error {
	signature, err := c.signBlockHeader(blockHeader)
	if err != nil {
		return err
	}

	if len(signature) == 0 {
		return nil
	}

	if err := c.store.SaveBlockHeader(blockHeader); err != nil {
		return err
	}

	xpub := config.CommonConfig.PrivateKey().XPub()
	return c.eventDispatcher.Post(event.BlockSignatureEvent{BlockHash: blockHeader.Hash(), Signature: signature, XPub: xpub[:]})
}

func (c *Chain) signBlockHeader(blockHeader *types.BlockHeader) ([]byte, error) {
	xprv := config.CommonConfig.PrivateKey()
	xpub := xprv.XPub()
	node, err := c.getConsensusNode(&blockHeader.PreviousBlockHash, xpub.String())
	blockHash := blockHeader.Hash()
	if err == errNotFoundConsensusNode {
		log.WithFields(log.Fields{"module": logModule, "blockHash": blockHash.String()}).Debug("can't find consensus node of current node")
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	if len(blockHeader.Get(node.Order)) != 0 {
		return nil, nil
	}

	if err := c.checkDoubleSign(blockHeader, node.XPub.String()); err == errDoubleSignBlock {
		log.WithFields(log.Fields{"module": logModule, "blockHash": blockHash.String()}).Warn("current node has double sign the block")
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	signature := xprv.Sign(blockHeader.Hash().Bytes())
	blockHeader.Set(node.Order, signature)
	return signature, nil
}
