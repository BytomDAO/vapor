package db

import (
	"github.com/jinzhu/gorm"
	. "github.com/tendermint/tmlibs/common"
)

type SQLDB interface {
	Name() string
	Db() *gorm.DB
}

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
	SqliteDBBackendStr  = "sqlitedb"
)

type dbCreator func(name string, dir string) (DB, error)

var backends = map[string]dbCreator{}

func RegisterDBCreator(backend string, creator dbCreator, force bool) {
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

type sqlDbCreator func(name string, dir string) (SQLDB, error)

var sqlBackends = map[string]sqlDbCreator{}

func RegisterSqlDBCreator(backend string, creator sqlDbCreator, force bool) {
	_, ok := sqlBackends[backend]
	if !force && ok {
		return
	}
	sqlBackends[backend] = creator
}

func NewSqlDB(name string, backend string, dir string) SQLDB {
	db, err := sqlBackends[backend](name, dir)
	if err != nil {
		PanicSanity(Fmt("Error initializing DB: %v", err))
	}
	return db
}
