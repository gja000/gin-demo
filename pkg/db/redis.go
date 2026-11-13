package db

import (
	"context"
	"time"

	"gin-demo/config"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

func InitRedis() error {
	c := config.Get()
	RDB = redis.NewClient(&redis.Options{
		Addr:     c.RedisAddr,
		Password: c.RedisPassword,
		DB:       c.RedisDB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return RDB.Ping(ctx).Err()
}
