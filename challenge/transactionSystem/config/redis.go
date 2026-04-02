package config

import (
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	TimeCache     = 5 * time.Minute
	TimeSession   = 15 * time.Minute
	TimeRateLimit = 10 * time.Second
	TimeLock      = 30 * time.Second
)

func ConnectRedis() *redis.Client {
	rdb := redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_HOST") + ":" + os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})
	log.Println(os.Getenv("REDIS_HOST"))
	log.Println(os.Getenv("REDIS_PORT"))
	log.Println(os.Getenv("REDIS_PASSWORD"))
	return rdb
}
