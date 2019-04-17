package sqlite

import (
	"github.com/vapor/database/db"
)

func init() {
	dbCreator := func(name string, dir string) (db.DB, error) {
		return NewSqliteDB(name, dir)
	}
	db.RegisterDBCreator(db.SqliteDBBackendStr, dbCreator, false)
}

type SqliteDB struct {
}

func NewSqliteDB(name string, dir string) (*SqliteDB, error) {

	database := &SqliteDB{}
	return database, nil
}

//var _ db.DB = (*SqliteDB)(nil)

// Close is a noop currently
func (sd *SqliteDB) Close() {
}

func (sd *SqliteDB) Delete(key []byte) {

}

func (sd *SqliteDB) DeleteSync(key []byte) {

}

func (sd *SqliteDB) Set(key, value []byte) {

}

func (sd *SqliteDB) SetSync(key, value []byte) {

}

func (sd *SqliteDB) Get(key []byte) []byte {
	return nil
}

func (sd *SqliteDB) Has(key []byte) bool {
	return true
}

func (sd *SqliteDB) ReverseIterator(start, end []byte) db.Iterator {
	return nil
}

func (sd *SqliteDB) NewBatch() db.Batch {
	return &batch{}
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
