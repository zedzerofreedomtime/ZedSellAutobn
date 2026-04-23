package database

import (
	"github.com/go-redis/redis/v8"

	"zedsellauto/internal/config"
)

func NewRedis(cfg config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
}
