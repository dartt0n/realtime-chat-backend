package kv

import (
	"errors"
	"time"

	"github.com/go-redis/redis/v7"
)

type Redis struct {
	client *redis.Client
}

var _ KeyValueStore = (*Redis)(nil)

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

func (r *Redis) Get(key string) (string, error) {
	return r.client.Get(key).Result()
}

func (r *Redis) Set(key string, value string, exp time.Duration) error {
	return r.client.Set(key, value, exp).Err()
}
