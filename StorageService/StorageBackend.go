package StorageService

import (
	"Puzzle/conf"
	"github.com/go-redis/redis/v7"
)

type StorageBackend interface {
	Set(key,value string, ex int64) error
	Get(key string) (string, error)
}

func NewRedisBackend(conf *conf.Config) *redisBackend {
	return &redisBackend{
		client:redis.NewClient(&redis.Options{
			Addr:conf.RedisConf.Host+":"+conf.RedisConf.Port,
			Password:conf.RedisConf.Password,
			DB:0,
		}),
	}
}
