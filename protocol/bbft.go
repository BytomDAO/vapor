package protocol

import (
	"encoding/hex"
	"fmt"

	"github.com/golang/groupcache/lru"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/config"
	"github.com/vapor/crypto/ed25519"
	"github.com/vapor/crypto/ed25519/chainkd"
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
)

type bbft struct {
	consensusNodeManager *consensusNodeManager
	orphanManage         *OrphanManage
	signatureCache       *lru.Cache
	eventDispatcher      *event.Dispatcher
}

func newBbft(store Store, blockIndex *state.BlockIndex, orphanManage *OrphanManage, eventDispatcher *event.Dispatcher) *bbft {
	return &bbft{
		orphanManage:         orphanManage,
		consensusNodeManager: newConsensusNodeManager(store, blockIndex),
		signatureCache:       lru.New(maxSignatureCacheSize),
		eventDispatcher:      eventDispatcher,
	}
}

func (b *bbft) isIrreversible(block *types.Block) bool {
	consensusNodes, err := b.consensusNodeManager.getConsensusNodesByVoteResult(&block.PreviousBlockHash)
	if err != nil {
		return false
	}

	signNum, err := b.validateSign(block)
	if err != nil {
		return false
	}

	return signNum > (uint64(len(consensusNodes)) * 2 / 3)
}

// NextLeaderTime returns the start time of the specified public key as the next leader node
func (b *bbft) NextLeaderTimeRange(pubkey []byte, prevBlockHash *bc.Hash) (uint64, uint64, error) {
	return b.consensusNodeManager.nextLeaderTimeRange(pubkey, prevBlockHash)
}

func (b *bbft) ApplyBlock(voteResultMap map[uint64]*state.VoteResult, block *types.Block) (err error) {
	return b.consensusNodeManager.applyBlock(voteResultMap, block)
}

func (b *bbft) DetachBlock(voteResultMap map[uint64]*state.VoteResult, block *types.Block) error {
	return b.consensusNodeManager.detachBlock(voteResultMap, block)
}

// ProcessBlockSignature process the received block signature messages
// return whether a block become irreversible, if so, the chain module must update status
func (b *bbft) ProcessBlockSignature(signature []byte, xPub [64]byte, blockHeight uint64, blockHash *bc.Hash) (bool, error) {
	block, err := b.consensusNodeManager.store.GetBlock(blockHash)
	if err != nil {
		// block is not exist, save the signature
		key := fmt.Sprintf("%s:%s", blockHash.String(), hex.EncodeToString(xPub[:]))
		b.signatureCache.Add(key, signature)
		return false, err
	}

	consensusNode, err := b.consensusNodeManager.getConsensusNode(&block.PreviousBlockHash, hex.EncodeToString(xPub[:]))
	if err != nil {
		return false, err
	}

	if chainkd.XPub(xPub).Verify(blockHash.Bytes(), signature) {
		return false, errInvalidSignature
	}

	isDoubleSign, err := b.checkDoubleSign(consensusNode.order, blockHeight, *blockHash)
	if err != nil {
		return false, err
	}

	if isDoubleSign {
		log.WithFields(log.Fields{"module": logModule, "blockHash": blockHash.String(), "xPub": hex.EncodeToString(xPub[:])}).Warn("the consensus node double sign the same height of different block")
		return false, errDoubleSignBlock
	}

	orphanBlock, ok := b.orphanManage.Get(blockHash)
	if ok {
		orphanBlock.Witness[consensusNode.order] = signature
		return false, nil
	}

	if err := b.updateBlockSignature(block, consensusNode.order, signature); err != nil {
		return false, err
	}

	return b.isIrreversible(block), nil
}

// ValidateBlock verify whether the block is valid
func (b *bbft) ValidateBlock(block *types.Block) error {
	signNum, err := b.validateSign(block)
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
func (b *bbft) validateSign(block *types.Block) (uint64, error) {
	var correctSignNum uint64
	consensusNodeMap, err := b.consensusNodeManager.getConsensusNodesByVoteResult(&block.PreviousBlockHash)
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
			signature, ok := b.signatureCache.Get(key)
			if ok {
				block.Witness[node.order] = signature.([]byte)
			}
		}

		pubKeyBytes, err := hex.DecodeString(pubKey)
		if err != nil {
			return 0, err
		}

		if ed25519.Verify(ed25519.PublicKey(pubKeyBytes[:32]), blockHash.Bytes(), block.Witness[node.order]) {
			isDoubleSign, err := b.checkDoubleSign(node.order, block.Height, block.Hash())
			if err != nil {
				return 0, err
			}

			if isDoubleSign {
				log.WithFields(log.Fields{"module": logModule, "blockHash": blockHash.String(), "pubKey": pubKey}).Warn("the consensus node double sign the same height of different block")
				// Consensus node is signed twice with the same block height, discard the signature
				block.Witness[node.order] = nil
			} else {
				correctSignNum++
				isBlocker, err := b.consensusNodeManager.isBlocker(block, pubKey)
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

func (b *bbft) checkDoubleSign(nodeOrder, blockHeight uint64, blockHash bc.Hash) (bool, error) {
	blockNodes := b.consensusNodeManager.blockIndex.NodesByHeight(blockHeight)
	for _, blockNode := range blockNodes {
		if blockNode.Hash == blockHash {
			continue
		}
		if ok, err := blockNode.BlockWitness.Test(uint32(nodeOrder)); err != nil && ok {
			block, err := b.consensusNodeManager.store.GetBlock(&blockHash)
			if err != nil {
				return false, err
			}

			// reset nil to discard signature
			if err := b.updateBlockSignature(block, nodeOrder, nil); err != nil {
				return false, err
			}

			return true, nil
		}
	}
	return false, nil
}

// SignBlock signing the block if current node is consensus node
func (b *bbft) SignBlock(block *types.Block) ([]byte, error) {
	xprv := config.CommonConfig.PrivateKey()
	xpub := [64]byte(xprv.XPub())
	node, err := b.consensusNodeManager.getConsensusNode(&block.PreviousBlockHash, hex.EncodeToString(xpub[:]))
	if err != nil && err != errNotFoundConsensusNode {
		return nil, err
	}

	if node == nil {
		return nil, nil
	}

	signature := block.Witness[node.order]
	if len(signature) == 0 {
		signature = xprv.Sign(block.Hash().Bytes())
		block.Witness[node.order] = signature
	}
	return signature, nil
}

func (b *bbft) updateBlockSignature(block *types.Block, nodeOrder uint64, signature []byte) error {
	blockHash := block.Hash()
	blockNode := b.consensusNodeManager.blockIndex.GetNode(&blockHash)

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
	txStatus, err := b.consensusNodeManager.store.GetTransactionStatus(&blockHash)
	if err != nil {
		return err
	}

	return b.consensusNodeManager.store.SaveBlock(block, txStatus)
}

// SetBlockIndex set the block index field
func (b *bbft) SetBlockIndex(blockIndex *state.BlockIndex) {
	b.consensusNodeManager.blockIndex = blockIndex
}
