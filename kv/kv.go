package kv

import "time"

// KeyValueStore represents an interface for a key-value storage system
// providing basic operations like Set, Get and Delete
type KeyValueStore interface {
	// Set stores a key-value pair with optional expiration duration
	Set(key, value string, exp time.Duration) error
	// Get retrieves the value associated with the given key
	Get(key string) (string, error)
	// Del removes the key-value pair and returns the deleted key
	Del(key string) (string, error)
}
