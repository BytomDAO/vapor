package database

import (
	"github.com/vapor/database/dbutils"
	dbm "github.com/vapor/database/leveldb"
)

// NewDB return new DB according to backend, defult is "leveldb"
func NewDB(name string, backend string, dir string) dbutils.DB {
	switch backend {
	default: // default is "leveldb"
		return dbm.NewDB(name, "leveldb", dir)
	}
}
