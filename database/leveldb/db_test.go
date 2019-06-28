package leveldb

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func newTempDB(t *testing.T, backend string) (db DB, dbDir string) {
	dirname, err := ioutil.TempDir("", "db_common_test")
	require.Nil(t, err)
	return NewDB("testdb", backend, dirname), dirname
}

func TestDBIteratorSingleKey(t *testing.T) {
	for backend := range backends {
		t.Run(fmt.Sprintf("Backend %s", backend), func(t *testing.T) {
			db, dir := newTempDB(t, backend)
			defer os.RemoveAll(dir)

			db.Set([]byte("1"), []byte("value_1"))
			itr := db.IteratorRange(nil, nil)
			require.Equal(t, []byte(""), itr.Key())
			require.Equal(t, true, itr.Next())
			require.Equal(t, []byte("1"), itr.Key())
		})
	}
}

func TestDBIteratorTwoKeys(t *testing.T) {
	for backend := range backends {
		t.Run(fmt.Sprintf("Backend %s", backend), func(t *testing.T) {
			db, dir := newTempDB(t, backend)
			defer os.RemoveAll(dir)

			db.SetSync([]byte("1"), []byte("value_1"))
			db.SetSync([]byte("2"), []byte("value_1"))

			itr := db.IteratorRange([]byte("1"), nil)

			require.Equal(t, []byte("1"), itr.Key())

			require.Equal(t, true, itr.Next())
			itr = db.IteratorRange([]byte("2"), nil)

			require.Equal(t, false, itr.Next())
		})
	}
}
