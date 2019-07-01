package chainmgr

import (
	"encoding/binary"
	"sync"

	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc/types"
)

const (
	maxRamFastSync = 800 * 1024 * 1024 //100MB
)

var (
	errFindBlock = errors.New("can't find block from storage")
)

type Storage interface {
	WriteBlocks(peerID string, blocks []*types.Block) error
	ReadBlock(height uint64) (*blockStore, error)
}

type FileStore interface {
	writeBlock(block *types.Block) error
	readBlock(height uint64) (*types.Block, error)
}

type blockStore struct {
	block  *types.Block
	peerID string
}

type blockStorage struct {
	blockMsg *blockStore
	isRam    bool
}

type storage struct {
	actualUsage uint64
	blocks      map[uint64]blockStorage
	underlying  FileStore
	mux         sync.Mutex
}

func newStorage(db dbm.DB) *storage {
	return &storage{
		blocks:     make(map[uint64]blockStorage),
		underlying: newFileStore(db),
	}
}

func (s *storage) WriteBlocks(peerID string, blocks []*types.Block) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	for _, block := range blocks {
		binaryBlock, err := block.MarshalText()
		if err != nil {
			return errors.Wrap(err, "Marshal block header")
		}

		if uint64(len(binaryBlock))+s.actualUsage < maxRamFastSync {
			s.blocks[block.Height] = blockStorage{blockMsg: &blockStore{block: block, peerID: peerID}, isRam: true}
			s.actualUsage += uint64(len(binaryBlock))
			continue
		}

		if err := s.underlying.writeBlock(block); err != nil {
			return err
		}

		s.blocks[block.Height] = blockStorage{blockMsg: &blockStore{peerID: peerID}, isRam: false}
	}

	return nil
}

func (s *storage) ReadBlock(height uint64) (*blockStore, error) {
	s.mux.Lock()
	defer s.mux.Unlock()

	blockStore, ok := s.blocks[height]
	if !ok {
		return nil, errFindBlock
	}

	if blockStore.isRam {
		return blockStore.blockMsg, nil
	}

	block, err := s.underlying.readBlock(height)
	if err != nil {
		return nil, err
	}

	blockStore.blockMsg.block = block
	return blockStore.blockMsg, nil
}

type fileStore struct {
	db dbm.DB
}

func newFileStore(db dbm.DB) *fileStore {
	return &fileStore{
		db: db,
	}
}

func (fs *fileStore) writeBlock(block *types.Block) error {
	binaryBlock, err := block.MarshalText()
	if err != nil {
		return err
	}

	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, block.Height)
	fs.db.Set(key, binaryBlock)
	return nil
}

func (fs *fileStore) readBlock(height uint64) (*types.Block, error) {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, height)
	binaryBlock := fs.db.Get(key)
	if binaryBlock == nil {
		return nil, errors.New("can't find block from db")
	}

	block := &types.Block{}
	err := block.UnmarshalText(binaryBlock)
	return block, err
}
