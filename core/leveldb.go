package core

import (
	"github.com/syndtr/goleveldb/leveldb"
)

var dbPath = "/var/lib/trojan-manager"

// GetValue 获取leveldb值
func GetValue(key string) (string, error) {
	db, err := leveldb.OpenFile(dbPath, nil)
	defer db.Close()
	if err != nil {
		return "", err
	}
	result, err := db.Get([]byte(key), nil)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// SetValue 设置leveldb值
func SetValue(key string, value string) error {
	db, err := leveldb.OpenFile(dbPath, nil)
	defer db.Close()
	if err != nil {
		return err
	}
	return db.Put([]byte(key), []byte(value), nil)
}

// DelValue 删除值
func DelValue(key string) error {
	db, err := leveldb.OpenFile(dbPath, nil)
	defer db.Close()
	if err != nil {
		return err
	}
	return db.Delete([]byte(key), nil)
}
