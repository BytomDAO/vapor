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
	errStorageFindBlock = errors.New("can't find block from storage")
	errDBFindBlock      = errors.New("can't find block from DB")
)

type Storage interface {
	ResetParameter()
	WriteBlocks(peerID string, blocks []*types.Block) error
	ReadBlock(height uint64) (*blockStore, error)
}

type LocalStore interface {
	writeBlock(block *types.Block) error
	readBlock(height uint64) (*types.Block, error)
	clearData()
}

type blockStore struct {
	block  *types.Block
	peerID string
	isRam  bool
}

type storage struct {
	actualUsage int
	blocks      map[uint64]*blockStore
	localStore  LocalStore
	mux         sync.RWMutex
}

func newStorage(db dbm.DB) *storage {
	return &storage{
		blocks:     make(map[uint64]*blockStore),
		localStore: newDBStore(db),
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

		if len(binaryBlock)+s.actualUsage < maxRamFastSync {
			s.blocks[block.Height] = &blockStore{block: block, peerID: peerID, isRam: true}
			s.actualUsage += len(binaryBlock)
			continue
		}

		if err := s.localStore.writeBlock(block); err != nil {
			return err
		}

		s.blocks[block.Height] = &blockStore{peerID: peerID, isRam: false}
	}

	return nil
}

func (s *storage) ReadBlock(height uint64) (*blockStore, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	blockStore, ok := s.blocks[height]
	if !ok {
		return nil, errStorageFindBlock
	}

	if blockStore.isRam {
		return blockStore, nil
	}

	block, err := s.localStore.readBlock(height)
	if err != nil {
		return nil, err
	}

	blockStore.block = block
	return blockStore, nil
}

func (s *storage) ResetParameter() {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.blocks = make(map[uint64]*blockStore)
	s.actualUsage = 0
	s.localStore.clearData()
}

type levelDBStore struct {
	db dbm.DB
}

func newDBStore(db dbm.DB) *levelDBStore {
	return &levelDBStore{
		db: db,
	}
}

func (fs *levelDBStore) clearData() {
	iter := fs.db.Iterator()
	defer iter.Release()

	for iter.Next() {
		fs.db.Delete(iter.Key())
	}
}

func (fs *levelDBStore) writeBlock(block *types.Block) error {
	binaryBlock, err := block.MarshalText()
	if err != nil {
		return err
	}

	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, block.Height)
	fs.db.Set(key, binaryBlock)
	return nil
}

func (fs *levelDBStore) readBlock(height uint64) (*types.Block, error) {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, height)
	binaryBlock := fs.db.Get(key)
	if binaryBlock == nil {
		return nil, errDBFindBlock
	}

	block := &types.Block{}
	return block, block.UnmarshalText(binaryBlock)
}
