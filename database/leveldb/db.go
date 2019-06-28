package leveldb

import . "github.com/tendermint/tmlibs/common"

type DB interface {
	Get([]byte) []byte
	Set([]byte, []byte)
	SetSync([]byte, []byte)
	Delete([]byte)
	DeleteSync([]byte)
	Close()
	NewBatch() Batch
	Iterator() Iterator
	IteratorPrefix([]byte) Iterator
	// Iterate over a domain of keys in ascending order. End is exclusive.
	// Start must be less than end, or the Iterator is invalid.
	// A nil start is interpreted as an empty byteslice.
	// If end is nil, iterates up to the last item (inclusive).
	// CONTRACT: No writes may happen within a domain while an iterator exists over it.
	// CONTRACT: start, end readonly []byte
	IteratorRange(start, end []byte) Iterator

	// For debugging
	Print()
	Stats() map[string]string
}

type Batch interface {
	Set(key, value []byte)
	Delete(key []byte)
	Write()
}

type Iterator interface {
	Next() bool

	Key() []byte
	Value() []byte
	Seek([]byte) bool

	Release()
	Error() error
}

//-----------------------------------------------------------------------------

const (
	LevelDBBackendStr   = "leveldb" // legacy, defaults to goleveldb.
	CLevelDBBackendStr  = "cleveldb"
	GoLevelDBBackendStr = "goleveldb"
	MemDBBackendStr     = "memdb"
)

type dbCreator func(name string, dir string) (DB, error)

var backends = map[string]dbCreator{}

func registerDBCreator(backend string, creator dbCreator, force bool) {
	_, ok := backends[backend]
	if !force && ok {
		return
	}
	backends[backend] = creator
}

func NewDB(name string, backend string, dir string) DB {
	db, err := backends[backend](name, dir)
	if err != nil {
		PanicSanity(Fmt("Error initializing DB: %v", err))
	}
	return db
}
