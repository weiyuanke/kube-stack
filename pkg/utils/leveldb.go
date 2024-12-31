/* from https://github.com/dwburke/go-leveldb-ttl/blob/master/cache.go
 */
package utils

import (
	"encoding/json"
	"time"

	"github.com/syndtr/goleveldb/leveldb"
	leveldb_errors "github.com/syndtr/goleveldb/leveldb/errors"
)

var ldb *leveldb.DB

type CacheType struct {
	Data    []byte `json:"data"`
	Created int64  `json:"created"`
	Expires int64  `json:"expires"`
}

func InitDB(path string) error {
	var err error = nil
	ldb, err = leveldb.OpenFile(path, nil)
	return err
}

func CloseDB(path string) {
	if ldb != nil {
		ldb.Close()
	}
}

func Get(table, key string) (string, error) {
	key = keyStr(table, key)
	data, err := ldb.Get([]byte(key), nil)
	if err != nil && err != leveldb_errors.ErrNotFound {
		return "", err
	}

	if data == nil {
		return "", nil
	}

	var cache CacheType
	err = json.Unmarshal(data, &cache)
	if err != nil {
		return "", nil
	}

	secs := time.Now().Unix()
	if cache.Expires > 0 && cache.Expires <= secs {
		ldb.Delete([]byte(key), nil)
		return "", nil
	}

	return string(cache.Data), nil
}

func Set(table, key, value string) error {
	key = keyStr(table, key)
	cache := CacheType{Data: []byte(value), Created: time.Now().Unix(), Expires: 0}
	json_string, err := json.Marshal(cache)
	if err != nil {
		return err
	}
	return ldb.Put([]byte(key), []byte(json_string), nil)
}

func keyStr(table, key string) string {
	return table + "__" + key
}
