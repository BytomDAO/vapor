package sqlite

import (
	"path"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/vapor/database/db"
)

func init() {
	dbCreator := func(name string, dir string) (db.SQLDB, error) {
		return NewSqliteDB(name, dir)
	}
	db.RegisterSqlDBCreator(db.SqliteDBBackendStr, dbCreator, false)
}

type SqliteDB struct {
	db *gorm.DB
}

func NewSqliteDB(name string, dir string) (*SqliteDB, error) {
	dbPath := path.Join(dir, name)
	cmn.EnsureDir(dbPath, 0700)
	dbFilePath := path.Join(dbPath, name+".db")
	db, err := gorm.Open("sqlite3", dbFilePath)
	if err != nil {
		return nil, err
	}
	database := &SqliteDB{db: db}
	return database, nil
}

func (s *SqliteDB) Name() string {
	return "sqlite3"
}

func (s *SqliteDB) Db() *gorm.DB {
	return s.db
}
