package kv

import (
	"errors"
	"time"

	"github.com/go-redis/redis/v7"
)

// Redis implements the KeyValueStore interface using Redis as the backend
type Redis struct {
	client *redis.Client
}

var _ KeyValueStore = (*Redis)(nil)

// InitRedis initializes a Redis connection with the given address, password and database number.
// Returns an error if the connection cannot be established.
func InitRedis(addr, pwd string, db int) error {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: pwd,
		DB:       db,
	})

	if err := client.Ping().Err(); err != nil {
		return err
	}

	_globalKV = &Redis{client: client}
	return nil
}

// Del deletes a key from Redis. Returns the deleted key if successful,
// or an error if the key doesn't exist or deletion fails.
func (r *Redis) Del(key string) (string, error) {
	count, err := r.client.Del(key).Result()
	if err != nil {
		return "", err
	}

	if count == 0 {
		return "", errors.New("delete cmd failed")
	}

	return key, nil

}

// Get retrieves a value from Redis by key.
// Returns the value and nil if successful, or empty string and error if the key doesn't exist.
func (r *Redis) Get(key string) (string, error) {
	return r.client.Get(key).Result()
}

// Set stores a key-value pair in Redis with an optional expiration duration.
// Returns nil if successful, error otherwise.
func (r *Redis) Set(key string, value string, exp time.Duration) error {
	return r.client.Set(key, value, exp).Err()
}
