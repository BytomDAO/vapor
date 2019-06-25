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

// GetVoteResultByHash return vote result by block hash
func (c *Chain) GetVoteResultByHash(blockHash *bc.Hash) (*state.VoteResult, error) {
	blockHeader, err := c.store.GetBlockHeader(blockHash)
	if err != nil {
		return nil, err
	}
	return c.getVoteResult(state.CalcVoteSeq(blockHeader.Height), blockHeader)
}

// IsBlocker returns whether the consensus node is a blocker at the specified time
func (c *Chain) IsBlocker(prevBlockHash *bc.Hash, pubKey string, timeStamp uint64) (bool, error) {
	xPub, err := c.GetBlocker(prevBlockHash, timeStamp)
	if err != nil {
		return false, err
	}
	return xPub == pubKey, nil
}

// ProcessBlockSignature process the received block signature messages
// return whether a block become irreversible, if so, the chain module must update status
func (c *Chain) ProcessBlockSignature(signature, xPub []byte, blockHash *bc.Hash) error {
	xpubStr := hex.EncodeToString(xPub[:])
	blockHeader, _ := c.store.GetBlockHeader(blockHash)

	// save the signature if the block is not exist
	if blockHeader == nil {
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

// validateSign verify the signatures of block, and return the number of correct signature
// if some signature is invalid, they will be reset to nil
// if the block has not the signature of blocker, it will return error
func (c *Chain) validateSign(block *types.Block) error {
	consensusNodeMap, err := c.getConsensusNodes(&block.PreviousBlockHash)
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

		consensusNode, err := c.getConsensusNode(&blockHeader.PreviousBlockHash, consensusNode.XPub.String())
		if err != nil && err != errNotFoundConsensusNode {
			return err
		}

		if err == errNotFoundConsensusNode {
			continue
		}

		if blockHeader.BlockWitness.Get(consensusNode.Order) != nil {
			return errDoubleSignBlock
		}
	}
	return nil
}

// SignBlock signing the block if current node is consensus node
func (c *Chain) SignBlock(block *types.Block) ([]byte, error) {
	xprv := config.CommonConfig.PrivateKey()
	xpubStr := xprv.XPub().String()
	node, err := c.getConsensusNode(&block.PreviousBlockHash, xpubStr)
	if err == errNotFoundConsensusNode {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	//check double sign in same block height
	blockHashes, err := c.store.GetBlockHashesByHeight(block.Height)
	if err != nil {
		return nil, err
	}

	for _, hash := range blockHashes {
		blockHeader, err := c.store.GetBlockHeader(hash)
		if err != nil {
			return nil, err
		}

		// Has already signed the same height block
		if blockHeader.BlockWitness.Get(node.Order) != nil {
			return nil, nil
		}
	}

	signature := block.Get(node.Order)
	if len(signature) == 0 {
		signature = xprv.Sign(block.Hash().Bytes())
		block.Set(node.Order, signature)
	}
	return signature, nil
}

func (c *Chain) updateBlockSignature(blockHeader *types.BlockHeader, nodeOrder uint64, signature []byte) error {
	blockHeader.Set(nodeOrder, signature)
	if err := c.store.SaveBlockHeader(blockHeader); err != nil {
		return err
	}

	if c.isIrreversible(blockHeader) && blockHeader.Height > c.bestIrrBlockHeader.Height {
		if err := c.store.SaveChainStatus(c.bestBlockHeader, blockHeader, []*types.BlockHeader{}, state.NewUtxoViewpoint(), []*state.VoteResult{}); err != nil {
			return err
		}
		c.bestIrrBlockHeader = blockHeader
	}
	return nil
}
