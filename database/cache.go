package database

import (
	"strconv"

	"github.com/golang/groupcache/singleflight"

	"github.com/bytom/vapor/common"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/protocol/state"
)

const (
	maxCachedBlockHeaders      = 4096
	maxCachedBlockTransactions = 1024
	maxCachedBlockHashes       = 8192
	maxCachedMainChainHashes   = 8192
	maxCachedConsensusResults  = 128
	maxCachedVoteBlockHash     = 8192
)

type fillBlockHeaderFn func(hash *bc.Hash) (*types.BlockHeader, error)
type fillBlockTransactionsFn func(hash *bc.Hash) ([]*types.Tx, error)
type fillBlockHashesFn func(height uint64) ([]*bc.Hash, error)
type fillMainChainHashFn func(height uint64) (*bc.Hash, error)
type fillConsensusResultFn func(seq uint64) (*state.ConsensusResult, error)
type fillPreRoundVoteBlockHashFn func(hash *bc.Hash) (*bc.Hash, error)

func newCache(fillBlockHeader fillBlockHeaderFn, fillBlockTxs fillBlockTransactionsFn, fillBlockHashes fillBlockHashesFn, fillMainChainHash fillMainChainHashFn, fillConsensusResult fillConsensusResultFn, fillPreRoundVoteBlockHash fillPreRoundVoteBlockHashFn) *cache {
	return &cache{
		lruBlockHeaders:            common.NewCache(maxCachedBlockHeaders),
		lruBlockTxs:                common.NewCache(maxCachedBlockTransactions),
		lruBlockHashes:             common.NewCache(maxCachedBlockHashes),
		lruMainChainHashes:         common.NewCache(maxCachedMainChainHashes),
		lruConsensusResults:        common.NewCache(maxCachedConsensusResults),
		lruPreRoundVoteBlockHashes: common.NewCache(maxCachedVoteBlockHash),

		fillBlockHeaderFn:           fillBlockHeader,
		fillBlockTransactionFn:      fillBlockTxs,
		fillBlockHashesFn:           fillBlockHashes,
		fillMainChainHashFn:         fillMainChainHash,
		fillConsensusResultFn:       fillConsensusResult,
		fillPreRoundVoteBlockHashFn: fillPreRoundVoteBlockHash,
	}
}

type cache struct {
	lruBlockHeaders            *common.Cache
	lruBlockTxs                *common.Cache
	lruBlockHashes             *common.Cache
	lruMainChainHashes         *common.Cache
	lruConsensusResults        *common.Cache
	lruPreRoundVoteBlockHashes *common.Cache

	fillBlockHeaderFn           func(hash *bc.Hash) (*types.BlockHeader, error)
	fillBlockTransactionFn      func(hash *bc.Hash) ([]*types.Tx, error)
	fillBlockHashesFn           func(uint64) ([]*bc.Hash, error)
	fillMainChainHashFn         func(uint64) (*bc.Hash, error)
	fillConsensusResultFn       func(seq uint64) (*state.ConsensusResult, error)
	fillPreRoundVoteBlockHashFn func(hash *bc.Hash) (*bc.Hash, error)

	sf singleflight.Group
}

func (c *cache) lookupPreRoundVoteBlockHash(header *types.BlockHeader, isVoteBlock func(height uint64) bool) (*bc.Hash, error) {
	hash := header.Hash()
	if isVoteBlock(header.Height) {
		c.lruPreRoundVoteBlockHashes.Add(hash, (&hash).Bytes())
		return &hash, nil
	}

	if data, ok := c.lruPreRoundVoteBlockHashes.Get(&header.PreviousBlockHash); ok {
		return data.(*bc.Hash), nil
	}

	blockHash, err := c.sf.Do("VoteBlock:"+(&hash).String(), func() (interface{}, error) {
		voteBlockHash, err := c.fillPreRoundVoteBlockHashFn(&hash)
		if err != nil {
			return nil, err
		}

		c.lruPreRoundVoteBlockHashes.Add(hash, voteBlockHash)
		return voteBlockHash, err
	})
	if err != nil {
		return nil, err
	}

	return blockHash.(*bc.Hash), nil
}

