package Storage

import (
	"github.com/go-redis/redis/v7"
	"time"
)

type redisBackend struct {
	client *redis.Client
}

func (r *redisBackend) Set(key, value string, ex int64) error {
	return r.client.Set(key, value, time.Duration(ex) * time.Second).Err()
}

func (r *redisBackend) Get(key string) (string, error) {
	return r.client.Get(key).Result()
}

func (r *redisBackend) HSet(key, field, val string) error {
	return r.client.HSet(key, field, val).Err()
}

func (r *redisBackend) HGetAll(key string) (map[string]string, error)  {
	return r.client.HGetAll(key).Result()
}

func (r *redisBackend) HMSet(key string, fields map[string]interface{}) error {
	return r.client.HMSet(key, fields).Err()
}

func (r *redisBackend) Close() error {
	return r.client.Close()
}