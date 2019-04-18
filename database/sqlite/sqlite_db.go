package sqlite

import (
	"path"

	"github.com/go-xorm/xorm"
	_ "github.com/mattn/go-sqlite3"
	cmn "github.com/tendermint/tmlibs/common"

	"github.com/vapor/database/db"
)

func init() {
	dbCreator := func(name string, dir string) (db.DB, error) {
		return NewSqliteDB(name, dir)
	}
	db.RegisterDBCreator(db.SqliteDBBackendStr, dbCreator, false)
}

type SqliteDB struct {
	engine *xorm.Engine
}

func NewSqliteDB(name string, dir string) (*SqliteDB, error) {
	dbPath := path.Join(dir, name)
	cmn.EnsureDir(dbPath, 0700)
	dbFilePath := path.Join(dbPath, name+".db")
	engine, err := xorm.NewEngine("sqlite3", dbFilePath)
	if err != nil {
		return nil, err
	}
	database := &SqliteDB{engine: engine}
	return database, nil
}

func (sd *SqliteDB) Get(key []byte) []byte {
	return nil
}

func (sd *SqliteDB) Set(key, value []byte) {

}

func (sd *SqliteDB) SetSync(key, value []byte) {

}

func (sd *SqliteDB) Delete(key []byte) {

}

func (sd *SqliteDB) DeleteSync(key []byte) {

}

func (sd *SqliteDB) Close() {
	sd.engine.Clone()
}

func (sd *SqliteDB) Print() {
	panic("Unimplemented")
}

func (sd *SqliteDB) Stats() map[string]string {
	return nil
}

func (sd *SqliteDB) Iterator() db.Iterator {
	return nil
}

func (sd *SqliteDB) IteratorPrefix(prefix []byte) db.Iterator {
	return nil
}

func (sd *SqliteDB) NewBatch() db.Batch {
	return &batch{}
}

type reverseIterator struct {
}

//var _ db.Iterator = (*iterator)(nil)

func (rItr *reverseIterator) Valid() bool {
	return true
}

func (rItr *reverseIterator) Domain() (start, end []byte) {

	return nil, nil
}

// Next advances the current reverseIterator
func (rItr *reverseIterator) Next() {

}

func (rItr *reverseIterator) Key() []byte {
	return nil
}

func (rItr *reverseIterator) Value() []byte {
	return nil
}

func (rItr *reverseIterator) Close() {
}

type iterator struct {
}

//var _ db.Iterator = (*iterator)(nil)

func (itr *iterator) Valid() bool {
	return true
}

func (itr *iterator) Domain() (start, end []byte) {

	return nil, nil
}

// Next advances the current iterator
func (itr *iterator) Next() {

}

func (itr *iterator) Key() []byte {

	return nil
}

func (itr *iterator) Value() []byte {

	return nil
}

func (itr *iterator) Close() {

}

type batch struct {
}

var _ db.Batch = (*batch)(nil)

func (bat *batch) Set(key, value []byte) {

}

func (bat *batch) Delete(key []byte) {

}

func (bat *batch) Write() {

}

func (bat *batch) WriteSync() {

}

func (bat *batch) Close() {

}