func (c *cache) lookupBlockHeader(hash *bc.Hash) (*types.BlockHeader, error) {
	if data, ok := c.lruBlockHeaders.Get(*hash); ok {
		return data.(*types.BlockHeader), nil
	}

	blockHeader, err := c.sf.Do("BlockHeader:"+hash.String(), func() (interface{}, error) {
		blockHeader, err := c.fillBlockHeaderFn(hash)
		if err != nil {
			return nil, err
		}

		c.lruBlockHeaders.Add(blockHeader.Hash(), blockHeader)
		return blockHeader, nil
	})
	if err != nil {
		return nil, err
	}
	return blockHeader.(*types.BlockHeader), nil
}

func (c *cache) lookupBlockTxs(hash *bc.Hash) ([]*types.Tx, error) {
	if data, ok := c.lruBlockTxs.Get(*hash); ok {
		return data.([]*types.Tx), nil
	}

	blockTxs, err := c.sf.Do("BlockTxs:"+hash.String(), func() (interface{}, error) {
		blockTxs, err := c.fillBlockTransactionFn(hash)
		if err != nil {
			return nil, err
		}

		c.lruBlockTxs.Add(*hash, blockTxs)
		return blockTxs, nil
	})
	if err != nil {
		return nil, err
	}
	return blockTxs.([]*types.Tx), nil
}

func (c *cache) lookupConsensusResult(seq uint64) (*state.ConsensusResult, error) {
	if data, ok := c.lruConsensusResults.Get(seq); ok {
		return data.(*state.ConsensusResult).Fork(), nil
	}

	seqStr := strconv.FormatUint(seq, 10)
	consensusResult, err := c.sf.Do("ConsensusResult:"+seqStr, func() (interface{}, error) {
		consensusResult, err := c.fillConsensusResultFn(seq)
		if err != nil {
			return nil, err
		}

		c.lruConsensusResults.Add(consensusResult.Seq, consensusResult)
		return consensusResult, nil
	})
	if err != nil {
		return nil, err
	}
	return consensusResult.(*state.ConsensusResult).Fork(), nil
}

func (c *cache) lookupMainChainHash(height uint64) (*bc.Hash, error) {
	if hash, ok := c.lruMainChainHashes.Get(height); ok {
		return hash.(*bc.Hash), nil
	}

	heightStr := strconv.FormatUint(height, 10)
	hash, err := c.sf.Do("BlockHashByHeight:"+heightStr, func() (interface{}, error) {
		hash, err := c.fillMainChainHashFn(height)
		if err != nil {
			return nil, err
		}

		c.lruMainChainHashes.Add(height, hash)
		return hash, nil
	})
	if err != nil {
		return nil, err
	}
	return hash.(*bc.Hash), nil
}

func (c *cache) lookupBlockHashesByHeight(height uint64) ([]*bc.Hash, error) {
	if hashes, ok := c.lruBlockHashes.Get(height); ok {
		return hashes.([]*bc.Hash), nil
	}

	heightStr := strconv.FormatUint(height, 10)
	hashes, err := c.sf.Do("BlockHashesByHeight:"+heightStr, func() (interface{}, error) {
		hashes, err := c.fillBlockHashesFn(height)
		if err != nil {
			return nil, err
		}

		c.lruBlockHashes.Add(height, hashes)
		return hashes, nil
	})
	if err != nil {
		return nil, err
	}
	return hashes.([]*bc.Hash), nil
}

func (c *cache) removeBlockHeader(blockHeader *types.BlockHeader) {
	c.lruBlockHeaders.Remove(blockHeader.Hash())
}

func (c *cache) removeBlockHashes(height uint64) {
	c.lruBlockHashes.Remove(height)
}

func (c *cache) removeMainChainHash(height uint64) {
	c.lruMainChainHashes.Remove(height)
}

func (c *cache) removeConsensusResult(consensusResult *state.ConsensusResult) {
	c.lruConsensusResults.Remove(consensusResult.Seq)
}

func (c *cache) removePreRoundVoteBlockHash(blockHash *bc.Hash) {
	c.lruPreRoundVoteBlockHashes.Remove(blockHash)
}
