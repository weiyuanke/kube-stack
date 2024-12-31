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

func Get(ldb *leveldb.DB, key string) ([]byte, error) {
	data, err := ldb.Get([]byte(key), nil)

	if err != nil && err != leveldb_errors.ErrNotFound {
		return nil, err
	}

	if data == nil {
		return nil, nil
	}

	var cache CacheType
	err = json.Unmarshal(data, &cache)

	if err != nil {
		return nil, nil
	}

	secs := time.Now().Unix()

	if cache.Expires > 0 && cache.Expires <= secs {
		ldb.Delete([]byte(key), nil)
		return nil, nil
	}

	return cache.Data, nil
}

func Set(ldb *leveldb.DB, key string, value string, expires int64) error {
	cache := CacheType{Data: []byte(value), Created: time.Now().Unix(), Expires: 0}

	if expires > 0 {
		cache.Expires = cache.Created + expires
	}

	json_string, err := json.Marshal(cache)

	err = ldb.Put([]byte(key), []byte(json_string), nil)

	return err
}
