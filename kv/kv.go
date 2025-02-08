package kv

import "time"

type KeyValueStore interface {
	Set(key, value string, exp time.Duration) error
	Get(key string) (string, error)
	Del(key string) (string, error)
}

var _globalKV KeyValueStore

func GetKV() KeyValueStore {
	return _globalKV
}
