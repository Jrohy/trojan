package core

import (
	"os"
	"sync"

	"github.com/syndtr/goleveldb/leveldb"
)

var dbPath = func() string {
	if path := os.Getenv("TROJAN_MANAGER_DB_PATH"); path != "" {
		return path
	}
	return "/var/lib/trojan-manager"
}()
var dbMu sync.Mutex

// GetValue 获取leveldb值
func GetValue(key string) (string, error) {
	dbMu.Lock()
	defer dbMu.Unlock()

	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return "", err
	}
	defer db.Close()
	result, err := db.Get([]byte(key), nil)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// SetValue 设置leveldb值
func SetValue(key string, value string) error {
	dbMu.Lock()
	defer dbMu.Unlock()

	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Put([]byte(key), []byte(value), nil)
}

// DelValue 删除值
func DelValue(key string) error {
	dbMu.Lock()
	defer dbMu.Unlock()

	db, err := leveldb.OpenFile(dbPath, nil)
	if err != nil {
		return err
	}
	defer db.Close()
	return db.Delete([]byte(key), nil)
}
